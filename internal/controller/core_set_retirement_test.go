package controller

import (
	"testing"
	"time"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	util "github.com/emqx/emqx-operator/internal/controller/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/utils/ptr"
)

func TestCoreSetRetirementState(t *testing.T) {
	coreSet := &appsv1.StatefulSet{Spec: appsv1.StatefulSetSpec{Replicas: ptr.To(int32(4))}}
	now := time.Now().Truncate(time.Nanosecond)
	retirement := &coreSetRetirement{watermark: 5, updatedAt: now}
	util.AttachAnnotations(coreSet, retirement.annotations())

	state, err := readCoreSetRetirementState(coreSet)
	require.NoError(t, err)
	assert.Equal(t, int32(5), state.watermark)
	assert.Equal(t, now.UTC(), state.updatedAt)

	coreSet.Annotations[crd.AnnotationCoreRetirementOrdinalWatermark] = "6"
	state, err = readCoreSetRetirementState(coreSet)
	require.NoError(t, err)
	assert.Equal(t, int32(6), state.watermark)

	coreSet.Annotations[crd.AnnotationCoreRetirementOrdinalWatermark] = "-1"
	_, err = readCoreSetRetirementState(coreSet)
	require.ErrorContains(t, err, "invalid reusable core pod ordinal watermark")
}

func TestRetiringCorePodOrdinalRange(t *testing.T) {
	coreSet := &appsv1.StatefulSet{Spec: appsv1.StatefulSetSpec{Replicas: ptr.To(int32(3))}}
	state := &coreSetRetirement{watermark: 6}

	assert.False(t, state.isRetiringOrdinal(coreSet, 2))
	assert.True(t, state.isRetiringOrdinal(coreSet, 3))
	assert.True(t, state.isRetiringOrdinal(coreSet, 4))
	assert.True(t, state.isRetiringOrdinal(coreSet, 5))
	assert.False(t, state.isRetiringOrdinal(coreSet, 6))
}
