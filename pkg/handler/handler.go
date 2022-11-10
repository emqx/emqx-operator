package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	json "github.com/json-iterator/go"

	emperror "emperror.dev/errors"
	"github.com/banzaicloud/k8s-objectmatcher/patch"
	appsv1beta3 "github.com/emqx/emqx-operator/apis/apps/v1beta3"
	apiClient "github.com/emqx/emqx-operator/pkg/apiclient"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const (
	ManageContainersAnnotation = "apps.emqx.io/manage-containers"
	LastAppliedAnnotation      = "apps.emqx.io/last-applied"
)

type Patcher struct {
	*patch.Annotator
	patch.Maker
}

type Handler struct {
	Patcher   *Patcher
	Client    client.Client
	clientset *kubernetes.Clientset
	config    *rest.Config
}

func newPatcher() *Patcher {
	var patcher *Patcher = new(Patcher)
	patcher.Annotator = patch.NewAnnotator(LastAppliedAnnotation)
	patcher.Maker = patch.NewPatchMaker(
		patcher.Annotator,
		&patch.K8sStrategicMergePatcher{},
		&patch.BaseJSONMergePatcher{},
	)
	return patcher
}

func NewHandler(mgr manager.Manager) *Handler {
	return &Handler{
		Patcher:   newPatcher(),
		Client:    mgr.GetClient(),
		clientset: kubernetes.NewForConfigOrDie(mgr.GetConfig()),
		config:    mgr.GetConfig(),
	}
}

func (handler *Handler) RequestAPI(obj client.Object, containerName string, method, username, password, apiPort, path string) (*http.Response, []byte, error) {
	podList := &corev1.PodList{}
	if err := handler.Client.List(
		context.TODO(),
		podList,
		client.InNamespace(obj.GetNamespace()),
		client.MatchingLabels(obj.GetLabels()),
	); err != nil {
		return nil, nil, err
	}

	if len(podList.Items) == 0 {
		return nil, nil, emperror.Errorf("not found pods")
	}

	podName := findReadyEmqxPod(podList, containerName)
	if podName == "" {
		return nil, nil, emperror.Errorf("pods not ready")
	}

	stopChan, readyChan := make(chan struct{}, 1), make(chan struct{}, 1)

	apiClient := apiClient.APIClient{
		Username: username,
		Password: password,
		PortForwardOptions: apiClient.PortForwardOptions{
			Namespace: obj.GetNamespace(),
			PodName:   podName,
			PodPorts: []string{
				fmt.Sprintf(":%s", apiPort),
			},
			Clientset:    handler.clientset,
			Config:       handler.config,
			ReadyChannel: readyChan,
			StopChannel:  stopChan,
		},
	}

	return apiClient.Do(method, path)
}

func (handler *Handler) CreateOrUpdateList(instance client.Object, scheme *runtime.Scheme, resources []client.Object) error {
	for _, resource := range resources {
		if err := ctrl.SetControllerReference(instance, resource, scheme); err != nil {
			return err
		}
		err := handler.CreateOrUpdate(resource)
		if err != nil {
			return err
		}

	}
	return nil
}

func (handler *Handler) CreateOrUpdate(obj client.Object) error {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(obj.GetObjectKind().GroupVersionKind())
	err := handler.Client.Get(context.TODO(), client.ObjectKeyFromObject(obj), u)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return handler.Create(obj)
		}
		return emperror.Wrapf(err, "failed to get %s %s", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetName())
	}

	obj.SetResourceVersion(u.GetResourceVersion())
	obj.SetCreationTimestamp(u.GetCreationTimestamp())
	obj.SetManagedFields(u.GetManagedFields())
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	for key, value := range u.GetAnnotations() {
		if _, present := annotations[key]; !present {
			annotations[key] = value
		}
	}
	obj.SetAnnotations(annotations)

	opts := []patch.CalculateOption{}
	switch resource := obj.(type) {
	case *appsv1.StatefulSet:
		opts = append(
			opts,
			patch.IgnoreStatusFields(),
			patch.IgnoreVolumeClaimTemplateTypeMetaAndStatus(),
			IgnoreOtherContainers(),
		)
	case *appsv1.Deployment:
		opts = append(
			opts,
			IgnoreOtherContainers(),
		)
	case *corev1.Service:
		storageResource := &corev1.Service{}
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.UnstructuredContent(), storageResource)
		if err != nil {
			return err
		}
		// Required fields when updating service in k8s 1.21
		if storageResource.Spec.ClusterIP != "" {
			resource.Spec.ClusterIP = storageResource.Spec.ClusterIP
		}
		obj = resource
	}

	patchResult, err := handler.Patcher.Calculate(u, obj, opts...)
	if err != nil {
		return emperror.Wrapf(err, "failed to calculate patch for %s %s", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetName())
	}
	if !patchResult.IsEmpty() {
		return handler.Update(obj)
	}
	return nil
}

