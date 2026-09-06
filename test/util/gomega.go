package util

import (
	"encoding/json"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func xformExtractController(o any) *metav1.OwnerReference {
	switch o := o.(type) {
	case metav1.Object:
		return metav1.GetControllerOf(o)
	case metav1.PartialObjectMetadata:
		return metav1.GetControllerOf(&o.ObjectMeta)
	}
	return nil
}

func HaveCondition(conditionType string, matcher types.GomegaMatcher) types.GomegaMatcher {
	return gomega.WithTransform(
		func(instance *crd.EMQX) *metav1.Condition {
			return instance.Status.GetCondition(conditionType)
		},
		matcher,
	)
}

func HaveLabel(label string, matcher types.GomegaMatcher) types.GomegaMatcher {
	return gomega.HaveField("Labels", gomega.HaveKeyWithValue(label, matcher))
}

func BeControlledBy(controller metav1.Object) types.GomegaMatcher {
	return gomega.WithTransform(
		xformExtractController,
		gomega.And(
			gomega.Not(gomega.BeNil()),
			gomega.HaveValue(gomega.HaveField("UID", gomega.Equal(controller.GetUID()))),
		),
	)
}

func BeNotControlled() types.GomegaMatcher {
	return gomega.WithTransform(
		xformExtractController,
		gomega.BeNil(),
	)
}

func UnmarshalInto(v any) types.GomegaMatcher {
	return gomega.WithTransform(
		func(in string) error {
			return json.Unmarshal([]byte(in), v)
		},
		gomega.Succeed(),
	)
}

func BeUnmarshalledAs(v any, matcher types.GomegaMatcher) types.GomegaMatcher {
	return gomega.WithTransform(
		func(in string) (any, error) {
			err := json.Unmarshal([]byte(in), &v)
			return v, err
		},
		gomega.HaveValue(matcher),
	)
}
