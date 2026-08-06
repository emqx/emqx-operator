/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"errors"
	"strings"
	"time"

	emperror "emperror.dev/errors"
	crd "github.com/emqx/emqx-operator/api/v3beta1"
	config "github.com/emqx/emqx-operator/internal/controller/config"
	emqxapi "github.com/emqx/emqx-operator/internal/emqx/api"
	req "github.com/emqx/emqx-operator/internal/requester"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const emqxAPIKeyFinalizer = "apps.emqx.io/emqxapikey-finalizer"

const (
	apiKeySecretKey    = "apiKey"
	apiSecretSecretKey = "apiSecret"
)

// EMQXAPIKeyReconciler reconciles an EMQXAPIKey object.
type EMQXAPIKeyReconciler struct {
	Client        client.Client
	Scheme        *runtime.Scheme
	EventRecorder record.EventRecorder
}

func NewEMQXAPIKeyReconciler(mgr manager.Manager) *EMQXAPIKeyReconciler {
	return &EMQXAPIKeyReconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		EventRecorder: mgr.GetEventRecorderFor("emqxapikey-controller"),
	}
}

// +kubebuilder:rbac:groups=apps.emqx.io,resources=emqxapikeys,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.emqx.io,resources=emqxapikeys/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps.emqx.io,resources=emqxapikeys/finalizers,verbs=update

type apiKeyReconcileRound struct {
	ctx       context.Context
	log       logr.Logger
	emqx      *crd.EMQX
	state     *reconcileState
	requester apiRequester

	isDirty, isDirtyStatus bool
}

// Instantiate default API requester for a core node.
// Picks oldest core node that is considered ready: up and running, not evacuating.
func (r *apiKeyReconcileRound) oldestCoreRequester() req.RequesterInterface {
	return r.requester.forOldestCore(r.state, &podConditionFilter{cond: corev1.ContainersReady})
}

func (r *apiKeyReconcileRound) dirty(flag bool) {
	r.isDirty = r.isDirty || flag
}

func (r *apiKeyReconcileRound) dirtyStatus(flag bool) {
	r.isDirtyStatus = r.isDirtyStatus || flag
}

var (
	emqxNotFound       = reconcileBlocked{"EMQXNotFound", nil}
	emqxAPIUnavailable = reconcileBlocked{"EMQXAPIUnavailable", nil}
)

type reconcileBlocked struct {
	reason string
	cause  error
}

func (e *reconcileBlocked) withCause(err error) *reconcileBlocked {
	e.cause = err
	return e
}

func (e reconcileBlocked) Error() string {
	return e.cause.Error()
}

func (e reconcileBlocked) Is(target error) bool {
	if target, ok := target.(reconcileBlocked); ok {
		return e.reason == target.reason
	}
	return false
}

// Reconcile synchronizes an EMQXAPIKey object with the referenced EMQX cluster.
func (r *EMQXAPIKeyReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	round := &apiKeyReconcileRound{
		ctx: ctx,
		log: log.FromContext(ctx),
	}

	apiKey := &crd.EMQXAPIKey{}
	if err := r.Client.Get(ctx, request.NamespacedName, apiKey); err != nil {
		if k8sErrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !apiKey.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(round, apiKey)
	}

	return r.reconcileSync(round, apiKey)
}

// SetupWithManager sets up the controller with the Manager.
func (r *EMQXAPIKeyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&crd.EMQXAPIKey{}).
		Named("emqxapikey").
		Complete(r)
}

func (r *EMQXAPIKeyReconciler) findEMQXInstance(round *apiKeyReconcileRound, apiKey *crd.EMQXAPIKey) error {
	emqx := &crd.EMQX{}
	err := r.Client.Get(round.ctx, client.ObjectKey{
		Name:      apiKey.Spec.EMQXRef.Name,
		Namespace: apiKey.Namespace,
	}, emqx)
	if k8sErrors.IsNotFound(err) {
		return emqxNotFound.withCause(emperror.Wrapf(err, "missing EMQX CR instance"))
	}
	if err != nil {
		return emperror.Wrap(err, "failed to get referenced EMQX CR")
	}
	round.emqx = emqx
	return nil
}