func (handler *Handler) Create(obj client.Object) error {
	switch obj.(type) {
	case *appsv1beta3.EmqxBroker:
	case *appsv1beta3.EmqxEnterprise:
	default:
		if err := handler.Patcher.SetLastAppliedAnnotation(obj); err != nil {
			return emperror.Wrapf(err, "failed to set last applied annotation for %s %s", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetName())
		}
	}

	if err := handler.Client.Create(context.TODO(), obj); err != nil {
		return emperror.Wrapf(err, "failed to create %s %s", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetName())
	}
	return nil
}

func (handler *Handler) Update(obj client.Object) error {
	if err := handler.Patcher.SetLastAppliedAnnotation(obj); err != nil {
		return emperror.Wrapf(err, "failed to set last applied annotation for %s %s", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetName())
	}

	if err := handler.Client.Update(context.TODO(), obj); err != nil {
		return emperror.Wrapf(err, "failed to update %s %s", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetName())
	}
	return nil
}

func (handler *Handler) GetBootstrapUser(instance client.Object) (username, password string, err error) {
	return "admin", "public", nil
}

func IgnoreOtherContainers() patch.CalculateOption {
	return func(current, modified []byte) ([]byte, []byte, error) {
		current, err := selectManagerContainer(current)
		if err != nil {
			return []byte{}, []byte{}, emperror.Wrap(err, "could not delete the field from current byte sequence")
		}

		modified, err = selectManagerContainer(modified)
		if err != nil {
			return []byte{}, []byte{}, emperror.Wrap(err, "could not delete the field from modified byte sequence")
		}

		return current, modified, nil
	}
}

func selectManagerContainer(obj []byte) ([]byte, error) {
	var podTemplate corev1.PodTemplateSpec
	var objMap map[string]interface{}
	err := json.Unmarshal(obj, &objMap)
	if err != nil {
		return nil, emperror.Wrap(err, "could not unmarshal json")
	}

	kind := objMap["kind"].(string)
	switch kind {
	case "Deployment":
		deploy := &appsv1.Deployment{}
		err := json.Unmarshal(obj, deploy)
		if err != nil {
			return nil, emperror.Wrap(err, "could not unmarshal json")
		}
		podTemplate = deploy.Spec.Template
	case "StatefulSet":
		sts := &appsv1.StatefulSet{}
		err := json.Unmarshal(obj, sts)
		if err != nil {
			return nil, emperror.Wrap(err, "could not unmarshal json")
		}
		podTemplate = sts.Spec.Template
	default:
		return nil, emperror.Wrapf(err, "unsupported kind: %s", kind)
	}

	containerNames := podTemplate.Annotations[ManageContainersAnnotation]
	containers := []corev1.Container{}
	for _, container := range podTemplate.Spec.Containers {
		if strings.Contains(containerNames, container.Name) {
			containers = append(containers, container)
		}
	}
	podTemplate.Spec.Containers = containers
	objMap["spec"].(map[string]interface{})["template"] = podTemplate
	return json.ConfigCompatibleWithStandardLibrary.Marshal(objMap)
}

func findReadyEmqxPod(pods *corev1.PodList, containerName string) string {
	for _, pod := range pods.Items {
		for _, status := range pod.Status.ContainerStatuses {
			if status.Name == containerName && status.Ready {
				return pod.Name
			}
		}
	}
	return ""
}
