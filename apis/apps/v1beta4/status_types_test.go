package v1beta4

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIndexCondition(t *testing.T) {
	conditions := []Condition{
		{
			Type:   ConditionInitResourceReady,
			Status: v1.ConditionTrue,
		},
		{
			Type:   ConditionRunning,
			Status: v1.ConditionFalse,
		},
	}
	assert.Equal(t, 0, indexCondition(conditions, ConditionInitResourceReady))
	assert.Equal(t, 1, indexCondition(conditions, ConditionRunning))
}

func TestSortConditions(t *testing.T) {
	conditions := []Condition{
		{
			Type:         ConditionInitResourceReady,
			LastUpdateAt: metav1.NewTime(time.Now().Add(-24 * time.Hour)),
		},
		{
			Type:         ConditionRunning,
			LastUpdateAt: metav1.Now(),
		},
	}
	sortConditions(conditions)

	assert.Equal(t, ConditionRunning, conditions[0].Type)
	assert.Equal(t, ConditionInitResourceReady, conditions[1].Type)
}
