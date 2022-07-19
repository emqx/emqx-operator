package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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
)

const (
	ManageContainersAnnotation = "apps.emqx.io/manage-containers"
	EmqxContainerName          = "emqx"
)

type Handler struct {
	client.Client
	kubernetes.Clientset
	rest.Config
}

func (handler *Handler) RequestAPI(obj client.Object, method, username, password, apiPort, path string) (*http.Response, []byte, error) {
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
		return nil, nil, fmt.Errorf("not found pods")
	}

	podName := findReadyEmqxPod(podList)
	if podName == "" {
		return nil, nil, fmt.Errorf("pods not ready")
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
			Clientset:    handler.Clientset,
			Config:       &handler.Config,
			ReadyChannel: readyChan,
			StopChannel:  stopChan,
		},
	}

	return apiClient.Do(method, path)
}

func (handler *Handler) CreateOrUpdateList(instance client.Object, scheme *runtime.Scheme, resources []client.Object, postFun func(client.Object) error) error {
	for _, resource := range resources {
		if err := ctrl.SetControllerReference(instance, resource, scheme); err != nil {
			return err
		}
		err := handler.CreateOrUpdate(resource, postFun)
		if err != nil {
			return err
		}
	}
	return nil
}

func (handler *Handler) CreateOrUpdate(obj client.Object, postFun func(client.Object) error) error {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(obj.GetObjectKind().GroupVersionKind())
	err := handler.Client.Get(context.TODO(), client.ObjectKeyFromObject(obj), u)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return handler.Create(obj, postFun)
		}
		return fmt.Errorf("failed to get %s %s: %v", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetName(), err)
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
		_ = client.NewDryRunClient(handler.Client).Update(context.TODO(), obj)
		opts = append(
			opts,
			patch.IgnoreStatusFields(),
			patch.IgnoreVolumeClaimTemplateTypeMetaAndStatus(),
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

	patchResult, err := patch.DefaultPatchMaker.Calculate(u, obj, opts...)
	if err != nil {
		return fmt.Errorf("failed to calculate patch for %s %s: %v", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetName(), err)
	}
	if !patchResult.IsEmpty() {
		return handler.Update(obj, postFun)
	}
	return nil
}

func (handler *Handler) Create(obj client.Object, postCreated func(client.Object) error) error {
	switch obj.(type) {
	case *appsv1beta3.EmqxBroker:
	case *appsv1beta3.EmqxEnterprise:
	default:
		if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(obj); err != nil {
			return fmt.Errorf("failed to set last applied annotation for %s %s: %v", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetName(), err)
		}
	}

	if err := handler.Client.Create(context.TODO(), obj); err != nil {
		return fmt.Errorf("failed to create %s %s: %v", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetName(), err)
	}
	return postCreated(obj)
}

func (handler *Handler) Update(obj client.Object, postUpdated func(client.Object) error) error {
	switch obj.(type) {
	case *appsv1beta3.EmqxBroker:
	case *appsv1beta3.EmqxEnterprise:
	default:
		if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(obj); err != nil {
			return fmt.Errorf("failed to set last applied annotation for %s %s: %v", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetName(), err)
		}
	}

	if err := handler.Client.Update(context.TODO(), obj); err != nil {
		return fmt.Errorf("failed to update %s %s: %v", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetName(), err)
	}
	return postUpdated(obj)
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
	sts := &appsv1.StatefulSet{}
	_ = json.Unmarshal(obj, sts)
	containerNames := sts.Annotations[ManageContainersAnnotation]
	containers := []corev1.Container{}
	for _, container := range sts.Spec.Template.Spec.Containers {
		if strings.Contains(containerNames, container.Name) {
			containers = append(containers, container)
		}
	}
	sts.Spec.Template.Spec.Containers = containers
	return json.Marshal(sts)
}

func findReadyEmqxPod(pods *corev1.PodList) string {
	for _, pod := range pods.Items {
		for _, status := range pod.Status.ContainerStatuses {
			if status.Name == EmqxContainerName && status.Ready {
				return pod.Name
			}
		}
	}
	return ""
}
