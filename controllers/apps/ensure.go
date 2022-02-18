package apps

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/banzaicloud/k8s-objectmatcher/patch"
	"github.com/emqx/emqx-operator/apis/apps/v1beta1"
	"github.com/emqx/emqx-operator/pkg/service"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (handler *Handler) Ensure(emqx v1beta1.Emqx) error {
	resources := service.Generate(emqx)

	for _, resource := range resources {
		err := handler.createOrUpdate(resource, emqx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (handler *Handler) createOrUpdate(obj client.Object, emqx v1beta1.Emqx) error {
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
			if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(obj); err != nil {
				// handler.logger.Error(err, "Unable to patch emqx %s with comparison object", obj.GetObjectKind().GroupVersionKind().Kind)
				logger.Error(err, "Unable to patch with comparison object")
				return err
			}
			return handler.doCreate(obj)
		}
		return err
	}

	opts := []patch.CalculateOption{}
	switch obj.(type) {
	case *appsv1.StatefulSet:
		opts = append(
			opts,
			patch.IgnoreStatusFields(),
			patch.IgnoreVolumeClaimTemplateTypeMetaAndStatus(),
		)
	case *corev1.Secret:
		opts = append(
			opts,
			patch.IgnoreField("stringData"),
		)
	}

	if _, ok := obj.(*appsv1.StatefulSet); ok {
		opts = []patch.CalculateOption{
			patch.IgnoreStatusFields(),
			patch.IgnoreVolumeClaimTemplateTypeMetaAndStatus(),
			patch.IgnoreField("metadata"),
		}
	}

	patchResult, err := patch.DefaultPatchMaker.Calculate(u, obj, opts...)
	if err != nil {
		// handler.logger.Error(err, "Unable to patch emqx %s with comparison object", obj.GetObjectKind().GroupVersionKind().Kind)
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
	err := handler.client.Create(context.TODO(), obj)
	if err != nil {
		return err
	}
	logger.Info("create resource successfully")
	return nil
}

func (handler *Handler) doUpdate(obj, storageObj client.Object) error {
	obj.SetResourceVersion(storageObj.GetResourceVersion())
	obj.SetCreationTimestamp(storageObj.GetCreationTimestamp())
	obj.SetManagedFields(storageObj.GetManagedFields())

	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	for key, value := range storageObj.GetAnnotations() {
		if _, present := annotations[key]; !present {
			annotations[key] = value
		}
		obj.SetAnnotations(annotations)
	}
	if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(obj); err != nil {
		return err
	}

	logger := handler.logger.WithValues(
		"groupVersionKind", obj.GetObjectKind().GroupVersionKind().String(),
		"namespace", obj.GetNamespace(),
		"name", obj.GetName(),
	)
	err := handler.client.Update(context.TODO(), obj)
	if err != nil {
		return err
	}
	logger.Info("update resource successfully")
	return nil
}
func (handler *Handler) postUpdate(obj client.Object, emqx v1beta1.Emqx) error {
	if obj.GetName() == fmt.Sprintf("%s-%s", emqx.GetName(), "license") {
		pods, err := handler.getPods(emqx)
		if err != nil {
			return err
		}
		for _, pod := range pods.Items {
			_, stderr, err := handler.executor.ExecToPod(emqx.GetNamespace(), pod.GetName(), emqx.GetName(), "emqx_ctl license reload /mounted/license/emqx.lic", nil)
			if err != nil {
				return fmt.Errorf("exec pod %s error: %v", pod.GetName(), err)
			}
			if stderr != "" {
				return fmt.Errorf("pod %s update license failed: %s", pod.GetName(), stderr)
			}
			_, stderr, err = handler.executor.ExecToPod(emqx.GetNamespace(), pod.GetName(), emqx.GetName(), "emqx_ctl license info", nil)
			if err != nil {
				return fmt.Errorf("exec pod %s error: %v", pod.GetName(), err)
			}
			if stderr != "" {
				return fmt.Errorf("pod %s get license info failed: %s", pod.GetName(), stderr)
			}
			handler.logger.Info(fmt.Sprintf("pod %s update license successfully", pod.GetName()))
		}
	}
	return nil
}

func (handler *Handler) getPods(emqx v1beta1.Emqx) (*corev1.PodList, error) {
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
