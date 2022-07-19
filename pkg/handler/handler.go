package handler

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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Handler struct {
	client.Client
	kubernetes.Clientset
	rest.Config
}

func (handler *Handler) RequestAPI(obj client.Object, method, username, password, apiPort, path string) (*http.Response, []byte, error) {
	pods := &corev1.PodList{}
	if err := handler.Client.List(
		context.TODO(),
		pods,
		client.InNamespace(obj.GetNamespace()),
		client.MatchingLabels(obj.GetLabels()),
	); err != nil {
		return nil, nil, err
	}

	if len(pods.Items) == 0 {
		return nil, nil, fmt.Errorf("not found pods")
	}

	var podName string
findPod:
	for _, pod := range pods.Items {
		for _, status := range pod.Status.ContainerStatuses {
			if status.Name == "emqx" && status.Ready {
				podName = pod.Name
				break findPod
			}
		}
	}

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

func (handler *Handler) CreateOrUpdateList(instance client.Object, scheme *runtime.Scheme, resources []client.Object, postFun func(client.Object) error) error {
	for _, resource := range resources {
		if err := ctrl.SetControllerReference(instance, resource, scheme); err != nil {
			return err
		}

		nothing := func(client.Object) error { return nil }
		err := handler.CreateOrUpdate(resource, nothing)
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

	for i, container := range sts.Spec.Template.Spec.Containers {
		if container.Name != "emqx" && container.Name != "reloader" {
			sts.Spec.Template.Spec.Containers = append(sts.Spec.Template.Spec.Containers[:i], sts.Spec.Template.Spec.Containers[i+1:]...)
		}
	}

	return json.Marshal(sts)
}
