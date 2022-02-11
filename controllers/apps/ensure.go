package apps

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/banzaicloud/k8s-objectmatcher/patch"
	"github.com/emqx/emqx-operator/apis/apps/v1beta1"
	"github.com/emqx/emqx-operator/pkg/service"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (handler *Handler) Ensure(emqx v1beta1.Emqx) error {
	resources := service.Generate(emqx)

	for _, resource := range resources {
		err := handler.createOrUpdate(resource)
		if err != nil {
			return err
		}
	}

	return nil
}

func (handler *Handler) createOrUpdate(obj client.Object) error {
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

	var opts []patch.CalculateOption

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
		logger.Error(err, "Unable to patch with comparison object")
		return err
	}
	if !patchResult.IsEmpty() {
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
		if err := patch.DefaultAnnotator.SetLastAppliedAnnotation(obj); err != nil {
			return err
		}
		return handler.doUpdate(obj)
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
	logger.Info("Create successfully")
	return nil
}

func (handler *Handler) doUpdate(obj client.Object) error {
	logger := handler.logger.WithValues(
		"groupVersionKind", obj.GetObjectKind().GroupVersionKind().String(),
		"namespace", obj.GetNamespace(),
		"name", obj.GetName(),
	)
	err := handler.client.Update(context.TODO(), obj)
	if err != nil {
		return err
	}
	logger.Info("Update successfully")
	return nil
}
