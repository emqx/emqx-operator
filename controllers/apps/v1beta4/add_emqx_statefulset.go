package v1beta4

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	emperror "emperror.dev/errors"
	"github.com/banzaicloud/k8s-objectmatcher/patch"
	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type addEmqxStatefulSet struct {
	*EmqxReconciler
	*requestAPI
}

func (a addEmqxStatefulSet) reconcile(ctx context.Context, instance appsv1beta4.Emqx, args ...any) subResult {
	sts := args[0].(*appsv1.StatefulSet)
	newSts, err := a.getNewStatefulSet(instance, sts)
	if err != nil {
		return subResult{err: emperror.Wrap(err, "failed to get new statefulset")}
	}
	if err := a.CreateOrUpdateList(instance, a.Scheme, []client.Object{newSts}); err != nil {
		a.EventRecorder.Event(instance, corev1.EventTypeWarning, "FailedCreateOrUpdate", err.Error())
		return subResult{err: emperror.Wrap(err, "failed to create or update statefulset")}
	}

	enterprise, ok := instance.(*appsv1beta4.EmqxEnterprise)
	if !ok {
		return subResult{}
	}

	if enterprise.Status.EmqxBlueGreenUpdateStatus != nil &&
		enterprise.Status.EmqxBlueGreenUpdateStatus.StartedAt != nil {
		s := enterprise.Spec.EmqxBlueGreenUpdate.InitialDelaySeconds - int32(time.Since(enterprise.Status.EmqxBlueGreenUpdateStatus.StartedAt.Time).Seconds())
		if s > 0 {
			a.EventRecorder.Event(instance, corev1.EventTypeNormal, "Evacuate", fmt.Sprintf("Delay %d seconds", s))
			return subResult{result: &ctrl.Result{RequeueAfter: time.Duration(s) * time.Second}}
		}

		if err := a.syncStatefulSet(enterprise); err != nil {
			return subResult{err: emperror.Wrap(err, "failed to sync statefulset")}
		}
	}

	return subResult{}
}

func (a addEmqxStatefulSet) getNewStatefulSet(instance appsv1beta4.Emqx, sts *appsv1.StatefulSet) (*appsv1.StatefulSet, error) {
	enterprise, ok := instance.(*appsv1beta4.EmqxEnterprise)
	if !ok {
		return sts, nil
	}
	if enterprise.Spec.EmqxBlueGreenUpdate == nil {
		return sts, nil
	}

	allSts, _ := getAllStatefulSet(a.Client, instance)

	patchOpts := []patch.CalculateOption{
		justCheckPodTemplate(),
	}

	for i := range allSts {
		patchResult, _ := a.Patcher.Calculate(
			allSts[i].DeepCopy(),
			sts.DeepCopy(),
			patchOpts...,
		)
		if patchResult.IsEmpty() {
			sts.ObjectMeta = *allSts[i].ObjectMeta.DeepCopy()
			return sts, nil
		}
	}

	// Do-while loop
	var collisionCount *int32 = new(int32)
	for {
		podTemplateSpecHash := computeHash(&sts.Spec.Template, collisionCount)
		name := sts.Name + "-" + podTemplateSpecHash
		err := a.Client.Get(context.TODO(), types.NamespacedName{
			Namespace: sts.Namespace,
			Name:      name,
		}, &appsv1.StatefulSet{})
		*collisionCount++

		if err != nil {
			if k8sErrors.IsNotFound(err) {
				sts.Name = name
				return sts, nil
			}
			return nil, err
		}
	}
}

func (a addEmqxStatefulSet) syncStatefulSet(enterprise *appsv1beta4.EmqxEnterprise) error {
	if enterprise.Status.EmqxBlueGreenUpdateStatus == nil {
		return nil
	}

	inClusterStss, err := getInClusterStatefulSets(a.Client, enterprise)
	if err != nil {
		return err
	}

	podMap, err := getPodMap(a.Client, enterprise, inClusterStss)
	if err != nil {
		return err
	}

	currentSts := &appsv1.StatefulSet{}
	if err := a.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: enterprise.Namespace,
		Name:      enterprise.Status.EmqxBlueGreenUpdateStatus.CurrentStatefulSet,
	}, currentSts); err != nil {
		return emperror.Wrap(err, "failed to get current statefulset")
	}

	originSts := &appsv1.StatefulSet{}
	if err := a.Client.Get(context.TODO(), types.NamespacedName{
		Namespace: enterprise.Namespace,
		Name:      enterprise.Status.EmqxBlueGreenUpdateStatus.OriginStatefulSet,
	}, originSts); err != nil {
		return emperror.Wrap(err, "failed to get origin statefulset")
	}

	if a.canBeScaledDown(enterprise, originSts, podMap) {
		scaleDown := *originSts.Spec.Replicas - 1
		stsCopy := originSts.DeepCopy()
		if err := a.Client.Get(context.TODO(), client.ObjectKeyFromObject(stsCopy), stsCopy); err != nil {
			if !k8sErrors.IsNotFound(err) {
				return err
			}
		}
		stsCopy.Spec.Replicas = &scaleDown

		a.EventRecorder.Event(enterprise, corev1.EventTypeNormal, "ScaleDown", fmt.Sprintf("Scale down StatefulSet %s to %d", originSts.Name, scaleDown))
		if err := a.Client.Update(context.TODO(), stsCopy); err != nil {
			return err
		}
	}

	if len(enterprise.Status.EmqxBlueGreenUpdateStatus.EvacuationsStatus) == 0 {
		pods := podMap[originSts.UID]
		if len(pods) == 0 {
			return nil
		}
		// evacuate the last pod
		sort.Sort(PodsByNameNewer(pods))
		emqxNodeName := getEmqxNodeName(enterprise, pods[0])

		a.EventRecorder.Event(enterprise, corev1.EventTypeNormal, "Evacuate", fmt.Sprintf("Evacuate node %s start", emqxNodeName))
		if err := a.startEvacuateNodeByAPI(enterprise, podMap[currentSts.UID], emqxNodeName); err != nil {
			a.EventRecorder.Event(enterprise, corev1.EventTypeWarning, "Evacuate", fmt.Sprintf("Evacuate node %s failed: %s", emqxNodeName, err.Error()))
			return emperror.Wrap(err, "evacuate node failed")
		}
	}

	return nil
}

func (a addEmqxStatefulSet) canBeScaledDown(enterprise *appsv1beta4.EmqxEnterprise, originSts *appsv1.StatefulSet, podMap map[types.UID][]*corev1.Pod) bool {
	for _, e := range enterprise.Status.EmqxBlueGreenUpdateStatus.EvacuationsStatus {
		if *e.Stats.CurrentConnected == 0 && *e.Stats.CurrentSessions == 0 && e.State == "prohibiting" {
			podName := strings.Split(strings.Split(e.Node, "@")[1], ".")[0]
			if strings.Contains(podName, originSts.Name) {
				pods := podMap[originSts.UID]
				// Get latest pod for sts
				sort.Sort(PodsByNameNewer(pods))
				if pods[0].Name == podName {
					a.EventRecorder.Event(enterprise, corev1.EventTypeNormal, "Evacuate", fmt.Sprintf("Evacuate node %s successfully", getEmqxNodeName(enterprise, pods[0])))
					return true
				}
			}
		}
	}
	return false
}
