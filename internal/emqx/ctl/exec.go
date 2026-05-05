package ctl

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"slices"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/httpstream"
	client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

type ctlError struct {
	stderr string
	inner  error
}

func (e ctlError) Error() string {
	if e.stderr != "" {
		return fmt.Sprintf("emqx ctl failed: %s: %s", e.stderr, e.inner.Error())
	}
	return fmt.Sprintf("emqx ctl failed: %s", e.inner.Error())
}

func Ctl(
	ctx context.Context,
	restConfig *rest.Config,
	pod *corev1.Pod,
	args ...string,
) error {
	command := slices.Concat([]string{"emqx", "ctl"}, args)
	_, stderr, err := Execute(ctx, restConfig, pod, command)
	if err != nil {
		return ctlError{stderr, err}
	}
	return nil
}

func Execute(
	ctx context.Context,
	restConfig *rest.Config,
	pod *corev1.Pod,
	command []string,
) (string, string, error) {
	clientset, err := client.NewForConfig(restConfig)
	if err != nil {
		return "", "", err
	}

	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec")
	req.VersionedParams(&corev1.PodExecOptions{
		Container: crd.DefaultContainerName,
		Command:   command,
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}, scheme.ParameterCodec)

	var stdout, stderr bytes.Buffer
	err = execute(ctx, restConfig, req.URL(), &stdout, &stderr)

	return stdout.String(), stderr.String(), err
}

func execute(ctx context.Context, restConfig *rest.Config, url *url.URL, stdout, stderr io.Writer) error {
	// WebSocketExecutor must be "GET" method as described in RFC 6455 Sec. 4.1 (page 17).
	websocketExec, err := remotecommand.NewWebSocketExecutor(restConfig, "GET", url.String())
	if err != nil {
		return err
	}
	spdyExec, err := remotecommand.NewSPDYExecutor(restConfig, "POST", url)
	if err != nil {
		return err
	}
	exec, err := remotecommand.NewFallbackExecutor(websocketExec, spdyExec, func(err error) bool {
		if httpstream.IsUpgradeFailure(err) || httpstream.IsHTTPSProxyError(err) {
			return true
		}
		return false
	})
	if err != nil {
		return err
	}

	return exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: stdout,
		Stderr: stderr,
		Tty:    false,
	})
}
