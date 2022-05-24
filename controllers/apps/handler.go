package apps

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/banzaicloud/k8s-objectmatcher/patch"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Handler struct {
	client.Client
	kubernetes.Clientset
	record.EventRecorder
	rest.Config
}

func (handler *Handler) ExecToPods(obj client.Object, containerName, command string) error {
	pods := &corev1.PodList{}
	if err := handler.Client.List(context.TODO(), pods, client.InNamespace(obj.GetNamespace()), client.MatchingLabels(obj.GetLabels())); err != nil {
		return err
	}
	for _, pod := range pods.Items {
		_, stderr, err := handler.execToPod(pod.GetNamespace(), pod.GetName(), containerName, command, nil)
		if err != nil {
			return fmt.Errorf("exec %s container %s in pod %s error: %v", command, containerName, pod.GetName(), err)
		}
		if stderr != "" {
			return fmt.Errorf("exec %s container %s in pod %s stderr: %v", command, containerName, pod.GetName(), stderr)
		}
		str := fmt.Sprintf("Exec %s to container %s successfully", command, containerName)
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
	option := &v1.PodExecOptions{
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
		return "", "", fmt.Errorf("error in Stream: %v", err)
	}

	return stdout.String(), stderr.String(), nil
}

func (handler *Handler) CreateOrUpdate(obj client.Object, postUpdate func() error) error {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(obj.GetObjectKind().GroupVersionKind())
	err := handler.Client.Get(
		context.TODO(),
		types.NamespacedName{
			Name:      obj.GetName(),
			Namespace: obj.GetNamespace(),
		},
		u,
	)

	if err != nil {
		if errors.IsNotFound(err) {
			return handler.doCreate(obj)
		}
		return err
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
		obj.SetAnnotations(annotations)
	}

	opts := []patch.CalculateOption{}
	switch resource := obj.(type) {
	case *appsv1.StatefulSet:
		opts = append(
			opts,
			patch.IgnoreStatusFields(),
			patch.IgnoreVolumeClaimTemplateTypeMetaAndStatus(),
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

	if err := client.NewDryRunClient(handler.Client).Update(context.TODO(), obj); err != nil {
		return err
	}

	patchResult, err := patch.DefaultPatchMaker.Calculate(u, obj, opts...)
	if err != nil {
		handler.EventRecorder.Event(obj, corev1.EventTypeWarning, "Patched", err.Error())
		return err
	}
	if !patchResult.IsEmpty() {
		if err := handler.doUpdate(obj, u); err != nil {
			return err
		}
		if err := postUpdate(); err != nil {
			return err
		}
	}
	return nil
}

func (handler *Handler) doCreate(obj client.Object) error {
	if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(obj); err != nil {
		handler.EventRecorder.Event(obj, corev1.EventTypeWarning, "Patched", err.Error())
		return err
	}
	if err := handler.Client.Create(context.TODO(), obj); err != nil {
		handler.EventRecorder.Event(obj, corev1.EventTypeWarning, "Created", err.Error())
		return err
	}
	handler.EventRecorder.Event(obj, corev1.EventTypeNormal, "Created", "Create resource successfully")
	return nil
}

func (handler *Handler) doUpdate(obj, storageObj client.Object) error {
	if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(obj); err != nil {
		handler.EventRecorder.Event(obj, corev1.EventTypeWarning, "Patched", err.Error())
		return err
	}
	if err := handler.Client.Update(context.TODO(), obj); err != nil {
		handler.EventRecorder.Event(obj, corev1.EventTypeWarning, "Updated", err.Error())
		return err
	}
	handler.EventRecorder.Event(obj, corev1.EventTypeNormal, "Updated", "Update resource successfully")
	return nil
}
