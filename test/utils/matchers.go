package utils

import (
	"encoding/json"

	appsv2beta1 "github.com/emqx/emqx-operator/api/v2beta1"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func HaveCondition(conditionType string, matcher types.GomegaMatcher) types.GomegaMatcher {
	return gomega.WithTransform(
		func(instance *appsv2beta1.EMQX) *metav1.Condition {
			_, condition := instance.Status.GetCondition(conditionType)
			return condition
		},
		matcher,
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
