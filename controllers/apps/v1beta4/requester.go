package v1beta4

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	emperror "emperror.dev/errors"
	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
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

func newRequesterByPod(client client.Client, instance appsv1beta4.Emqx, pod *corev1.Pod) (*requester, error) {
	username, password, err := getBootstrapUser(context.Background(), client, instance)
	if err != nil {
		return nil, err
	}

	return &requester{
		Host:     fmt.Sprintf("%s:8081", pod.Status.PodIP),
		Username: username,
		Password: password,
	}, nil
}

func newRequesterBySvc(client client.Client, instance appsv1beta4.Emqx) (*requester, error) {
	username, password, err := getBootstrapUser(context.Background(), client, instance)
	if err != nil {
		return nil, err
	}

	names := appsv1beta4.Names{Object: instance}
	return &requester{
		// Host:     fmt.Sprintf("%s.%s.svc:8081", names.HeadlessSvc(), instance.GetNamespace()),
		Host:     fmt.Sprintf("%s.%s.svc.cluster.local:8081", names.HeadlessSvc(), instance.GetNamespace()),
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
		return nil, nil, emperror.Wrap(err, "failed to request API")
	}

	defer resp.Body.Close()
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, emperror.Wrap(err, "failed to read response body")
	}
	return resp, body, nil
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
