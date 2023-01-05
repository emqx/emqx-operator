package v1beta4

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

func TestBrokerAddCondition(t *testing.T) {
	status := &EmqxBrokerStatus{}

	status.AddCondition(ConditionInitResourceReady, v1.ConditionTrue, "fake", "fake")
	assert.Equal(t, 1, len(status.Conditions))
}
