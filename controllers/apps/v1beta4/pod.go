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
	innerErr "github.com/emqx/emqx-operator/internal/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type EmqxHttpAPI interface {
	GetUsername() string
	GetPassword() string
	Request(method, path string, body []byte) (resp *http.Response, respBody []byte, err error)
	GetPod() *corev1.Pod
}

type emqxHttpAPI struct {
	Username string
	Password string
	Pod      *corev1.Pod
}

func newEmqxHttpAPI(client client.Client, instance appsv1beta4.Emqx, pod *corev1.Pod) (*emqxHttpAPI, error) {
	username, password, err := getBootstrapUser(context.Background(), client, instance)
	if err != nil {
		return nil, err
	}
	if pod == nil {
		pod, err = getReadyPod(client, instance)
		if pod == nil || err != nil {
			return nil, err
		}
	}
	return &emqxHttpAPI{
		Username: username,
		Password: password,
		Pod:      pod,
	}, nil
}

func getReadyPod(client client.Client, instance appsv1beta4.Emqx) (*corev1.Pod, error) {
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
				return pod, nil
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

func (e *emqxHttpAPI) GetUsername() string {
	return e.Username
}

func (e *emqxHttpAPI) GetPassword() string {
	return e.Password
}

func (e *emqxHttpAPI) GetPod() *corev1.Pod {
	return e.Pod
}

func (e *emqxHttpAPI) Request(method, path string, body []byte) (resp *http.Response, respBody []byte, err error) {
	if e == nil || e.GetPod() == nil {
		return nil, nil, emperror.Errorf("failed to request %s %s, emqxHttpAPI is not ready", method, path)
	}
	url := url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%d", e.GetPod().Status.PodIP, 8081),
		Path:   path,
	}

	httpClient := http.Client{}
	req, err := http.NewRequest(method, url.String(), bytes.NewReader(body))
	if err != nil {
		return nil, nil, emperror.Wrap(err, "failed to create request")
	}
	req.SetBasicAuth(e.GetUsername(), e.GetPassword())
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