func (r *EMQXAPIKeyReconciler) setupAPIRequester(round *apiKeyReconcileRound) error {
	var err error

	round.state, err = loadReconcileState(round.ctx, r.Client, round.emqx)
	if err != nil {
		return emperror.Wrap(err, "failed to load EMQX reconcile state")
	}
	bootstrapAPIKey, err := getBootstrapAPIKey(round.ctx, r.Client, round.emqx)
	if err != nil {
		return emperror.Wrap(err, "failed to get bootstrap API key")
	}

	conf, err := config.EMQXConfigWithDefaults(applicableConfig(round.emqx))
	if err != nil {
		return emqxAPIUnavailable.withCause(emperror.Wrap(err, "failed to parse EMQX config"))
	}
	requester, err := newAPIRequesterBuilder(conf, bootstrapAPIKey)
	if err != nil {
		return emqxAPIUnavailable.withCause(emperror.Wrap(err, "failed to create API requester"))
	}

	round.requester = requester
	return nil
}

func (r *EMQXAPIKeyReconciler) reconcileDelete(
	round *apiKeyReconcileRound,
	apiKey *crd.EMQXAPIKey,
) (ctrl.Result, error) {
	// No finalizer is set: nothing to do.
	if !controllerutil.ContainsFinalizer(apiKey, emqxAPIKeyFinalizer) {
		return ctrl.Result{}, nil
	}

	// Find respective EMQX CR instance:
	err := r.findEMQXInstance(round, apiKey)

	// Owning EMQX CR is gone yet finalizer is set: just remove finalizer.
	if errors.Is(err, emqxNotFound) {
		return r.reconcileRemoveFinalizer(round, apiKey)
	}

	// Error getting EMQX CR:
	if err != nil {
		return r.processError(round, apiKey, err)
	}

	// Setup API requester:
	err = r.setupAPIRequester(round)
	if err != nil {
		return r.processError(round, apiKey, err)
	}

	api := round.oldestCoreRequester()
	if api == nil {
		return ctrl.Result{Requeue: true}, nil
	}

	err = emqxapi.DeleteAPIKey(api, apiKey.Name)
	if err != nil && !emperror.Is(err, emqxapi.ErrorNotFound) {
		// TODO: Specify error?
		return r.markAPIError(round, apiKey, err)
	}

	return r.reconcileRemoveFinalizer(round, apiKey)
}

func (r *EMQXAPIKeyReconciler) reconcileRemoveFinalizer(
	round *apiKeyReconcileRound,
	apiKey *crd.EMQXAPIKey,
) (ctrl.Result, error) {
	controllerutil.RemoveFinalizer(apiKey, emqxAPIKeyFinalizer)
	err := r.Client.Update(round.ctx, apiKey)
	return ctrl.Result{}, emperror.Wrap(err, "failed to remove finalizer")
}

func (r *EMQXAPIKeyReconciler) reconcileSync(
	round *apiKeyReconcileRound,
	apiKey *crd.EMQXAPIKey,
) (ctrl.Result, error) {
	// Find respective EMQX CR instance:
	err := r.findEMQXInstance(round, apiKey)
	if err != nil {
		return r.processError(round, apiKey, err)
	}

	// Find managed secret:
	secret := &corev1.Secret{}
	secretName := types.NamespacedName{Namespace: apiKey.Namespace, Name: apiKey.Spec.SecretRef.Name}
	err = r.Client.Get(round.ctx, secretName, secret)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			secret = nil
		} else {
			return r.processError(round, apiKey, err)
		}
	}

	// Make EMQX CR own this API Key:
	changed := apiKey.AttachOwnerReference(round.emqx, r.Scheme)
	round.dirtyStatus(changed)

	// Set up a finalizer:
	changed = controllerutil.AddFinalizer(apiKey, emqxAPIKeyFinalizer)
	round.dirty(changed)

	// Set up initial conditions:
	changed = apiKey.SetCondition(crd.EMQXAPIKeyIssuing, metav1.ConditionFalse, "NotReady", "Syncing API key")
	round.dirtyStatus(changed)

	// Setup API requester:
	err = r.setupAPIRequester(round)
	if err != nil {
		return r.processError(round, apiKey, err)
	}

	// Postpone: no usable API api yet.
	api := round.oldestCoreRequester()
	if api == nil {
		return ctrl.Result{Requeue: true}, nil
	}

	actual, err := emqxapi.GetAPIKey(api, apiKey.Name)

	if err != nil && emperror.Is(err, emqxapi.ErrorNotFound) {
		if secret != nil {
			if err := r.Client.Delete(round.ctx, secret); err != nil {
				return r.processError(round, apiKey, err)
			}
		}
		return r.reconcileCreate(round, api, apiKey)
	}

	if err != nil {
		return r.markAPIError(round, apiKey, emperror.Wrap(err, "read request failed"))
	}

	if !apiKeySecretMatches(secret, actual) {
		err = emqxapi.DeleteAPIKey(api, apiKey.Name)
		if err != nil && !emperror.Is(err, emqxapi.ErrorNotFound) {
			return r.markAPIError(round, apiKey, emperror.Wrap(err, "delete request failed"))
		}
		return r.reconcileCreate(round, api, apiKey)
	}

	if !apiKeyDetailsMatches(apiKey, actual) {
		actual, err = emqxapi.UpdateAPIKey(api, apiKey.Name, apiKeyRequest(apiKey))
		if err == nil {
			r.event(apiKey, corev1.EventTypeNormal, "Updated", "Updated EMQX API key")
		} else {
			return r.markAPIError(round, apiKey, emperror.Wrap(err, "update request failed"))
		}
	}

	return r.markReady(round, apiKey, actual)
}

