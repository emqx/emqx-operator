package controller

import (
	"net"
	"reflect"
	"strconv"
	"strings"

	emperror "emperror.dev/errors"
	crd "github.com/emqx/emqx-operator/api/v3alpha1"
	config "github.com/emqx/emqx-operator/internal/controller/config"
	resources "github.com/emqx/emqx-operator/internal/controller/resources"
	util "github.com/emqx/emqx-operator/internal/controller/util"
	req "github.com/emqx/emqx-operator/internal/requester"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type apiRequesterBuilder struct {
	schema   string
	port     string
	username string
	password string
}

type apiRequester interface {
	forOldestCore(state *reconcileState, filter ...podRequesterFilter) req.RequesterInterface
	forPod(pod *corev1.Pod) req.RequesterInterface
}

type podRequesterFilter interface {
	filter(pod *corev1.Pod) bool
}

type podConditionFilter struct {
	cond corev1.PodConditionType
}

func (f *podConditionFilter) filter(pod *corev1.Pod) bool {
	return util.IsPodConditionTrue(pod, f.cond)
}

type managedByFilter struct {
	manager metav1.Object
}

func (f *managedByFilter) filter(pod *corev1.Pod) bool {
	controller := metav1.GetControllerOf(pod)
	if controller == nil {
		return false
	}
	if f.manager == nil || reflect.ValueOf(f.manager).IsNil() {
		return false
	}
	return controller.UID == f.manager.GetUID()
}

type emqxVersionFilter struct {
	instance *crd.EMQX
	prefix   string
}

func (f *emqxVersionFilter) filter(pod *corev1.Pod) bool {
	node := f.instance.Status.FindNodeByPodName(pod.Name)
	return node != nil && strings.HasPrefix(node.Version, f.prefix)
}

func (b *apiRequesterBuilder) forOldestCore(
	state *reconcileState,
	filter ...podRequesterFilter,
) req.RequesterInterface {
	pods := state.podsWithRole("core")
	sortByCreationTimestamp(pods)
outer:
	for _, pod := range pods {
		req := b.forPod(pod)
		if req == nil {
			continue
		}
		for _, f := range filter {
			if !f.filter(pod) {
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
