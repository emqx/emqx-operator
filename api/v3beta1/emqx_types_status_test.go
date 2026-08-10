package v3beta1

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSetConditionTransitionTime(t *testing.T) {
	oldTransition := metav1.NewTime(time.Now().Add(-time.Hour))
	status := EMQXStatus{
		Conditions: []metav1.Condition{
			{
				Type:               Ready,
				Status:             metav1.ConditionFalse,
				Reason:             "Progressing",
				Message:            "first attempt",
				LastTransitionTime: oldTransition,
			},
		},
	}

	status.SetCondition(Ready, metav1.ConditionFalse, "Progressing", "second attempt")
	_, condition := status.GetCondition(Ready)
	if assert.NotNil(t, condition) {
		assert.Equal(t, oldTransition, condition.LastTransitionTime)
		assert.Equal(t, "second attempt", condition.Message)
	}

	status.SetCondition(Ready, metav1.ConditionTrue, "UpToDate", "third attempt")
	_, condition = status.GetCondition(Ready)
	if assert.NotNil(t, condition) {
		assert.NotEqual(t, oldTransition, condition.LastTransitionTime)
		assert.Equal(t, metav1.ConditionTrue, condition.Status)
	}
}
