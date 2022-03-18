package apps

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/banzaicloud/k8s-objectmatcher/patch"
	"github.com/emqx/emqx-operator/apis/apps/v1beta3"
	"github.com/emqx/emqx-operator/pkg/service"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (handler *Handler) Ensure(emqx v1beta3.Emqx) error {
	resources := service.Generate(emqx)

	for _, resource := range resources {
		err := handler.createOrUpdate(resource, emqx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (handler *Handler) createOrUpdate(obj client.Object, emqx v1beta3.Emqx) error {
	logger := handler.logger.WithValues(
		"groupVersionKind", obj.GetObjectKind().GroupVersionKind().String(),
		"namespace", obj.GetNamespace(),
		"name", obj.GetName(),
	)

	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(obj.GetObjectKind().GroupVersionKind())
	err := handler.client.Get(
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
	case *corev1.ServiceAccount:
		opts = append(opts,
			patch.IgnoreField("metadata"), // ignore metadata.managedFields
			patch.IgnoreField("secret"),
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

	if err := client.NewDryRunClient(handler.client).Update(context.TODO(), obj); err != nil {
		return err
	}

	patchResult, err := patch.DefaultPatchMaker.Calculate(u, obj, opts...)
	if err != nil {
		logger.Error(err, "unable to patch with comparison object")
		return err
	}
	if !patchResult.IsEmpty() {
		if err := handler.doUpdate(obj, u); err != nil {
			return err
		}
		if err := handler.postUpdate(obj, emqx); err != nil {
			return err
		}
	}

	return nil
}

func (handler *Handler) doCreate(obj client.Object) error {
	logger := handler.logger.WithValues(
		"groupVersionKind", obj.GetObjectKind().GroupVersionKind().String(),
		"namespace", obj.GetNamespace(),
		"name", obj.GetName(),
	)
	if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(obj); err != nil {
		logger.Error(err, "unable to patch emqx with comparison object")
		return err
	}
	if err := handler.client.Create(context.TODO(), obj); err != nil {
		logger.Error(err, "crate resource failed")
		return err
	}
	logger.Info("create resource successfully")
	return nil
}

func (handler *Handler) doUpdate(obj, storageObj client.Object) error {
	logger := handler.logger.WithValues(
		"groupVersionKind", obj.GetObjectKind().GroupVersionKind().String(),
		"namespace", obj.GetNamespace(),
		"name", obj.GetName(),
	)
	if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(obj); err != nil {
		logger.Error(err, "unable to patch emqx with comparison object")
		return err
	}
	if err := handler.client.Update(context.TODO(), obj); err != nil {
		logger.Error(err, "update resource failed")
		return err
	}
	logger.Info("update resource successfully")
	return nil
}
func (handler *Handler) postUpdate(obj client.Object, emqx v1beta3.Emqx) error {
	names := v1beta3.Names{Object: emqx}
	if obj.GetName() == names.License() {
		err := handler.execToPods(emqx, "emqx", "emqx_ctl license reload /mounted/license/emqx.lic")
		if err != nil {
			return err
		}
	}
	if obj.GetName() == names.MQTTSCertificate() {
		err := handler.execToPods(emqx, "emqx", "listeners restart mqtt:ssl:external")
		if err != nil {
			return err
		}
	}

	if obj.GetName() == names.WSSCertificate() {
		err := handler.execToPods(emqx, "emqx", "listeners restart mqtt:wss:external")
		if err != nil {
			return err
		}
	}
	if obj.GetName() == names.Telegraf() {
		err := handler.execToPods(emqx, "telegraf", "/bin/kill 1")
		if err != nil {
			return err
		}
	}
	return nil
}

func (handler *Handler) getPods(emqx v1beta3.Emqx) (*corev1.PodList, error) {
	pods := &corev1.PodList{}
	err := handler.client.List(
		context.TODO(),
		pods,
		&client.ListOptions{
			Namespace:     emqx.GetNamespace(),
			LabelSelector: labels.SelectorFromSet(emqx.GetLabels()),
		},
	)
	return pods, err
}

func (handler *Handler) execToPods(emqx v1beta3.Emqx, containerName, command string) error {
	pods, err := handler.getPods(emqx)
	if err != nil {
		return err
	}
	for _, pod := range pods.Items {
		_, stderr, err := handler.executor.ExecToPod(pod.GetNamespace(), pod.GetName(), containerName, command, nil)
		if err != nil {
			return fmt.Errorf("exec %s container %s in pod %s error: %v", command, containerName, pod.GetName(), err)
		}
		if stderr != "" {
			return fmt.Errorf("exec %s container %s in pod %s stderr: %v", command, containerName, pod.GetName(), stderr)
		}
		handler.logger.WithValues(
			"groupVersionKind", pod.GetObjectKind().GroupVersionKind().String(),
			"namespace", pod.GetNamespace(),
			"name", pod.GetName(),
		).Info(fmt.Sprintf("exec %s to container %s successfully", command, containerName))
	}
	return nil
}
