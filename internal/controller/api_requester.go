package controller

import (
	"context"
	"net"
	"sort"
	"strconv"
	"strings"

	emperror "emperror.dev/errors"
	appsv2beta1 "github.com/emqx/emqx-operator/api/v2beta1"
	config "github.com/emqx/emqx-operator/internal/controller/config"
	req "github.com/emqx/emqx-operator/internal/requester"
	corev1 "k8s.io/api/core/v1"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
)

func apiRequester(
	ctx context.Context,
	client k8s.Client,
	instance *appsv2beta1.EMQX,
	conf *config.Conf,
) (req.RequesterInterface, error) {
	username, password, err := getBootstrapAPIKey(ctx, client, instance)
	if err != nil {
		return nil, err
	}

	var schema, port string
	portMap := conf.GetDashboardPortMap()
	if dashboardHttps, ok := portMap["dashboard-https"]; ok {
		schema = "https"
		port = strconv.Itoa(dashboardHttps)
	}
	if dashboard, ok := portMap["dashboard"]; ok {
		schema = "http"
		port = strconv.Itoa(dashboard)
	}

	podList := &corev1.PodList{}
	_ = client.List(ctx, podList,
		k8s.InNamespace(instance.Namespace),
		k8s.MatchingLabels(appsv2beta1.DefaultCoreLabels(instance)),
	)
	sort.Slice(podList.Items, func(i, j int) bool {
		return podList.Items[i].CreationTimestamp.Before(&podList.Items[j].CreationTimestamp)
	})
	for _, pod := range podList.Items {
		if pod.GetDeletionTimestamp() == nil && pod.Status.PodIP != "" {
			cond := appsv2beta1.FindPodCondition(&pod, corev1.ContainersReady)
			if cond != nil && cond.Status == corev1.ConditionTrue {
				return &req.Requester{
					Schema:   schema,
					Host:     net.JoinHostPort(pod.Status.PodIP, port),
					Username: username,
					Password: password,
				}, nil
			}
		}
	}
	return nil, nil
}

func getBootstrapAPIKey(ctx context.Context, client k8s.Client, instance *appsv2beta1.EMQX) (username, password string, err error) {
	bootstrapAPIKey := &corev1.Secret{}
	if err = client.Get(ctx, instance.BootstrapAPIKeyNamespacedName(), bootstrapAPIKey); err != nil {
		err = emperror.Wrap(err, "get secret failed")
		return
	}

	if data, ok := bootstrapAPIKey.Data["bootstrap_api_key"]; ok {
		users := strings.Split(string(data), "\n")
		for _, user := range users {
			index := strings.Index(user, ":")
			if index > 0 && user[:index] == appsv2beta1.DefaultBootstrapAPIKey {
				username = user[:index]
				password = user[index+1:]
				return
			}
		}
	}

	err = emperror.Errorf("the secret does not contain the bootstrap_api_key")
	return
}
