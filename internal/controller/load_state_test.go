package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
)

func TestListOutdatedPods(t *testing.T) {
	const coreSetName = "emqx-core"
	const coreSetUID = types.UID("test-sts-uid")
	const otherOwnerUID = types.UID("other-rs-uid")

	mkCoreSet := func(currentRev, updateRev string) *appsv1.StatefulSet {
		return &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      coreSetName,
				Namespace: "default",
				UID:       coreSetUID,
			},
			Status: appsv1.StatefulSetStatus{
				CurrentRevision: currentRev,
				UpdateRevision:  updateRev,
			},
		}
	}

	mkCoreSetPod := func(name string, revision ...string) *corev1.Pod {
		labels := map[string]string{}
		if len(revision) == 1 {
			labels[appsv1.ControllerRevisionHashLabelKey] = revision[0]
		}
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "default",
				Labels:    labels,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "apps/v1",
						Kind:       "StatefulSet",
						Name:       coreSetName,
						UID:        coreSetUID,
						Controller: ptr.To(true),
					},
				},
			},
		}
	}

	mkReplicantPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx-replicant-xyz",
			Namespace: "default",
			Labels: map[string]string{
				appsv1.ControllerRevisionHashLabelKey: "some-replicant-hash",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "ReplicaSet",
					Name:       "emqx-replicant-hash",
					UID:        otherOwnerUID,
					Controller: ptr.To(true),
				},
			},
		},
	}

	t.Run("empty updateRevision yields no outdated pods", func(t *testing.T) {
		sts := mkCoreSet("rev-a", "")
		state := &reconcileState{
			coreSets: []*appsv1.StatefulSet{sts},
			pods: []*corev1.Pod{
				mkCoreSetPod(coreSetName+"-0", "rev-a"),
				mkCoreSetPod(coreSetName+"-1", "rev-b"),
			},
		}
		assert.Empty(t, state.listOutdatedPods())
	})

	t.Run("standard rolling update: pods on currentRevision only", func(t *testing.T) {
		sts := mkCoreSet("rev-old", "rev-new")
		state := &reconcileState{
			coreSets: []*appsv1.StatefulSet{sts},
			pods: []*corev1.Pod{
				mkCoreSetPod(coreSetName+"-0", "rev-old"),
				mkCoreSetPod(coreSetName+"-1", "rev-old"),
			},
		}
		got := state.listOutdatedPods()
		require.Len(t, got, 2)
		assert.Equal(t, coreSetName+"-0", got[0].Name)
		assert.Equal(t, coreSetName+"-1", got[1].Name)
	})

	t.Run("partial rollout: only pods not on updateRevision", func(t *testing.T) {
		sts := mkCoreSet("rev-old", "rev-new")
		state := &reconcileState{
			coreSets: []*appsv1.StatefulSet{sts},
			pods: []*corev1.Pod{
				mkCoreSetPod(coreSetName+"-0", "rev-new"),
				mkCoreSetPod(coreSetName+"-1", "rev-old"),
			},
		}
		got := state.listOutdatedPods()
		require.Len(t, got, 1)
		assert.Equal(t, coreSetName+"-1", got[0].Name)
	})

	t.Run("untracked third revision during rollout", func(t *testing.T) {
		sts := mkCoreSet("rev-old", "rev-new")
		state := &reconcileState{
			coreSets: []*appsv1.StatefulSet{sts},
			pods: []*corev1.Pod{
				mkCoreSetPod(coreSetName+"-0", "rev-ancient"),
				mkCoreSetPod(coreSetName+"-1", "rev-old"),
			},
		}
		got := state.listOutdatedPods()
		require.Len(t, got, 2)
		assert.Equal(t, coreSetName+"-0", got[0].Name)
		assert.Equal(t, coreSetName+"-1", got[1].Name)
	})

	t.Run("all pods on updateRevision", func(t *testing.T) {
		sts := mkCoreSet("rev-old", "rev-new")
		state := &reconcileState{
			coreSets: []*appsv1.StatefulSet{sts},
			pods: []*corev1.Pod{
				mkCoreSetPod(coreSetName+"-0", "rev-new"),
				mkCoreSetPod(coreSetName+"-1", "rev-new"),
			},
		}
		assert.Empty(t, state.listOutdatedPods())
	})

	t.Run("pods not managed by core StatefulSet are ignored", func(t *testing.T) {
		sts := mkCoreSet("rev-old", "rev-new")
		state := &reconcileState{
			coreSets: []*appsv1.StatefulSet{sts},
			pods: []*corev1.Pod{
				mkCoreSetPod(coreSetName+"-0", "rev-old"),
				mkReplicantPod,
			},
		}
		got := state.listOutdatedPods()
		require.Len(t, got, 1)
		assert.Equal(t, coreSetName+"-0", got[0].Name)
	})

	t.Run("missing controller-revision-hash label counts as outdated", func(t *testing.T) {
		sts := mkCoreSet("rev-old", "rev-new")
		state := &reconcileState{
			coreSets: []*appsv1.StatefulSet{sts},
			pods: []*corev1.Pod{
				mkCoreSetPod(coreSetName + "-0"),
			},
		}
		got := state.listOutdatedPods()
		require.Len(t, got, 1)
		assert.Equal(t, coreSetName+"-0", got[0].Name)
	})

	t.Run("currentRevision empty but updateRevision set still lists mismatches", func(t *testing.T) {
		sts := mkCoreSet("", "rev-new")
		state := &reconcileState{
			coreSets: []*appsv1.StatefulSet{sts},
			pods: []*corev1.Pod{
				mkCoreSetPod(coreSetName+"-0", "rev-old"),
			},
		}
		got := state.listOutdatedPods()
		require.Len(t, got, 1)
		assert.Equal(t, coreSetName+"-0", got[0].Name)
	})
}
