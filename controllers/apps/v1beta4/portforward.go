package v1beta4

import (
	"context"
	"net/http"
	"strings"

	emperror "emperror.dev/errors"
	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	innerErr "github.com/emqx/emqx-operator/internal/errors"
	innerPortFW "github.com/emqx/emqx-operator/internal/portforward"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PortForwardAPI interface {
	GetUsername() string
	GetPassword() string
	GetOptions() *innerPortFW.PortForwardOptions
	RequestAPI(method, path string, body []byte) (resp *http.Response, respBody []byte, err error)
}

// portForwardAPI provides a wrapper around the port-forward API.
type portForwardAPI struct {
	Username string
	Password string
	Options  *innerPortFW.PortForwardOptions
}

func newPortForwardAPI(ctx context.Context, client client.Client, clientset *kubernetes.Clientset, config *rest.Config, instance appsv1beta4.Emqx) (*portForwardAPI, error) {
	options, err := newPortForwardOptions(client, clientset, config, instance)
	if err != nil {
		return nil, err
	}
	if options == nil {
		return nil, nil
	}
	username, password, err := getBootstrapUser(context.Background(), client, instance)
	if err != nil {
		return nil, err
	}
	return &portForwardAPI{
		Username: username,
		Password: password,
		Options:  options,
	}, nil
}

func newPortForwardOptions(client client.Client, clientset *kubernetes.Clientset, config *rest.Config, instance appsv1beta4.Emqx) (*innerPortFW.PortForwardOptions, error) {
	list, err := getInClusterStatefulSets(client, instance)
	if err != nil {
		if !emperror.Is(err, innerErr.ErrStsNotReady) {
			return nil, emperror.Wrap(err, "failed to get statefulSet")
		}
		if list, err = getAllStatefulSet(client, instance); err != nil {
			return nil, emperror.Wrap(err, "failed to get statefulSet")
		}
	}

	sts := list[len(list)-1]
	podMap, err := getPodMap(client, instance, []*appsv1.StatefulSet{sts})
	if err != nil {
		return nil, err
	}
	if len(podMap[sts.UID]) == 0 {
		return nil, emperror.Wrap(innerErr.ErrPodNotReady, "failed to get pod")
	}

	for _, pod := range podMap[sts.UID] {
		for _, c := range pod.Status.Conditions {
			if c.Type == corev1.ContainersReady && c.Status == corev1.ConditionTrue {
				o, err := innerPortFW.NewPortForwardOptions(clientset, config, pod, "8081")
				if err != nil {
					return nil, emperror.Wrap(err, "failed to create port forward")
				}
				return o, err
			}
		}
	}
	return nil, nil
}

func getBootstrapUser(ctx context.Context, client client.Client, instance appsv1beta4.Emqx) (username, password string, err error) {
	bootstrapUser := &corev1.Secret{}
	if err = client.Get(ctx, types.NamespacedName{
		Namespace: instance.GetNamespace(),
		Name:      instance.GetName() + "-bootstrap-user",
	}, bootstrapUser); err != nil {
		err = emperror.Wrap(err, "get secret failed")
		return
	}

	if data, ok := bootstrapUser.Data["bootstrap_user"]; ok {
		users := strings.Split(string(data), "\n")
		for _, user := range users {
			index := strings.Index(user, ":")
			if index > 0 && user[:index] == defUsername {
				username = user[:index]
				password = user[index+1:]
				return
			}
		}
	}

	err = emperror.Errorf("the secret does not contain the bootstrap_user")
	return
}

func (p *portForwardAPI) GetUsername() string {
	return p.Username
}

func (p *portForwardAPI) GetPassword() string {
	return p.Password
}

func (p *portForwardAPI) GetOptions() *innerPortFW.PortForwardOptions {
	return p.Options
}

func (p *portForwardAPI) RequestAPI(method, path string, body []byte) (resp *http.Response, respBody []byte, err error) {
	if p == nil {
		return nil, nil, emperror.Errorf("failed to %s %s, portForward is not ready", method, path)
	}
	return p.Options.RequestAPI(p.Username, p.Password, method, path, body)
}
