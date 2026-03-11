package controller

import (
	"testing"
	"time"

	crdv2 "github.com/emqx/emqx-operator/api/v2"
	req "github.com/emqx/emqx-operator/internal/requester"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
)

func TestRequesterFilter(t *testing.T) {
	var coreSetName string = "emqx-core"
	var coreSetUID types.UID = "123"

	instance := &crdv2.EMQX{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx",
		},
		Status: crdv2.EMQXStatus{
			CoreNodesStatus: crdv2.CoreNodesStatus{},
			ReplicantNodesStatus: crdv2.ReplicantNodesStatus{
				CurrentRevision: "cur",
				UpdateRevision:  "upd",
			},
			CoreNodes: []crdv2.EMQXNode{
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
			ReplicantNodes: []crdv2.EMQXNode{},
		},
	}

	coreOwnerReference := metav1.OwnerReference{
		APIVersion: "apps/v1",
		Kind:       "StatefulSet",
		Name:       coreSetName,
		UID:        coreSetUID,
		Controller: ptr.To(true),
	}

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
					Labels:            crdv2.CoreLabels(),
					CreationTimestamp: metav1.NewTime(time.Now().Add(-1 * time.Minute)),
					OwnerReferences:   []metav1.OwnerReference{coreOwnerReference},
				},
				Status: corev1.PodStatus{
					PodIP:      "10.0.0.1",
					Phase:      corev1.PodPending,
					Conditions: []corev1.PodCondition{},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:              coreSetName + "-1",
					Labels:            crdv2.CoreLabels(),
					CreationTimestamp: metav1.NewTime(time.Now().Add(-1 * time.Second)),
					OwnerReferences:   []metav1.OwnerReference{coreOwnerReference},
				},
				Status: corev1.PodStatus{
					PodIP:      "10.0.0.2",
					Phase:      corev1.PodRunning,
					Conditions: []corev1.PodCondition{{Type: corev1.ContainersReady, Status: corev1.ConditionTrue}},
				},
			},
		},
	}

	builder := &apiRequesterBuilder{
		schema:   "http",
		port:     "1883",
		username: "emqx",
		password: "emqx",
	}

	var requester req.RequesterInterface

	requester = builder.forOldestCore(state)
	assert.NotNil(t, requester)
	assert.Equal(t, state.pods[1].Name, requester.GetDescription())

	requester = builder.forPod(state.pods[0])
	assert.Nil(t, requester)

	requester = builder.forPod(state.pods[1])
	assert.NotNil(t, requester)

	// Filter by the single core StatefulSet:
	requester = builder.forOldestCore(state, &managedByFilter{state.coreSet()})
	assert.NotNil(t, requester)
	assert.Equal(t, state.pods[1].Name, requester.GetDescription())

	requester = builder.forOldestCore(state, &emqxVersionFilter{instance, "5.10."})
	assert.NotNil(t, requester)

	requester = builder.forOldestCore(state, &emqxVersionFilter{instance, "6."})
	assert.Nil(t, requester)

}
