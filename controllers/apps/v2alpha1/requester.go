package v2alpha1

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	emperror "emperror.dev/errors"
	"github.com/emqx/emqx-operator/apis/apps/v2alpha1"
	appsv2alpha1 "github.com/emqx/emqx-operator/apis/apps/v2alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Requester interface {
	GetHost() string
	GetUsername() string
	GetPassword() string
	Request(method, path string, body []byte) (resp *http.Response, respBody []byte, err error)
}

type requester struct {
	Host     string
	Username string
	Password string
}

func newRequesterBySvc(client client.Client, instance *appsv2alpha1.EMQX) (*requester, error) {
	username, password, err := getBootstrapUser(context.Background(), client, instance)
	if err != nil {
		return nil, err
	}

	headlessService := instance.HeadlessServiceNamespacedName()

	var port string
	dashboardPort, err := appsv2alpha1.GetDashboardServicePort(instance)
	if err != nil || dashboardPort == nil {
		port = "18083"
	}

	if dashboardPort != nil {
		port = dashboardPort.TargetPort.String()
	}

	return &requester{
		// TODO: the telepersence is not support `$service.$namespace.svc` format in Linux
		// Host:     fmt.Sprintf("%s.%s.svc:%s", headlessService.Name, headlessService.Namespace, port),
		Host:     fmt.Sprintf("%s.%s.svc.cluster.local:%s", headlessService.Name, headlessService.Namespace, port),
		Username: username,
		Password: password,
	}, nil
}

func (requester *requester) GetUsername() string {
	return requester.Username
}

func (requester *requester) GetPassword() string {
	return requester.Password
}

func (requester *requester) GetHost() string {
	return requester.Host
}

func (requester *requester) Request(method, path string, body []byte) (resp *http.Response, respBody []byte, err error) {
	url := url.URL{
		Scheme: "http",
		Host:   requester.GetHost(),
		Path:   path,
	}

	httpClient := http.Client{}
	req, err := http.NewRequest(method, url.String(), bytes.NewReader(body))
	if err != nil {
		return nil, nil, emperror.Wrap(err, "failed to create request")
	}
	req.SetBasicAuth(requester.GetUsername(), requester.GetPassword())
	req.Close = true
	resp, err = httpClient.Do(req)
	if err != nil {
		return nil, nil, emperror.NewWithDetails("failed to request API", "method", method, "path", url.Path)
	}

	defer resp.Body.Close()
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, emperror.Wrap(err, "failed to read response body")
	}
	return resp, body, nil
}

func getBootstrapUser(ctx context.Context, client client.Client, instance *v2alpha1.EMQX) (username, password string, err error) {
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
