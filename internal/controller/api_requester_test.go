package controller

import (
	"testing"
	"time"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	req "github.com/emqx/emqx-operator/internal/requester"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
)

func TestRequesterFilter(t *testing.T) {
	var coreSetName = "emqx-core"
	var coreSetUID types.UID = "123"

	instance := &crd.EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
		Status: crd.EMQXStatus{
			CoreNodesStatus: crd.CoreNodesStatus{},
			ReplicantNodesStatus: crd.ReplicantNodesStatus{
				CurrentRevision: "cur",
				UpdateRevision:  "upd",
			},
			CoreNodes: []crd.EMQXNode{
				{
					PodName:     coreSetName + "-0",
					Name:        "emqx@core-0",
					Status:      "running",
					OTPRelease:  "27.2-3/15.2",
					Version:     "5.10.0",
					Role:        "core",
					Sessions:    0,
					Connections: 0,
				},
				{
					PodName:     coreSetName + "-1",
					Name:        "emqx@core-1",
					Status:      "running",
					OTPRelease:  "27.2-3/15.2",
					Version:     "5.10.0",
					Role:        "core",
					Sessions:    0,
					Connections: 0,
				},
			},
			ReplicantNodes: []crd.EMQXNode{},
		},
	}

	coreOwnerReference := metav1.OwnerReference{
		APIVersion: "apps/v1",
		Kind:       "StatefulSet",
		Name:       coreSetName,
		UID:        coreSetUID,
		Controller: ptr.To(true),
	}

	podSpec := corev1.PodSpec{
		Containers: []corev1.Container{{
			Name: crd.DefaultContainerName,
			Ports: []corev1.ContainerPort{{
				Name:          "dashboard",
				ContainerPort: 18083,
			}},
		}}}

	state := &reconcileState{
		coreSets: []*appsv1.StatefulSet{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: coreSetName,
					UID:  coreSetUID,
				},
				Status: appsv1.StatefulSetStatus{
					Replicas:       2,
					UpdateRevision: "upd",
				},
			},
		},
		replicantSets: []*appsv1.ReplicaSet{},
		pods: []*corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              coreSetName + "-0",
					Labels:            crd.CoreLabels(),
					CreationTimestamp: metav1.NewTime(time.Now().Add(-1 * time.Minute)),
					OwnerReferences:   []metav1.OwnerReference{coreOwnerReference},
				},
				Spec: podSpec,
				Status: corev1.PodStatus{
					PodIP:      "",
					Phase:      corev1.PodPending,
					Conditions: []corev1.PodCondition{},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              coreSetName + "-1",
					Labels:            crd.CoreLabels(),
					CreationTimestamp: metav1.NewTime(time.Now().Add(-1 * time.Second)),
					OwnerReferences:   []metav1.OwnerReference{coreOwnerReference},
				},
				Spec: podSpec,
				Status: corev1.PodStatus{
					PodIP:      "10.0.0.2",
					Phase:      corev1.PodRunning,
					Conditions: []corev1.PodCondition{{Type: corev1.ContainersReady, Status: corev1.ConditionTrue}},
				},
			},
		},
	}

	builder := &apiRequesterBuilder{
		username: "emqx",
		password: "emqx",
	}

	var requester req.RequesterInterface

	requester = builder.forCore(state)
	assert.NotNil(t, requester)
	assert.Equal(t, state.pods[1].Name, requester.GetDescription())

	requester = builder.forPod(state.pods[0])
	assert.Nil(t, requester)

	requester = builder.forPod(state.pods[1])
	assert.NotNil(t, requester)

	// Filter by the single core StatefulSet:
	requester = builder.forCore(state, &podsManagedBy{state.coreSet()})
	assert.NotNil(t, requester)
	assert.Equal(t, state.pods[1].Name, requester.GetDescription())

	requester = builder.forCore(state, &podsWithEMQXVersion{instance, "5.10."})
	assert.NotNil(t, requester)

	requester = builder.forCore(state, &podsWithEMQXVersion{instance, "6."})
	assert.Nil(t, requester)

}

