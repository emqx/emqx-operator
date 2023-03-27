package v1beta4

import (
	"context"
	"encoding/json"

	emperror "emperror.dev/errors"
	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	innerPortFW "github.com/emqx/emqx-operator/internal/portforward"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type updatePodConditions struct {
	*EmqxReconciler
}

func (u updatePodConditions) reconcile(ctx context.Context, instance appsv1beta4.Emqx, _ ...any) subResult {
	pods := &corev1.PodList{}
	_ = u.Client.List(ctx, pods,
		client.InNamespace(instance.GetNamespace()),
		client.MatchingLabels(instance.GetSpec().GetTemplate().Labels),
	)
	clusterPods := make(map[string]struct{})
	for _, emqx := range instance.GetStatus().GetEmqxNodes() {
		podName := extractPodName(emqx.Node)
		clusterPods[podName] = struct{}{}
	}

	for _, pod := range pods.Items {
		if _, ok := clusterPods[pod.Name]; !ok {
			continue
		}

		hash := make(map[corev1.PodConditionType]corev1.ConditionStatus)

		for _, condition := range pod.Status.Conditions {
			hash[condition.Type] = condition.Status
		}

		if s, ok := hash[corev1.PodReady]; !ok || s != corev1.ConditionTrue {
			continue
		}

		onServerCondition := corev1.PodCondition{
			Type:   appsv1beta4.PodOnServing,
			Status: corev1.ConditionFalse,
		}

		if _, ok := hash[appsv1beta4.PodOnServing]; !ok {
			onServerCondition.LastTransitionTime = metav1.Now()
		}

		if h, ok := pod.Labels["controller-revision-hash"]; ok && h == instance.GetStatus().GetCurrentStatefulSetVersion() {
			onServerCondition.Status = corev1.ConditionTrue
			if enterprise, ok := instance.(*appsv1beta4.EmqxEnterprise); ok {
				s, err := u.checkRebalanceStatus(enterprise, pod.DeepCopy())
				if err != nil {
					return subResult{err: err}
				}
				onServerCondition.Status = s
			}
		}

		patchBytes, _ := json.Marshal(corev1.Pod{
			Status: corev1.PodStatus{
				Conditions: []corev1.PodCondition{onServerCondition},
			},
		})
		err := u.Client.Status().Patch(ctx, &pod, client.RawPatch(types.StrategicMergePatchType, patchBytes))
		if err != nil {
			return subResult{err: emperror.Wrap(err, "failed to patch pod conditions")}
		}
	}
	return subResult{}
}

func (u updatePodConditions) getPortForwardAPI(instance appsv1beta4.Emqx, pod *corev1.Pod) (*portForwardAPI, error) {
	o, err := innerPortFW.NewPortForwardOptions(u.Clientset, u.Config, pod, "8081")
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create port forward options")
	}

	username, password, err := getBootstrapUser(context.Background(), u.Client, instance)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to get bootstrap user")
	}
	return &portForwardAPI{
		Username: username,
		Password: password,
		Options:  o,
	}, nil
}

func (u updatePodConditions) checkRebalanceStatus(instance *appsv1beta4.EmqxEnterprise, pod *corev1.Pod) (corev1.ConditionStatus, error) {
	p, err := u.getPortForwardAPI(instance, pod)
	if err != nil {
		return corev1.ConditionUnknown, emperror.Wrap(err, "failed to get portForwardAPI")
	}
	if p == nil {
		return corev1.ConditionUnknown, emperror.New("portForwardAPI is nil")
	}
	defer close(p.Options.StopChannel)
	if err := p.Options.ForwardPorts(); err != nil {
		return corev1.ConditionUnknown, emperror.Wrap(err, "failed to forward ports")
	}

	resp, _, err := p.requestAPI("GET", "api/v4/load_rebalance/availability_check", nil)
	if err != nil {
		return corev1.ConditionUnknown, emperror.Wrap(err, "failed to check pod availability")
	}
	if resp == nil || resp.StatusCode != 200 {
		return corev1.ConditionFalse, emperror.Errorf("pod %s-%s is unAvailable", pod.Namespace, pod.Name)
	}
	return corev1.ConditionTrue, nil
}
