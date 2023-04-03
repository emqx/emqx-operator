package v1beta4

import (
	"context"
	"encoding/json"
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
	PortForwardAPI
}

func (a addEmqxStatefulSet) reconcile(ctx context.Context, instance appsv1beta4.Emqx, args ...any) subResult {
	sts := args[0].(*appsv1.StatefulSet)
	newSts, err := a.getNewStatefulSet(instance, sts)
	if err != nil {
		return subResult{err: emperror.Wrap(err, "failed to get new statefulset")}
	}
	if err := a.CreateOrUpdateList(instance, a.Scheme, []client.Object{newSts}); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to create or update statefulset")}
	}

	if isEnterprise, enterprise := a.isEmqxEnterprise(instance); isEnterprise {
		return a.handleBlueGreenUpdate(enterprise)
	}

	return subResult{}
}

// Check whether it is Emqx enterprise version
func (a *addEmqxStatefulSet) isEmqxEnterprise(instance appsv1beta4.Emqx) (bool, *appsv1beta4.EmqxEnterprise) {
	enterprise, ok := instance.(*appsv1beta4.EmqxEnterprise)
	return ok, enterprise
}

// Handle Emqx BlueGreen Update
func (a *addEmqxStatefulSet) handleBlueGreenUpdate(enterprise *appsv1beta4.EmqxEnterprise) subResult {
	if enterprise.Status.EmqxBlueGreenUpdateStatus == nil {
		return subResult{}
	}
	if enterprise.Status.EmqxBlueGreenUpdateStatus.StartedAt == nil {
		return subResult{}
	}

	// Calculate the remaining delay time, and do not perform subsequent operations within the delay time
	delay := enterprise.Spec.EmqxBlueGreenUpdate.InitialDelaySeconds - int32(time.Since(enterprise.Status.EmqxBlueGreenUpdateStatus.StartedAt.Time).Seconds())
	if delay > 0 {
		a.EventRecorder.Event(enterprise, corev1.EventTypeNormal, "Evacuate", fmt.Sprintf("Delay %d seconds", delay))
		return subResult{result: ctrl.Result{RequeueAfter: time.Duration(delay) * time.Second}}
	}

	if err := a.syncStatefulSet(enterprise); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to sync statefulset")}
	}
	return subResult{}
}

func (a addEmqxStatefulSet) getNewStatefulSet(instance appsv1beta4.Emqx, sts *appsv1.StatefulSet) (*appsv1.StatefulSet, error) {
	if isEnterprise, enterprise := a.isEmqxEnterprise(instance); !isEnterprise || enterprise.Spec.EmqxBlueGreenUpdate == nil {
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

		a.EventRecorder.Event(enterprise, corev1.EventTypeNormal, "Evacuate", fmt.Sprintf("Scale down StatefulSet %s to %d", originSts.Name, scaleDown))
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
			return emperror.Wrapf(err, "Evacuate node %s failed: %s", emqxNodeName, err.Error())
		}
	}

	return nil
}

func (a addEmqxStatefulSet) canBeScaledDown(enterprise *appsv1beta4.EmqxEnterprise, originSts *appsv1.StatefulSet, podMap map[types.UID][]*corev1.Pod) bool {
	// Check if there are any nodes that are prohibiting and has 0 current connected and current sessions
	for _, e := range enterprise.Status.EmqxBlueGreenUpdateStatus.EvacuationsStatus {
		if *e.Stats.CurrentConnected == 0 && *e.Stats.CurrentSessions == 0 && e.State == "prohibiting" {
			// Extract the pod name from the node string
			podName := extractPodName(e.Node)

			// Check if pod belongs to the given StatefulSet
			if strings.Contains(podName, originSts.Name) {
				pods := podMap[originSts.UID]

				// Get the latest pod for the StatefulSet
				sort.Sort(PodsByNameNewer(pods))
				if pods[0].Name == podName {
					// Record event
					a.EventRecorder.Event(enterprise, corev1.EventTypeNormal, "Evacuate", fmt.Sprintf("Evacuated node %s successfully", getEmqxNodeName(enterprise, pods[0])))
					return true
				}
			}
		}
	}
	return false
}

// Request API
func (a addEmqxStatefulSet) startEvacuateNodeByAPI(instance appsv1beta4.Emqx, migrateToPods []*corev1.Pod, nodeName string) error {
	enterprise, ok := instance.(*appsv1beta4.EmqxEnterprise)
	if !ok {
		return emperror.New("failed to evacuate node, only support emqx enterprise")
	}

	migrateTo := []string{}
	for _, pod := range migrateToPods {
		emqxNodeName := getEmqxNodeName(instance, pod)
		migrateTo = append(migrateTo, emqxNodeName)
	}

	body := map[string]interface{}{
		"conn_evict_rate": enterprise.Spec.EmqxBlueGreenUpdate.EvacuationStrategy.ConnEvictRate,
		"sess_evict_rate": enterprise.Spec.EmqxBlueGreenUpdate.EvacuationStrategy.SessEvictRate,
		"migrate_to":      migrateTo,
	}
	if enterprise.Spec.EmqxBlueGreenUpdate.EvacuationStrategy.WaitTakeover > 0 {
		body["wait_takeover"] = enterprise.Spec.EmqxBlueGreenUpdate.EvacuationStrategy.WaitTakeover
	}

	b, err := json.Marshal(body)
	if err != nil {
		return emperror.Wrap(err, "marshal body failed")
	}

	_, _, err = a.PortForwardAPI.RequestAPI("POST", "api/v4/load_rebalance/"+nodeName+"/evacuation/start", b)
	return err
}

// Extract the pod name from the node string
func extractPodName(node string) string {
	podName := strings.Split(strings.Split(node, "@")[1], ".")[0]
	return podName
}