func TestCorePreference(t *testing.T) {
	const coreSetName = "emqx-core"

	coreSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: coreSetName,
			UID:  "abcdef",
		},
		Status: appsv1.StatefulSetStatus{
			UpdateRevision: "rev-new",
		},
	}
	coreSetReference := metav1.OwnerReference{
		APIVersion: "apps/v1",
		Kind:       "StatefulSet",
		Name:       coreSetName,
		UID:        "abcdef",
		Controller: ptr.To(true),
	}

	mkPod := func(name, revision, ip string, port int32, ready bool, deleting bool) *corev1.Pod {
		labels := crd.CoreLabels()
		labels[appsv1.ControllerRevisionHashLabelKey] = revision
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:              name,
				Labels:            labels,
				CreationTimestamp: metav1.NewTime(time.Now()),
				OwnerReferences:   []metav1.OwnerReference{coreSetReference},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name: crd.DefaultContainerName,
					Ports: []corev1.ContainerPort{{
						Name:          "dashboard",
						ContainerPort: port,
					}},
				}}},
			Status: corev1.PodStatus{
				PodIP: ip,
				Phase: corev1.PodRunning,
			},
		}
		if ready {
			pod.Status.Conditions = []corev1.PodCondition{
				{Type: corev1.ContainersReady, Status: corev1.ConditionTrue},
			}
		}
		if deleting {
			pod.DeletionTimestamp = ptr.To(metav1.NewTime(time.Now()))
		}
		return pod
	}

	builder := &apiRequesterBuilder{
		username: "emqx",
		password: "emqx",
	}

	t.Run("skips deleting and not ready pods", func(t *testing.T) {
		state := &reconcileState{
			coreSets: []*appsv1.StatefulSet{coreSet},
			pods: []*corev1.Pod{
				mkPod(coreSetName+"-0", "rev-new", "10.0.0.1", 18083, true, true),
				mkPod(coreSetName+"-1", "rev-new", "10.0.0.2", 18083, false, false),
				mkPod(coreSetName+"-2", "rev-new", "10.0.0.3", 18083, true, false),
			},
		}

		requester := builder.forCore(state)

		assert.NotNil(t, requester)
		assert.Equal(t, coreSetName+"-2", requester.GetDescription())
	})

	t.Run("returns eligible pods in preference order for non-HTTP callers", func(t *testing.T) {
		state := &reconcileState{
			coreSets: []*appsv1.StatefulSet{coreSet},
			pods: []*corev1.Pod{
				mkPod(coreSetName+"-0", "rev-new", "", 18083, true, true),
				mkPod(coreSetName+"-1", "rev-new", "", 18083, false, false),
				mkPod(coreSetName+"-2", "rev-old", "", 18083, true, false),
				mkPod(coreSetName+"-3", "rev-new", "", 18083, true, false),
			},
		}

		pods := preferredCorePods(state, podsManagedBy{coreSet})

		require.Len(t, pods, 2)
		assert.Equal(t, coreSetName+"-3", pods[0].Name)
		assert.Equal(t, coreSetName+"-2", pods[1].Name)
	})

	t.Run("deprioritizes sole outdated pod over larger fresh ordinal", func(t *testing.T) {
		state := &reconcileState{
			coreSets: []*appsv1.StatefulSet{coreSet},
			pods: []*corev1.Pod{
				mkPod(coreSetName+"-0", "rev-old", "10.0.0.1", 18083, true, false),
				mkPod(coreSetName+"-1", "rev-new", "10.0.0.2", 18083, true, false),
			},
		}

		requester := builder.forCore(state)

		assert.NotNil(t, requester)
		assert.Equal(t, coreSetName+"-1", requester.GetDescription())
	})

	t.Run("prefers smaller ordinal when more than one outdated pod remains", func(t *testing.T) {
		state := &reconcileState{
			coreSets: []*appsv1.StatefulSet{coreSet},
			pods: []*corev1.Pod{
				mkPod(coreSetName+"-0", "rev-old", "10.0.0.1", 18083, true, false),
				mkPod(coreSetName+"-1", "rev-old", "10.0.0.2", 18083, true, false),
				mkPod(coreSetName+"-2", "rev-new", "10.0.0.3", 18083, true, false),
			},
		}

		requester := builder.forCore(state)

		assert.NotNil(t, requester)
		assert.Equal(t, coreSetName+"-0", requester.GetDescription())
	})

	t.Run("prefers smaller ordinal among equally fresh pods", func(t *testing.T) {
		state := &reconcileState{
			coreSets: []*appsv1.StatefulSet{coreSet},
			pods: []*corev1.Pod{
				mkPod(coreSetName+"-2", "rev-new", "10.0.0.3", 18083, true, false),
				mkPod(coreSetName+"-1", "rev-new", "10.0.0.2", 18083, true, false),
			},
		}

		requester := builder.forCore(state)

		assert.NotNil(t, requester)
		assert.Equal(t, coreSetName+"-1", requester.GetDescription())
	})
}

func TestRequesterUsesPodDashboardPort(t *testing.T) {
	builder := &apiRequesterBuilder{username: "emqx", password: "secret"}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "emqx-core-0"},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{
			Name: crd.DefaultContainerName,
			Ports: []corev1.ContainerPort{{
				Name:          "dashboard",
				ContainerPort: 28083,
			}},
		}}},
		Status: corev1.PodStatus{PodIP: "10.0.0.1"},
	}

	requester := builder.forPod(pod)
	require.NotNil(t, requester)
	assert.Equal(t, "10.0.0.1:28083", requester.GetHost())
	assert.Equal(t, "http", requester.GetURL("status").Scheme)

	pod.Spec.Containers[0].Ports = []corev1.ContainerPort{
		{Name: "dashboard-https", ContainerPort: 28084},
	}
	requester = builder.forPod(pod)
	require.NotNil(t, requester)
	assert.Equal(t, "10.0.0.1:28084", requester.GetHost())
	assert.Equal(t, "https", requester.GetURL("status").Scheme)

	pod.Spec.Containers[0].Ports = []corev1.ContainerPort{
		{Name: "dashboard-https", ContainerPort: 28084},
		{Name: "dashboard", ContainerPort: 28083},
	}
	requester = builder.forPod(pod)
	require.NotNil(t, requester)
	assert.Equal(t, "10.0.0.1:28083", requester.GetHost())
	assert.Equal(t, "http", requester.GetURL("status").Scheme)

	pod.Spec.Containers[0].Ports = nil
	assert.Nil(t, builder.forPod(pod))
}
