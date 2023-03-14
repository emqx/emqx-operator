package v1beta4

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	emperror "emperror.dev/errors"
	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type addEmqxResources struct {
	*EmqxReconciler
	*portForwardAPI
}

func (a addEmqxResources) reconcile(ctx context.Context, instance appsv1beta4.Emqx, args ...any) subResult {
	initResources, ok := args[0].([]client.Object)
	if !ok {
		panic("args[0] is not []client.Object")
	}

	// ignore error, because if statefulSet is not created, the listener port will be not found
	listenerPorts, _ := a.getListenerPortsByAPI()

	var resources []client.Object

	license, err := a.getLicense(ctx, instance)
	if err != nil {
		return subResult{err: emperror.Wrap(err, "failed to get license")}
	}
	if license != nil {
		resources = append(resources, license)
	}

	acl := generateEmqxACL(instance)
	resources = append(resources, acl)

	headlessSvc := generateHeadlessService(instance, listenerPorts...)
	resources = append(resources, headlessSvc)

	svc := generateService(instance, listenerPorts...)
	if svc != nil {
		resources = append(resources, svc)
	}

	if err := a.CreateOrUpdateList(instance, a.Scheme, resources); err != nil {
		return subResult{err: emperror.Wrap(err, "failed to create or update resource")}
	}

	sts := generateStatefulSet(instance)
	sts = updateStatefulSetForACL(sts, acl)
	sts = updateStatefulSetForLicense(sts, license)

	names := appsv1beta4.Names{Object: instance}
	for _, initResource := range initResources {
		if initResource.GetName() == names.BootstrapUser() {
			bootstrapUser := initResource.(*corev1.Secret)
			sts = updateStatefulSetForBootstrapUser(sts, bootstrapUser)
		}
		if initResource.GetName() == names.PluginsConfig() {
			pluginsConfig := initResource.(*corev1.ConfigMap)
			sts = updateStatefulSetForPluginsConfig(sts, pluginsConfig)
		}
	}

	return subResult{args: sts}
}

func (a addEmqxResources) getLicense(ctx context.Context, instance appsv1beta4.Emqx) (*corev1.Secret, error) {
	enterprise, ok := instance.(*appsv1beta4.EmqxEnterprise)
	if !ok {
		return nil, nil
	}

	if enterprise.Spec.License.SecretName != "" {
		license := &corev1.Secret{}
		if err := a.Client.Get(
			ctx,
			types.NamespacedName{
				Name:      enterprise.Spec.License.SecretName,
				Namespace: enterprise.GetNamespace(),
			},
			license,
		); err != nil {
			return nil, err
		}
		return license, nil
	}
	return generateLicense(instance), nil
}

func (a addEmqxResources) getListenerPortsByAPI() ([]corev1.ServicePort, error) {
	type emqxListener struct {
		Protocol string `json:"protocol"`
		ListenOn string `json:"listen_on"`
	}

	type emqxListeners struct {
		Node      string         `json:"node"`
		Listeners []emqxListener `json:"listeners"`
	}

	intersection := func(listeners1 []emqxListener, listeners2 []emqxListener) []emqxListener {
		hSection := map[string]struct{}{}
		ans := make([]emqxListener, 0)
		for _, listener := range listeners1 {
			hSection[listener.ListenOn] = struct{}{}
		}
		for _, listener := range listeners2 {
			_, ok := hSection[listener.ListenOn]
			if ok {
				ans = append(ans, listener)
				delete(hSection, listener.ListenOn)
			}
		}
		return ans
	}

	_, body, err := a.portForwardAPI.requestAPI("GET", "api/v4/listeners", nil)
	if err != nil {
		return nil, err
	}

	listenerList := []emqxListeners{}
	data := gjson.GetBytes(body, "data")
	if err := json.Unmarshal([]byte(data.Raw), &listenerList); err != nil {
		return nil, emperror.Wrap(err, "failed to unmarshal node statuses")
	}

	var listeners []emqxListener
	if len(listenerList) == 1 {
		listeners = listenerList[0].Listeners
	} else {
		for i := 0; i < len(listenerList)-1; i++ {
			listeners = intersection(listenerList[i].Listeners, listenerList[i+1].Listeners)
		}
	}

	ports := []corev1.ServicePort{}
	for _, l := range listeners {
		var name string
		var protocol corev1.Protocol
		var strPort string
		var intPort int

		compile := regexp.MustCompile(".*(udp|dtls|sn).*")
		if compile.MatchString(l.Protocol) {
			protocol = corev1.ProtocolUDP
		} else {
			protocol = corev1.ProtocolTCP
		}

		if strings.Contains(l.ListenOn, ":") {
			_, strPort, err = net.SplitHostPort(l.ListenOn)
			if err != nil {
				strPort = l.ListenOn
			}
		} else {
			strPort = l.ListenOn
		}
		intPort, _ = strconv.Atoi(strPort)

		// Get name by protocol and port from API
		// protocol maybe like mqtt:wss:8084
		// protocol maybe like mqtt:tcp
		// We had to do something with the "protocol" to make it conform to the kubernetes service port name specification
		name = regexp.MustCompile(`:[\d]+`).ReplaceAllString(l.Protocol, "")
		name = strings.ReplaceAll(name, ":", "-")
		name = fmt.Sprintf("%s-%s", name, strPort)

		ports = append(ports, corev1.ServicePort{
			Name:       name,
			Protocol:   protocol,
			Port:       int32(intPort),
			TargetPort: intstr.FromInt(intPort),
		})
	}
	return ports, nil
}
