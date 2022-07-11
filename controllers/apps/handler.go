package apps

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	emperror "emperror.dev/errors"
	"github.com/banzaicloud/k8s-objectmatcher/patch"
	appsv1beta3 "github.com/emqx/emqx-operator/apis/apps/v1beta3"
	apiClient "github.com/emqx/emqx-operator/pkg/apiclient"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	emqxContainerName      = "emqx"
	reloaderContainerName  = "reloader"
	reloaderContainerImage = "emqx/emqx-operator-reloader:0.0.1"

	// managermentUsername = "admin"
	// managermentPassword = "public"
	// managermentApiPort = "8081"

)

type Handler struct {
	client.Client
	kubernetes.Clientset
	record.EventRecorder
	rest.Config
}

func (handler *Handler) getManagementField(obj appsv1beta3.Emqx) (username, password, apiPort string) {
	username = "admin"
	password = "public"
	apiPort = "8081"

	pluginsList := &appsv1beta3.EmqxPluginList{}
	_ = handler.Client.List(context.TODO(), pluginsList, client.InNamespace(obj.GetNamespace()))

	for _, plugin := range pluginsList.Items {
		selector, _ := labels.ValidatedSelectorFromSet(plugin.Spec.Selector)
		if selector.Empty() || !selector.Matches(labels.Set(obj.GetLabels())) {
			continue
		}
		if plugin.Spec.PluginName == "emqx_management" {
			if _, ok := plugin.Spec.Config["management.listener.http"]; ok {
				apiPort = plugin.Spec.Config["management.listener.http"]
			}
			if _, ok := plugin.Spec.Config["management.default_application.id"]; ok {
				username = plugin.Spec.Config["management.default_application.id"]
			}
			if _, ok := plugin.Spec.Config["management.default_application.secret"]; ok {
				password = plugin.Spec.Config["management.default_application.secret"]
			}
		}
	}

	return
}

func (handler *Handler) requestAPI(obj appsv1beta3.Emqx, method, path string) (*http.Response, []byte, error) {
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

	configMap := &corev1.ConfigMap{}
	if err := handler.Get(
		context.TODO(),
		client.ObjectKey{
			Name:      fmt.Sprintf("%s-%s", obj.GetName(), "plugins-config"),
			Namespace: obj.GetNamespace(),
		},
		configMap,
	); err != nil {
		return nil, nil, err
	}

	username, password, apiPort := handler.getManagementField(obj)

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

func (handler *Handler) ExecToPods(obj client.Object, containerName, command string) error {
	pods := &corev1.PodList{}
	if err := handler.Client.List(context.TODO(), pods, client.InNamespace(obj.GetNamespace()), client.MatchingLabels(obj.GetLabels())); err != nil {
		return err
	}
	for _, pod := range pods.Items {
		if pod.Status.Phase != corev1.PodRunning {
			return fmt.Errorf("pod %s is not running", pod.Name)
		}
		for _, containerStatus := range pod.Status.ContainerStatuses {
			if containerStatus.Name == containerName {
				if !containerStatus.Ready {
					return fmt.Errorf("container %s is not ready", containerName)
				}
			}
		}

		stdout, stderr, err := handler.execToPod(pod.GetNamespace(), pod.GetName(), containerName, command, nil)
		if err != nil {
			return fmt.Errorf("exec %s container %s in pod %s failed, stdout: %v, stderr: %v, error: %v", command, containerName, pod.GetName(), stdout, stderr, err)
		}
		str := fmt.Sprintf("exec %s to container %s successfully", command, containerName)
		handler.EventRecorder.Event(obj, corev1.EventTypeNormal, "Exec", str)
	}
	return nil
}

func (handler *Handler) execToPod(namespace, podName, containerName, command string, stdin io.Reader) (string, string, error) {
	cmd := []string{
		"sh",
		"-c",
		command,
	}

	req := handler.Clientset.CoreV1().RESTClient().Post().Resource("pods").Name(podName).
		Namespace(namespace).SubResource("exec")
	option := &corev1.PodExecOptions{
		// Command:   strings.Fields(command),
		Command:   cmd,
		Container: containerName,
		Stdin:     stdin != nil,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}
	req.VersionedParams(
		option,
		scheme.ParameterCodec,
	)
	exec, err := remotecommand.NewSPDYExecutor(&handler.Config, "POST", req.URL())
	if err != nil {
		return "", "", fmt.Errorf("error while creating Executor: %v", err)
	}

	var stdout, stderr bytes.Buffer
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})
	if err != nil {
		return stdout.String(), stderr.String(), fmt.Errorf("error in Stream: %v", err)
	}

	return stdout.String(), stderr.String(), nil
}

func (handler *Handler) CreateOrUpdateList(instance client.Object, resources []client.Object, postFun func(client.Object) error) error {
	ownerRef := metav1.NewControllerRef(instance, instance.GetObjectKind().GroupVersionKind())
	for _, resource := range resources {
		addOwnerRefToObject(resource, *ownerRef)
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
			return handler.doCreate(obj, postFun)
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
		return handler.doUpdate(obj, postFun)
	}
	return nil
}

func (handler *Handler) doCreate(obj client.Object, postCreated func(client.Object) error) error {
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
	handler.EventRecorder.Event(obj, corev1.EventTypeNormal, "Created", "Create resource successfully")
	return postCreated(obj)
}

func (handler *Handler) doUpdate(obj client.Object, postUpdated func(client.Object) error) error {
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
	handler.EventRecorder.Event(obj, corev1.EventTypeNormal, "Updated", "Update resource successfully")
	return postUpdated(obj)
}

func IgnoreOtherContainers() patch.CalculateOption {
	return func(current, modified []byte) ([]byte, []byte, error) {
		current, err := selectEmqxContainer(current)
		if err != nil {
			return []byte{}, []byte{}, emperror.Wrap(err, "could not delete the field from current byte sequence")
		}

		modified, err = selectEmqxContainer(modified)
		if err != nil {
			return []byte{}, []byte{}, emperror.Wrap(err, "could not delete the field from modified byte sequence")
		}

		return current, modified, nil
	}
}

func selectEmqxContainer(obj []byte) ([]byte, error) {
	sts := &appsv1.StatefulSet{}
	_ = json.Unmarshal(obj, sts)

	containers := []corev1.Container{}
	for _, container := range sts.Spec.Template.Spec.Containers {
		if _, ok := sts.Annotations[container.Name]; ok {
			containers = append(containers, container)
		}
	}
	sts.Spec.Template.Spec.Containers = append(sts.Spec.Template.Spec.Containers, containers...)
	return json.Marshal(sts)

	// containers := gjson.GetBytes(obj, `spec.template.spec.containers.#(name=="emqx")#`)
	// newObj, err := sjson.SetBytes(obj, "spec.template.spec.containers", containers.String())
	// if err != nil {
	// 	return []byte{}, emperror.Wrap(err, "could not set byte sequence")
	// }
	// return newObj, nil
}

func findReadyEmqxPod(pods *corev1.PodList) string {
	for _, pod := range pods.Items {
		for _, status := range pod.Status.ContainerStatuses {
			if status.Name == emqxContainerName && status.Ready {
				return pod.Name
			}
		}
	}
	return ""
}
