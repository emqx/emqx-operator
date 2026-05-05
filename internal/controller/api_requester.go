package controller

import (
	"net"
	"strconv"
	"strings"

	emperror "emperror.dev/errors"
	config "github.com/emqx/emqx-operator/internal/controller/config"
	resources "github.com/emqx/emqx-operator/internal/controller/resources"
	req "github.com/emqx/emqx-operator/internal/requester"
	corev1 "k8s.io/api/core/v1"
)

type apiRequesterBuilder struct {
	schema   string
	port     string
	username string
	password string
}

type apiRequester interface {
	forOldestCore(state *reconcileState, filter ...reconcileStatePodFilter) req.RequesterInterface
	forPod(pod *corev1.Pod) req.RequesterInterface
}

func (b *apiRequesterBuilder) forOldestCore(
	state *reconcileState,
	filter ...reconcileStatePodFilter,
) req.RequesterInterface {
	pods := state.listPods(podsWithRole{roleCore})
	sortByCreationTimestamp(pods)
outer:
	for _, pod := range pods {
		req := b.forPod(pod)
		if req == nil {
			continue
		}
		for _, f := range filter {
			if !f.passes(pod) {
				continue outer
			}
		}
		return req
	}
	return nil
}

func (b *apiRequesterBuilder) forPod(pod *corev1.Pod) req.RequesterInterface {
	if b == nil {
		return nil
	}
	if pod.Status.PodIP == "" || pod.Status.Phase != corev1.PodRunning {
		return nil
	}
	return &req.Requester{
		Schema:      b.schema,
		Host:        net.JoinHostPort(pod.Status.PodIP, b.port),
		Username:    b.username,
		Password:    b.password,
		Description: pod.Name,
	}
}

func newAPIRequesterBuilder(
	conf *config.EMQX,
	bootstrapAPIKey *corev1.Secret,
) (*apiRequesterBuilder, error) {
	username, password, err := getAPICredentials(bootstrapAPIKey)
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
	return &apiRequesterBuilder{
		schema:   schema,
		port:     port,
		username: username,
		password: password,
	}, nil
}

func getAPICredentials(bootstrapAPIKey *corev1.Secret) (string, string, error) {
	if data, ok := bootstrapAPIKey.Data["bootstrap_api_key"]; ok {
		users := strings.Split(string(data), "\n")
		for _, user := range users {
			index := strings.Index(user, ":")
			if index > 0 && user[:index] == resources.DefaultBootstrapAPIKey {
				username := user[:index]
				password := user[index+1:]
				return username, password, nil
			}
		}
	}

	return "", "", emperror.New("secret does not contain `bootstrap_api_key`")
}