func (r *EMQXAPIKeyReconciler) reconcileCreate(
	round *apiKeyReconcileRound,
	requester req.RequesterInterface,
	apiKey *crd.EMQXAPIKey,
) (ctrl.Result, error) {
	created, err := emqxapi.CreateAPIKey(requester, apiKeyRequest(apiKey))
	if err != nil {
		return r.markAPIError(round, apiKey, emperror.Wrap(err, "create request failed"))
	}

	err = r.preserveCredentials(round, apiKey, created)
	if err != nil {
		return r.processError(round, apiKey, err)
	}

	r.event(apiKey, corev1.EventTypeNormal, "Created", "Created EMQX API key")

	return r.markReady(round, apiKey, created)
}

func (r *EMQXAPIKeyReconciler) preserveCredentials(
	round *apiKeyReconcileRound,
	apiKey *crd.EMQXAPIKey,
	created *emqxapi.APIKey,
) error {
	if created.APIKey == "" || created.APISecret == "" {
		return emperror.New("create response does not contain API key credentials")
	}
	secret := r.credentialSecret(apiKey, created)
	err := r.Client.Create(round.ctx, secret)
	if err != nil {
		return emperror.Wrap(err, "failed to create API key secret")
	}
	return nil
}

func (r *EMQXAPIKeyReconciler) credentialSecret(apiKey *crd.EMQXAPIKey, created *emqxapi.APIKey) *corev1.Secret {
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: apiKey.Namespace,
			Name:      apiKey.Spec.SecretRef.Name,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			apiKeySecretKey:    []byte(created.APIKey),
			apiSecretSecretKey: []byte(created.APISecret),
		},
	}
	_ = ctrl.SetControllerReference(apiKey, secret, r.Scheme)
	return secret
}

func (r *EMQXAPIKeyReconciler) processError(
	round *apiKeyReconcileRound,
	apiKey *crd.EMQXAPIKey,
	err error,
) (ctrl.Result, error) {
	for _, blocked := range []reconcileBlocked{emqxNotFound, emqxAPIUnavailable} {
		if errors.Is(err, blocked) {
			var reconcileErr *reconcileBlocked
			if errors.As(err, &reconcileErr) {
				return r.markUnreconcilable(round, apiKey, reconcileErr.reason, reconcileErr.cause)
			}
			return r.markUnreconcilable(round, apiKey, blocked.reason, err)
		}
	}
	return ctrl.Result{}, err
}

func (r *EMQXAPIKeyReconciler) markUnreconcilable(
	round *apiKeyReconcileRound,
	apiKey *crd.EMQXAPIKey,
	reason string,
	err error,
) (ctrl.Result, error) {
	changed := apiKey.SetCondition(crd.EMQXAPIKeyIssuing, metav1.ConditionFalse, reason, err.Error())
	changed = apiKey.SetCondition(crd.EMQXAPIKeyReady, metav1.ConditionFalse, reason, err.Error()) || changed
	round.dirtyStatus(changed)
	if changed {
		r.event(apiKey, corev1.EventTypeWarning, reason, err.Error())
	}
	return r.syncDirty(round, apiKey)
}

