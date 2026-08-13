package controller

import (
	"cmp"
	"net"
	"slices"
	"strconv"
	"strings"

	emperror "emperror.dev/errors"
	crd "github.com/emqx/emqx-operator/api/v3beta1"
	config "github.com/emqx/emqx-operator/internal/controller/config"
	resources "github.com/emqx/emqx-operator/internal/controller/resources"
	util "github.com/emqx/emqx-operator/internal/controller/util"
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
	forCore(state *reconcileState, filter ...reconcileStatePodFilter) req.RequesterInterface
	forPod(pod *corev1.Pod) req.RequesterInterface
}

func (b *apiRequesterBuilder) forCore(
	state *reconcileState,
	filter ...reconcileStatePodFilter,
) req.RequesterInterface {
	pods := preferredCorePods(state, filter...)
	for _, pod := range pods {
		req := b.forPod(pod)
		if req == nil {
			continue
		}
		return req
	}
	return nil
}

func preferredCorePods(
	state *reconcileState,
	filter ...reconcileStatePodFilter,
) []*corev1.Pod {
	filter = append(
		filter,
		podsWithRole{crd.RoleCore},
		podsWithCondition{corev1.ContainersReady},
		podsAlive{},
	)
	pods := state.listPods(filter...)
	sortByPreference(state, pods)
	return pods
}

// Prefer lower ordinals by default. If exactly one outdated core remains,
// deprioritize it because syncCoreSet will recreate it next.
func sortByPreference(state *reconcileState, pods []*corev1.Pod) {
	outdated := state.listOutdatedPods()
	slices.SortFunc(pods, func(a, b *corev1.Pod) int {
		ai := util.PodOrdinal(a.Name)
		bi := util.PodOrdinal(b.Name)
		if len(outdated) == 1 {
			if slices.Contains(outdated, a) {
				ai += 10000
			}
			if slices.Contains(outdated, b) {
				bi += 10000
			}
		}
		return cmp.Compare(ai, bi)
	})
}

func (b *apiRequesterBuilder) forPod(pod *corev1.Pod) req.RequesterInterface {
	if b == nil {
		return nil
	}
	if pod.Status.PodIP == "" {
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
