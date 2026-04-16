package util

import (
	"encoding/json"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func HaveCondition(conditionType string, matcher types.GomegaMatcher) types.GomegaMatcher {
	return gomega.WithTransform(
		func(instance *crd.EMQX) *metav1.Condition {
			_, condition := instance.Status.GetCondition(conditionType)
			return condition
		},
		matcher,
	)
}

func HaveLabel(label string, matcher types.GomegaMatcher) types.GomegaMatcher {
	return gomega.HaveField("Labels", gomega.HaveKeyWithValue(label, matcher))
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