func (r *EMQXAPIKeyReconciler) markAPIError(
	round *apiKeyReconcileRound,
	apiKey *crd.EMQXAPIKey,
	err error,
) (ctrl.Result, error) {
	reason := "EMQXAPIError"
	changed := apiKey.SetCondition(crd.EMQXAPIKeyReady, metav1.ConditionFalse, reason, err.Error())
	if changed {
		r.event(apiKey, corev1.EventTypeWarning, reason, err.Error())
	}
	return r.syncDirty(round, apiKey)
}

func (r *EMQXAPIKeyReconciler) markReady(
	round *apiKeyReconcileRound,
	apiKey *crd.EMQXAPIKey,
	remote *emqxapi.APIKey,
) (ctrl.Result, error) {
	changed := apiKey.SetCondition(crd.EMQXAPIKeyReady, metav1.ConditionTrue, "Ready", "API key is ready")
	changed = apiKey.SetCondition(crd.EMQXAPIKeyIssuing, metav1.ConditionFalse, "Ready", "API key is synced with EMQX") || changed
	if remote != nil && remote.Expired {
		changed = apiKey.SetCondition(crd.EMQXAPIKeyExpired, metav1.ConditionTrue, "Expired",
			"EMQX considers the API key is expired") || changed
	} else {
		changed = apiKey.SetCondition(crd.EMQXAPIKeyExpired, metav1.ConditionFalse, "NotExpired",
			"EMQX considers the API key is not expired") || changed
	}
	round.dirtyStatus(changed)
	return r.syncDirty(round, apiKey)
}

func (r *EMQXAPIKeyReconciler) syncDirty(
	round *apiKeyReconcileRound,
	apiKey *crd.EMQXAPIKey,
) (ctrl.Result, error) {
	if round.isDirty {
		if err := r.Client.Update(round.ctx, apiKey); err != nil {
			return ctrl.Result{}, err
		}
		if !round.isDirtyStatus {
			return ctrl.Result{}, nil
		}
	}
	if round.isDirtyStatus {
		err := r.Client.Status().Update(round.ctx, apiKey)
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *EMQXAPIKeyReconciler) event(apiKey *crd.EMQXAPIKey, eventType, reason, message string) {
	if r.EventRecorder != nil {
		r.EventRecorder.Event(apiKey, eventType, reason, message)
	}
}

func apiKeyRequest(apiKey *crd.EMQXAPIKey) emqxapi.APIKeyRequest {
	var expiresAt *time.Time
	if apiKey.Spec.ExpiresAt != nil {
		t := apiKey.Spec.ExpiresAt.Time
		expiresAt = &t
	}
	return emqxapi.APIKeyRequest{
		Name:        apiKey.Name,
		Description: apiKey.Spec.Description,
		Role:        apiKeyRole(apiKey),
		ExpiresAt:   expiresAt,
	}
}

func apiKeyRole(apiKey *crd.EMQXAPIKey) string {
	return strings.ToLower(string(apiKey.Spec.Role))
}

func apiKeySecretMatches(secret *corev1.Secret, actual *emqxapi.APIKey) bool {
	if actual == nil || actual.APIKey == "" {
		return false
	}
	return string(secret.Data[apiKeySecretKey]) == actual.APIKey
}

func apiKeyDetailsMatches(apiKey *crd.EMQXAPIKey, actual *emqxapi.APIKey) bool {
	if actual == nil {
		return false
	}
	if actual.Description != apiKey.Spec.Description ||
		!actual.Enabled ||
		strings.ToLower(actual.Role) != apiKeyRole(apiKey) {
		return false
	}

	if apiKey.Spec.ExpiresAt == nil {
		return actual.ExpiredAt == "" || actual.ExpiredAt == "infinity"
	}

	actualExpiresAt, err := time.Parse(time.RFC3339, actual.ExpiredAt)
	if err != nil {
		return false
	}
	if actualExpiresAt.Unix() != apiKey.Spec.ExpiresAt.Unix() {
		return false
	}

	return true
}
