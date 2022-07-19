/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package apps

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	appsv1beta3 "github.com/emqx/emqx-operator/apis/apps/v1beta3"
	"github.com/emqx/emqx-operator/pkg/handler"
	"github.com/tidwall/gjson"
)

var _ reconcile.Reconciler = &EmqxBrokerReconciler{}

const (
	ReloaderContainerName  = "reloader"
	ReloaderContainerImage = "emqx/emqx-operator-reloader:0.0.1"
)

type EmqxReconciler struct {
	handler.Handler
	Scheme *runtime.Scheme
	record.EventRecorder
}

func (r *EmqxReconciler) Do(ctx context.Context, instance appsv1beta3.Emqx) (ctrl.Result, error) {
	var resources []client.Object
	var sts *appsv1.StatefulSet
	var loadedPlugins, defaultPluginsConfig *corev1.ConfigMap
	var err error
	var postFn func(client.Object) error
	sts = generateStatefulSetDef(instance)
	postFn = func(client.Object) error { return nil }
	loadedPlugins, _ = generateLoadedPlugins(instance, sts)
	defaultPluginsConfig, _ = generateDefaultPluginsConfig(instance, sts)
	if !isPluginInitialized(instance) {
		resources = append(resources, defaultPluginsConfig, loadedPlugins)
		pluginsList := &appsv1beta3.EmqxPluginList{}
		err = r.Client.List(ctx, pluginsList, client.InNamespace(instance.GetNamespace()))
		if err != nil && !k8sErrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		var condition *appsv1beta3.Condition
		pluginResourceList := generateInitPluginList(instance, pluginsList)
		resources = append(resources, pluginResourceList...)

		err = r.CreateOrUpdateList(instance, r.Scheme, resources, postFn)
		if err != nil {
			r.EventRecorder.Event(instance, corev1.EventTypeWarning, "FailedCreateOrUpdate", err.Error())
			condition = appsv1beta3.NewCondition(
				appsv1beta3.ConditionPluginInitialized,
				corev1.ConditionFalse,
				"PluginInitializeFailed",
				err.Error(),
			)
			instance.SetCondition(*condition)
			_ = r.Status().Update(ctx, instance)
			return ctrl.Result{RequeueAfter: time.Duration(5) * time.Second}, err
		}
		condition = appsv1beta3.NewCondition(
			appsv1beta3.ConditionPluginInitialized,
			corev1.ConditionTrue,
			"PluginInitializeSuccessfully",
			"All default plugins initialized",
		)
		instance.SetCondition(*condition)
		_ = r.Status().Update(ctx, instance)
		return ctrl.Result{RequeueAfter: time.Duration(5) * time.Second}, nil

	}
	//add acl
	acl, _ := generateAcl(instance, sts)
	resources = append(resources, acl)

	//add module
	module, _ := generateLoadedModules(instance, sts)
	resources = append(resources, module)

	if license, _ := generateLicense(instance, sts); license != nil {
		resources = append(resources, license)
	}

	serviceTemplate := instance.GetServiceTemplate()
	serviceTemplate.MergePorts(r.getListenerPortsByAPI(instance))
	instance.SetServiceTemplate(serviceTemplate)

	headlessSvc, svc, _ := generateSvc(instance, sts)
	resources = append(resources, headlessSvc, svc)

	//add reloader container
	reloaderContainer := generateReloaderContainer(r, instance, sts)
	sts.Spec.Template.Spec.Containers = append(sts.Spec.Template.Spec.Containers, reloaderContainer)
	// add container annotation
	if sts.Annotations == nil {
		sts.Annotations = map[string]string{}
	}
	sts.Annotations[handler.ManageContainersAnnotation] = generateAnnotationByContainers(sts.Spec.Template.Spec.Containers)
	// StateFulSet should be created last
	resources = append(resources, sts)
	if err := r.CreateOrUpdateList(instance, r.Scheme, resources, postFn); err != nil {
		r.EventRecorder.Event(instance, corev1.EventTypeWarning, "FailedCreateOrUpdate", err.Error())
		return ctrl.Result{}, err
	}
	condition := r.getClusterStatusByAPI(instance)
	instance.SetCondition(*condition)
	if err = r.Status().Update(ctx, instance); err != nil {
		return ctrl.Result{}, err
	}
	if !(condition.Type == appsv1beta3.ConditionRunning && condition.Status == corev1.ConditionTrue) {
		return ctrl.Result{RequeueAfter: time.Duration(5) * time.Second}, nil
	}
	return ctrl.Result{RequeueAfter: time.Duration(20) * time.Second}, nil
}

func (r *EmqxReconciler) getManagementField(obj appsv1beta3.Emqx) (username, password, apiPort string) {
	username = "admin"
	password = "public"
	apiPort = "8081"

	pluginsList := &appsv1beta3.EmqxPluginList{}
	_ = r.Client.List(context.TODO(), pluginsList, client.InNamespace(obj.GetNamespace()))

	for _, plugin := range pluginsList.Items {
		selector, _ := labels.ValidatedSelectorFromSet(plugin.Spec.Selector)
		if selector.Empty() || !selector.Matches(labels.Set(obj.GetLabels())) {
			continue
		}
		if plugin.Spec.PluginName == "emqx_management" {
			if _, ok := plugin.Spec.Config["management.listener.http"]; ok {
				apiPort = plugin.Spec.Config["management.listener.http"]
			}
			if _, ok := plugin.Spec.Config["management.default_application.id"]; ok {
				username = plugin.Spec.Config["management.default_application.id"]
			}
			if _, ok := plugin.Spec.Config["management.default_application.secret"]; ok {
				password = plugin.Spec.Config["management.default_application.secret"]
			}
		}
	}

	return
}

func (r *EmqxReconciler) getListenerPortsByAPI(instance appsv1beta3.Emqx) []corev1.ServicePort {
	username, password, apiPort := r.getManagementField(instance)
	resp, body, err := r.Handler.RequestAPI(instance, "GET", username, password, apiPort, "api/v4/listeners")
	if err != nil {
		return nil
	}
	if resp.StatusCode != 200 {
		return nil
	}

	ports := []corev1.ServicePort{}
	listeners := gjson.GetBytes(body, "data.0.listeners")
	for _, l := range listeners.Array() {
		var name string
		var protocol corev1.Protocol
		var strPort string
		var intPort int

		compile := regexp.MustCompile(".*(udp|dtls|sn).*")
		proto := gjson.Get(l.Raw, "protocol").String()
		if compile.MatchString(proto) {
			protocol = corev1.ProtocolUDP
		} else {
			protocol = corev1.ProtocolTCP
		}

		listenOn := gjson.Get(l.Raw, "listen_on").String()
		if strings.Contains(listenOn, ":") {
			_, strPort, err = net.SplitHostPort(listenOn)
			if err != nil {
				strPort = listenOn
			}
		} else {
			strPort = listenOn
		}
		intPort, _ = strconv.Atoi(strPort)

		// Get name by protocol and port from API
		// protocol maybe like mqtt:wss:8084
		// protocol maybe like mqtt:tcp
		// We had to do something with the "protocol" to make it conform to the kubernetes service port name specification
		name = regexp.MustCompile(`:[\d]+`).ReplaceAllString(proto, "")
		name = strings.ReplaceAll(name, ":", "-")
		name = fmt.Sprintf("%s-%s", name, strPort)

		ports = append(ports, corev1.ServicePort{
			Name:       name,
			Protocol:   protocol,
			Port:       int32(intPort),
			TargetPort: intstr.FromInt(intPort),
		})
	}
	return ports
}

func (r *EmqxReconciler) getClusterStatusByAPI(instance appsv1beta3.Emqx) *appsv1beta3.Condition {
	username, password, apiPort := r.getManagementField(instance)
	resp, body, err := r.Handler.RequestAPI(instance, "GET", username, password, apiPort, "api/v4/brokers")
	if err != nil {
		return appsv1beta3.NewCondition(
			appsv1beta3.ConditionRunning,
			corev1.ConditionFalse,
			"AccessAPIFailed",
			err.Error(),
		)
	}
	if resp.StatusCode != 200 {
		return appsv1beta3.NewCondition(
			appsv1beta3.ConditionRunning,
			corev1.ConditionFalse,
			"AccessAPIFailed",
			resp.Status,
		)
	}

	clusterData := gjson.GetBytes(body, "data.#.node")
	if len(clusterData.Array()) != int(*instance.GetReplicas()) {
		return appsv1beta3.NewCondition(
			appsv1beta3.ConditionRunning,
			corev1.ConditionFalse,
			"ClusterNotReady",
			clusterData.String(),
		)
	}
	return appsv1beta3.NewCondition(
		appsv1beta3.ConditionRunning,
		corev1.ConditionTrue,
		"ClusterReady",
		"All resources are ready",
	)
}

func generateStatefulSetDef(instance appsv1beta3.Emqx) *appsv1.StatefulSet {
	sts := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        instance.GetName(),
			Namespace:   instance.GetNamespace(),
			Labels:      instance.GetLabels(),
			Annotations: instance.GetAnnotations(),
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: instance.GetReplicas(),
			Selector: &metav1.LabelSelector{
				MatchLabels: instance.GetLabels(),
			},
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      instance.GetLabels(),
					Annotations: instance.GetAnnotations(),
				},
				Spec: corev1.PodSpec{
					Affinity:         instance.GetAffinity(),
					Tolerations:      instance.GetToleRations(),
					NodeName:         instance.GetNodeName(),
					NodeSelector:     instance.GetNodeSelector(),
					ImagePullSecrets: instance.GetImagePullSecrets(),
					SecurityContext:  instance.GetSecurityContext(),
					InitContainers:   instance.GetInitContainers(),
				},
			},
		},
	}
	emqxContainer := generateEmqxContainer(instance)
	sts.Spec.Template.Spec.Containers = append(sts.Spec.Template.Spec.Containers, emqxContainer)
	//add extra container
	sts.Spec.Template.Spec.Containers = append(sts.Spec.Template.Spec.Containers, instance.GetExtraContainers()...)
	terminationGracePeriodSeconds := int64(60)
	sts.Spec.Template.Spec.TerminationGracePeriodSeconds = &terminationGracePeriodSeconds

	return generateVolume(instance, sts)
}

func generateInitPluginList(instance appsv1beta3.Emqx, existPluginList *appsv1beta3.EmqxPluginList) []client.Object {
	matchedPluginList := []appsv1beta3.EmqxPlugin{}
	for _, existPlugin := range existPluginList.Items {
		selector, _ := labels.ValidatedSelectorFromSet(existPlugin.Spec.Selector)
		if selector.Empty() || !selector.Matches(labels.Set(instance.GetLabels())) {
			continue
		}
		matchedPluginList = append(matchedPluginList, existPlugin)
	}

	isExistPlugin := func(pluginName string, pluginList []appsv1beta3.EmqxPlugin) bool {
		for _, plugin := range pluginList {
			if plugin.Spec.PluginName == pluginName {
				return true
			}
		}
		return false
	}

	pluginList := []client.Object{}
	// Default plugins
	if !isExistPlugin("emqx_management", matchedPluginList) {
		emqxManagement := &appsv1beta3.EmqxPlugin{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apps.emqx.io/v1beta3",
				Kind:       "EmqxPlugin",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-management", instance.GetName()),
				Namespace: instance.GetNamespace(),
				Labels:    instance.GetLabels(),
			},
			Spec: appsv1beta3.EmqxPluginSpec{
				PluginName: "emqx_management",
				Selector:   instance.GetLabels(),
				Config: map[string]string{
					"management.listener.http":              "8081",
					"management.default_application.id":     "admin",
					"management.default_application.secret": "public",
				},
			},
		}
		pluginList = append(pluginList, emqxManagement)
	}

	if !isExistPlugin("emqx_dashboard", matchedPluginList) {
		emqxDashboard := &appsv1beta3.EmqxPlugin{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apps.emqx.io/v1beta3",
				Kind:       "EmqxPlugin",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-dashboard", instance.GetName()),
				Namespace: instance.GetNamespace(),
				Labels:    instance.GetLabels(),
			},
			Spec: appsv1beta3.EmqxPluginSpec{
				PluginName: "emqx_dashboard",
				Selector:   instance.GetLabels(),
				Config: map[string]string{
					"dashboard.listener.http":         "18083",
					"dashboard.default_user.login":    "admin",
					"dashboard.default_user.password": "public",
				},
			},
		}
		pluginList = append(pluginList, emqxDashboard)
	}

	if !isExistPlugin("emqx_rule_engine", matchedPluginList) {
		emqxRuleEngine := &appsv1beta3.EmqxPlugin{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apps.emqx.io/v1beta3",
				Kind:       "EmqxPlugin",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-rule-engine", instance.GetName()),
				Namespace: instance.GetNamespace(),
				Labels:    instance.GetLabels(),
			},
			Spec: appsv1beta3.EmqxPluginSpec{
				PluginName: "emqx_rule_engine",
				Selector:   instance.GetLabels(),
				Config:     map[string]string{},
			},
		}
		pluginList = append(pluginList, emqxRuleEngine)
	}

	if !isExistPlugin("emqx_retainer", matchedPluginList) {
		emqxRetainer := &appsv1beta3.EmqxPlugin{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apps.emqx.io/v1beta3",
				Kind:       "EmqxPlugin",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-retainer", instance.GetName()),
				Namespace: instance.GetNamespace(),
				Labels:    instance.GetLabels(),
			},
			Spec: appsv1beta3.EmqxPluginSpec{
				PluginName: "emqx_retainer",
				Selector:   instance.GetLabels(),
				Config:     map[string]string{},
			},
		}
		pluginList = append(pluginList, emqxRetainer)
	}

	_, ok := instance.(*appsv1beta3.EmqxEnterprise)
	if ok && !isExistPlugin("emqx_modules", matchedPluginList) {
		emqxModules := &appsv1beta3.EmqxPlugin{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apps.emqx.io/v1beta3",
				Kind:       "EmqxPlugin",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-modules", instance.GetName()),
				Namespace: instance.GetNamespace(),
				Labels:    instance.GetLabels(),
			},
			Spec: appsv1beta3.EmqxPluginSpec{
				PluginName: "emqx_modules",
				Selector:   instance.GetLabels(),
				Config: map[string]string{
					"modules.loaded_file": "/mounted/modules/loaded_modules",
				},
			},
		}
		pluginList = append(pluginList, emqxModules)
	}

	return pluginList
}

func generateDefaultPluginsConfig(instance appsv1beta3.Emqx, sts *appsv1.StatefulSet) (*corev1.ConfigMap, *appsv1.StatefulSet) {
	names := appsv1beta3.Names{Object: instance}

	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels:    instance.GetLabels(),
			Namespace: instance.GetNamespace(),
			Name:      names.PluginsConfig(),
		},
		Data: map[string]string{
			"emqx_modules.conf":           "",
			"emqx_management.conf":        "management.listener.http = 8081\nmanagement.default_application.id = admin\nmanagement.default_application.secret = public\n",
			"emqx_dashboard.conf":         "dashboard.listener.http = 18083\ndashboard.default_user.login = admin\ndashboard.default_user.password = public\n",
			"emqx_rule_engine.conf":       "",
			"emqx_retainer.conf":          "",
			"emqx_auth_http.conf":         "auth.http.auth_req.url = http://127.0.0.1:80/mqtt/auth\nauth.http.auth_req.method = post\nauth.http.auth_req.headers.content_type = application/x-www-form-urlencoded\nauth.http.auth_req.params = clientid=%c,username=%u,password=%P\nauth.http.acl_req.url = http://127.0.0.1:80/mqtt/acl\nauth.http.acl_req.method = post\nauth.http.acl_req.headers.content-type = application/x-www-form-urlencoded\nauth.http.acl_req.params = access=%A,username=%u,clientid=%c,ipaddr=%a,topic=%t,mountpoint=%m\nauth.http.timeout = 5s\nauth.http.connect_timeout = 5s\nauth.http.pool_size = 32\nauth.http.enable_pipelining = true\n",
			"emqx_auth_jwt.conf":          "auth.jwt.secret = emqxsecret\nauth.jwt.from = password\nauth.jwt.verify_claims = off\n",
			"emqx_auth_ldap.conf":         "auth.ldap.servers = 127.0.0.1\nauth.ldap.port = 389\nauth.ldap.pool = 8\nauth.ldap.bind_dn = cn=root,dc=emqx,dc=io\nauth.ldap.bind_password = public\nauth.ldap.timeout = 30s\nauth.ldap.device_dn = ou=device,dc=emqx,dc=io\nauth.ldap.match_objectclass = mqttUser\nauth.ldap.username.attributetype = uid\nauth.ldap.password.attributetype = userPassword\nauth.ldap.ssl = false\n",
			"emqx_auth_mnesia.conf":       "",
			"emqx_auth_mongo.conf":        "auth.mongo.type = single\nauth.mongo.srv_record = false\nauth.mongo.server = 127.0.0.1:27017\nauth.mongo.pool = 8\nauth.mongo.database = mqtt\nauth.mongo.topology.pool_size = 1\nauth.mongo.topology.max_overflow = 0\nauth.mongo.auth_query.password_hash = sha256\nauth.mongo.auth_query.collection = mqtt_user\nauth.mongo.auth_query.password_field = password\nauth.mongo.auth_query.selector = username=%u\nauth.mongo.super_query.collection = mqtt_user\nauth.mongo.super_query.super_field = is_superuser\nauth.mongo.super_query.selector = username=%u\nauth.mongo.acl_query.collection = mqtt_acl\nauth.mongo.acl_query.selector = username=%u\n",
			"emqx_auth_mysql.conf":        "auth.mysql.server = 127.0.0.1:3306\nauth.mysql.pool = 8\nauth.mysql.database = mqtt\nauth.mysql.auth_query = select password from mqtt_user where username = '%u' limit 1\nauth.mysql.password_hash = sha256\nauth.mysql.super_query = select is_superuser from mqtt_user where username = '%u' limit 1\nauth.mysql.acl_query = select allow, ipaddr, username, clientid, access, topic from mqtt_acl where ipaddr = '%a' or username = '%u' or username = '$all' or clientid = '%c'\n",
			"emqx_auth_pgsql.conf":        "auth.pgsql.server = 127.0.0.1:5432\nauth.pgsql.pool = 8\nauth.pgsql.username = root\nauth.pgsql.database = mqtt\nauth.pgsql.encoding = utf8\nauth.pgsql.ssl = off\nauth.pgsql.auth_query = select password from mqtt_user where username = '%u' limit 1\nauth.pgsql.password_hash = sha256\nauth.pgsql.super_query = select is_superuser from mqtt_user where username = '%u' limit 1\nauth.pgsql.acl_query = select allow, ipaddr, username, clientid, access, topic from mqtt_acl where ipaddr = '%a' or username = '%u' or username = '$all' or clientid = '%c'\n",
			"emqx_auth_redis.conf":        "auth.redis.type = single\nauth.redis.server = 127.0.0.1:6379\nauth.redis.pool = 8\nauth.redis.database = 0\nauth.redis.auth_cmd = HMGET mqtt_user:%u password\nauth.redis.password_hash = plain\nauth.redis.super_cmd = HGET mqtt_user:%u is_superuser\nauth.redis.acl_cmd = HGETALL mqtt_acl:%u\n",
			"emqx_backend_cassa.conf":     "backend.ecql.pool1.nodes = 127.0.0.1:9042\nbackend.ecql.pool1.size = 8\nbackend.ecql.pool1.auto_reconnect = 1\nbackend.ecql.pool1.username = cassandra\nbackend.ecql.pool1.password = cassandra\nbackend.ecql.pool1.keyspace = mqtt\nbackend.ecql.pool1.logger = info\nbackend.cassa.hook.client.connected.1    = {\"action\": {\"function\": \"on_client_connected\"}, \"pool\": \"pool1\"}\nbackend.cassa.hook.client.connected.2    = {\"action\": {\"function\": \"on_subscription_lookup\"}, \"pool\": \"pool1\"}\nbackend.cassa.hook.client.disconnected.1 = {\"action\": {\"function\": \"on_client_disconnected\"}, \"pool\": \"pool1\"}\nbackend.cassa.hook.session.subscribed.1  = {\"topic\": \"#\", \"action\": {\"function\": \"on_message_fetch\"}, \"offline_opts\": {\"max_returned_count\": 500, \"time_range\": \"2h\"}, \"pool\": \"pool1\"}\nbackend.cassa.hook.session.subscribed.2  = {\"action\": {\"function\": \"on_retain_lookup\"}, \"pool\": \"pool1\"}\nbackend.cassa.hook.session.unsubscribed.1= {\"topic\": \"#\", \"action\": {\"cql\": [\"delete from acked where clientid = ${clientid} and topic = ${topic}\"]}, \"pool\": \"pool1\"}\nbackend.cassa.hook.message.publish.1     = {\"topic\": \"#\", \"action\": {\"function\": \"on_message_publish\"}, \"pool\": \"pool1\"}\nbackend.cassa.hook.message.publish.2     = {\"topic\": \"#\", \"action\": {\"function\": \"on_message_retain\"}, \"pool\": \"pool1\"}\nbackend.cassa.hook.message.publish.3     = {\"topic\": \"#\", \"action\": {\"function\": \"on_retain_delete\"}, \"pool\": \"pool1\"}\nbackend.cassa.hook.message.acked.1       = {\"topic\": \"#\", \"action\": {\"function\": \"on_message_acked\"}, \"pool\": \"pool1\"}\n",
			"emqx_backend_dynamo.conf":    "backend.dynamo.region = us-west-2\nbackend.dynamo.pool1.server = http://localhost:8000\nbackend.dynamo.pool1.pool_size = 8\nbackend.dynamo.pool1.aws_access_key_id = FAKE_AWS_ACCESS_KEY_ID\nbackend.dynamo.pool1.aws_secret_access_key = FAKE_AWS_SECRET_ACCESS_KEY\nbackend.dynamo.hook.client.connected.1    = {\"action\": {\"function\": \"on_client_connected\"}, \"pool\": \"pool1\"}\nbackend.dynamo.hook.client.connected.2    = {\"action\": {\"function\": \"on_subscribe_lookup\"}, \"pool\": \"pool1\"}\nbackend.dynamo.hook.client.disconnected.1 = {\"action\": {\"function\": \"on_client_disconnected\"}, \"pool\": \"pool1\"}\nbackend.dynamo.hook.session.subscribed.1  = {\"topic\": \"#\", \"action\": {\"function\": \"on_message_fetch_for_queue\"}, \"pool\": \"pool1\"}\nbackend.dynamo.hook.session.subscribed.2  = {\"topic\": \"#\", \"action\": {\"function\": \"on_retain_lookup\"}, \"pool\": \"pool1\"}\nbackend.dynamo.hook.session.unsubscribed.1= {\"topic\": \"#\", \"action\": {\"function\": \"on_acked_delete\"}, \"pool\": \"pool1\"}\nbackend.dynamo.hook.message.publish.1     = {\"topic\": \"#\", \"action\": {\"function\": \"on_message_publish\"}, \"pool\": \"pool1\"}\nbackend.dynamo.hook.message.publish.2     = {\"topic\": \"#\", \"action\": {\"function\": \"on_message_retain\"}, \"pool\": \"pool1\"}\nbackend.dynamo.hook.message.publish.3     = {\"topic\": \"#\", \"action\": {\"function\": \"on_retain_delete\"}, \"pool\": \"pool1\"}\nbackend.dynamo.hook.message.acked.1       = {\"topic\": \"#\", \"action\": {\"function\": \"on_message_acked_for_queue\"}, \"pool\": \"pool1\"}\n",
			"emqx_backend_influxdb.conf":  "backend.influxdb.udp.pool1.server = 127.0.0.1:8089\nbackend.influxdb.udp.pool1.pool_size = 8\nbackend.influxdb.http.pool1.server = 127.0.0.1:8086\nbackend.influxdb.http.pool1.pool_size = 8\nbackend.influxdb.http.pool1.precision = ms\nbackend.influxdb.http.pool1.database = mydb\nbackend.influxdb.http.pool1.https_enabled = false\nbackend.influxdb.hook.message.publish.1 = {\"topic\": \"#\", \"action\": {\"function\": \"on_message_publish\"}, \"pool\": \"pool1\"}\n",
			"emqx_backend_mongo.conf":     "backend.mongo.pool1.type = single\nbackend.mongo.pool1.srv_record = false\nbackend.mongo.pool1.server = 127.0.0.1:27017\nbackend.mongo.pool1.c_pool_size = 8\nbackend.mongo.pool1.database = mqtt\nbackend.mongo.hook.client.connected.1    = {\"action\": {\"function\": \"on_client_connected\"}, \"pool\": \"pool1\"}\nbackend.mongo.hook.client.connected.2    = {\"action\": {\"function\": \"on_subscribe_lookup\"}, \"pool\": \"pool1\"}\nbackend.mongo.hook.client.disconnected.1 = {\"action\": {\"function\": \"on_client_disconnected\"}, \"pool\": \"pool1\"}\nbackend.mongo.hook.session.subscribed.1  = {\"topic\": \"#\", \"action\": {\"function\": \"on_message_fetch\"}, \"pool\": \"pool1\", \"offline_opts\": {\"time_range\": \"2h\", \"max_returned_count\": 500}}\nbackend.mongo.hook.session.subscribed.2  = {\"topic\": \"#\", \"action\": {\"function\": \"on_retain_lookup\"}, \"pool\": \"pool1\"}\nbackend.mongo.hook.session.unsubscribed.1= {\"topic\": \"#\", \"action\": {\"function\": \"on_acked_delete\"}, \"pool\": \"pool1\"}\nbackend.mongo.hook.message.publish.1     = {\"topic\": \"#\", \"action\": {\"function\": \"on_message_publish\"}, \"pool\": \"pool1\"}\nbackend.mongo.hook.message.publish.2     = {\"topic\": \"#\", \"action\": {\"function\": \"on_message_retain\"}, \"pool\": \"pool1\"}\nbackend.mongo.hook.message.publish.3     = {\"topic\": \"#\", \"action\": {\"function\": \"on_retain_delete\"}, \"pool\": \"pool1\"}\nbackend.mongo.hook.message.acked.1       = {\"topic\": \"#\", \"action\": {\"function\": \"on_message_acked\"}, \"pool\": \"pool1\"}\n",
			"emqx_backend_mysql.conf":     "backend.mysql.pool1.server = 127.0.0.1:3306\nbackend.mysql.pool1.pool_size = 8\nbackend.mysql.pool1.user = root\nbackend.mysql.pool1.password = public\nbackend.mysql.pool1.database = mqtt\nbackend.mysql.hook.client.connected.1    = {\"action\": {\"function\": \"on_client_connected\"}, \"pool\": \"pool1\"}\nbackend.mysql.hook.client.connected.2     = {\"action\": {\"function\": \"on_subscribe_lookup\"}, \"pool\": \"pool1\"}\nbackend.mysql.hook.client.disconnected.1 = {\"action\": {\"function\": \"on_client_disconnected\"}, \"pool\": \"pool1\"}\nbackend.mysql.hook.session.subscribed.1  = {\"topic\": \"#\", \"action\": {\"function\": \"on_message_fetch\"}, \"offline_opts\": {\"max_returned_count\": 500, \"time_range\": \"2h\"}, \"pool\": \"pool1\"}\nbackend.mysql.hook.session.subscribed.2  = {\"topic\": \"#\", \"action\": {\"function\": \"on_retain_lookup\"}, \"pool\": \"pool1\"}\nbackend.mysql.hook.session.unsubscribed.1= {\"topic\": \"#\", \"action\": {\"sql\": [\"delete from mqtt_acked where clientid = ${clientid} and topic = ${topic}\"]}, \"pool\": \"pool1\"}\nbackend.mysql.hook.message.publish.1     = {\"topic\": \"#\", \"action\": {\"function\": \"on_message_publish\"}, \"pool\": \"pool1\"}\nbackend.mysql.hook.message.publish.2     = {\"topic\": \"#\", \"action\": {\"function\": \"on_message_retain\"}, \"pool\": \"pool1\"}\nbackend.mysql.hook.message.publish.3     = {\"topic\": \"#\", \"action\": {\"function\": \"on_retain_delete\"}, \"pool\": \"pool1\"}\nbackend.mysql.hook.message.acked.1       = {\"topic\": \"#\", \"action\": {\"function\": \"on_message_acked\"}, \"pool\": \"pool1\"}\n",
			"emqx_backend_opentsdb.conf":  "backend.opentsdb.pool1.server = 127.0.0.1:4242\nbackend.opentsdb.pool1.pool_size = 8\nbackend.opentsdb.pool1.summary = true\nbackend.opentsdb.pool1.details = false\nbackend.opentsdb.pool1.sync = false\nbackend.opentsdb.pool1.sync_timeout = 0\nbackend.opentsdb.pool1.max_batch_size = 20\nbackend.opentsdb.hook.message.publish.1 = {\"topic\": \"#\", \"action\": {\"function\": \"on_message_publish\"}, \"pool\": \"pool1\"}\n",
			"emqx_backend_pgsql.conf":     "backend.pgsql.pool1.server = 127.0.0.1:5432\nbackend.pgsql.pool1.pool_size = 8\nbackend.pgsql.pool1.username = root\nbackend.pgsql.pool1.password = public\nbackend.pgsql.pool1.database = mqtt\nbackend.pgsql.pool1.ssl = false\nbackend.pgsql.hook.client.connected.1    = {\"action\": {\"function\": \"on_client_connected\"}, \"pool\": \"pool1\"}\nbackend.pgsql.hook.client.connected.2     = {\"action\": {\"function\": \"on_subscribe_lookup\"}, \"pool\": \"pool1\"}\nbackend.pgsql.hook.client.disconnected.1 = {\"action\": {\"function\": \"on_client_disconnected\"}, \"pool\": \"pool1\"}\nbackend.pgsql.hook.session.subscribed.1  = {\"topic\": \"#\", \"action\": {\"function\": \"on_message_fetch\"}, \"offline_opts\": {\"max_returned_count\": 500, \"time_range\": \"2h\"}, \"pool\": \"pool1\"}\nbackend.pgsql.hook.session.subscribed.2  = {\"topic\": \"#\", \"action\": {\"function\": \"on_retain_lookup\"}, \"pool\": \"pool1\"}\nbackend.pgsql.hook.session.unsubscribed.1= {\"topic\": \"#\", \"action\": {\"sql\": \"delete from mqtt_acked where clientid = ${clientid} and topic = ${topic}\"}, \"pool\": \"pool1\"}\nbackend.pgsql.hook.message.publish.1     = {\"topic\": \"#\", \"action\": {\"function\": \"on_message_publish\"}, \"pool\": \"pool1\"}\nbackend.pgsql.hook.message.publish.2     = {\"topic\": \"#\", \"action\": {\"function\": \"on_message_retain\"}, \"pool\": \"pool1\"}\nbackend.pgsql.hook.message.publish.3     = {\"topic\": \"#\", \"action\": {\"function\": \"on_retain_delete\"}, \"pool\": \"pool1\"}\nbackend.pgsql.hook.message.acked.1       = {\"topic\": \"#\", \"action\": {\"function\": \"on_message_acked\"}, \"pool\": \"pool1\"}\n",
			"emqx_backend_redis.conf":     "backend.redis.pool1.type = single\nbackend.redis.pool1.server = 127.0.0.1:6379\nbackend.redis.pool1.pool_size = 8\nbackend.redis.pool1.database = 0\nbackend.redis.pool1.channel = mqtt_channel\nbackend.redis.hook.client.connected.1    = {\"action\": {\"function\": \"on_client_connected\"}, \"pool\": \"pool1\"}\nbackend.redis.hook.client.connected.2    = {\"action\": {\"function\": \"on_subscribe_lookup\"}, \"pool\": \"pool1\"}\nbackend.redis.hook.client.disconnected.1 = {\"action\": {\"function\": \"on_client_disconnected\"}, \"pool\": \"pool1\"}\nbackend.redis.hook.session.subscribed.1  = {\"topic\": \"queue/#\", \"action\": {\"function\": \"on_message_fetch_for_queue\"}, \"pool\": \"pool1\"}\nbackend.redis.hook.session.subscribed.2  = {\"topic\": \"pubsub/#\", \"action\": {\"function\": \"on_message_fetch_for_pubsub\"}, \"pool\": \"pool1\"}\nbackend.redis.hook.session.subscribed.3  = {\"action\": {\"function\": \"on_retain_lookup\"}, \"pool\": \"pool1\"}\nbackend.redis.hook.session.unsubscribed.1= {\"topic\": \"#\", \"action\": {\"commands\": [\"DEL mqtt:acked:${clientid}:${topic}\"]}, \"pool\": \"pool1\"}\nbackend.redis.hook.message.publish.1     = {\"topic\": \"#\", \"action\": {\"function\": \"on_message_publish\"}, \"expired_time\" : 3600, \"pool\": \"pool1\"}\nbackend.redis.hook.message.publish.2     = {\"topic\": \"#\", \"action\": {\"function\": \"on_message_retain\"}, \"expired_time\" : 3600, \"pool\": \"pool1\"}\nbackend.redis.hook.message.publish.3     = {\"topic\": \"#\", \"action\": {\"function\": \"on_retain_delete\"}, \"pool\": \"pool1\"}\nbackend.redis.hook.message.acked.1       = {\"topic\": \"queue/#\", \"action\": {\"function\": \"on_message_acked_for_queue\"}, \"pool\": \"pool1\"}\nbackend.redis.hook.message.acked.2       = {\"topic\": \"pubsub/#\", \"action\": {\"function\": \"on_message_acked_for_pubsub\"}, \"pool\": \"pool1\"}\n",
			"emqx_backend_timescale.conf": "backend.timescale.pool1.server = 127.0.0.1:5432\nbackend.timescale.pool1.pool_size = 8\nbackend.timescale.pool1.username = root\nbackend.timescale.pool1.password = public\nbackend.timescale.pool1.database = mqtt\nbackend.timescale.pool1.ssl = false\nbackend.timescale.hook.message.publish.1 = {\"topic\": \"#\", \"action\": {\"function\": \"on_message_publish\"}, \"pool\": \"pool1\"}\n",
			"emqx_bridge_kafka.conf":      "bridge.kafka.servers = 127.0.0.1:9092\nbridge.kafka.query_api_versions = true\nbridge.kafka.connection_strategy = per_partition\nbridge.kafka.min_metadata_refresh_interval = 5S\nbridge.kafka.produce = sync\nbridge.kafka.produce.sync_timeout = 3s\nbridge.kafka.sock.sndbuf = 1MB\nbridge.kafka.hook.client.connected.1     = {\"topic\":\"ClientConnected\"}\nbridge.kafka.hook.client.disconnected.1  = {\"topic\":\"ClientDisconnected\"}\nbridge.kafka.hook.session.subscribed.1   = {\"filter\":\"#\", \"topic\":\"SessionSubscribed\"}\nbridge.kafka.hook.session.unsubscribed.1 = {\"filter\":\"#\", \"topic\":\"SessionUnsubscribed\"}\nbridge.kafka.hook.message.publish.1      = {\"filter\":\"#\", \"topic\":\"MessagePublish\"}\nbridge.kafka.hook.message.delivered.1    = {\"filter\":\"#\", \"topic\":\"MessageDelivered\"}\nbridge.kafka.hook.message.acked.1        = {\"filter\":\"#\", \"topic\":\"MessageAcked\"}\n",
			"emqx_bridge_mqtt.conf":       "bridge.mqtt.aws.address = 127.0.0.1:1883\nbridge.mqtt.aws.proto_ver = mqttv4\nbridge.mqtt.aws.start_type = manual\nbridge.mqtt.aws.clientid = bridge_aws\nbridge.mqtt.aws.clean_start = true\nbridge.mqtt.aws.username = user\nbridge.mqtt.aws.password = passwd\nbridge.mqtt.aws.forwards = topic1/#,topic2/#\nbridge.mqtt.aws.forward_mountpoint = bridge/aws/${node}/\nbridge.mqtt.aws.ssl = off\nbridge.mqtt.aws.cacertfile = etc/certs/cacert.pem\nbridge.mqtt.aws.certfile = etc/certs/client-cert.pem\nbridge.mqtt.aws.keyfile = etc/certs/client-key.pem\nbridge.mqtt.aws.ciphers = TLS_AES_256_GCM_SHA384,TLS_AES_128_GCM_SHA256,TLS_CHACHA20_POLY1305_SHA256,TLS_AES_128_CCM_SHA256,TLS_AES_128_CCM_8_SHA256,ECDHE-ECDSA-AES256-GCM-SHA384,ECDHE-RSA-AES256-GCM-SHA384,ECDHE-ECDSA-AES256-SHA384,ECDHE-RSA-AES256-SHA384,ECDHE-ECDSA-DES-CBC3-SHA,ECDH-ECDSA-AES256-GCM-SHA384,ECDH-RSA-AES256-GCM-SHA384,ECDH-ECDSA-AES256-SHA384,ECDH-RSA-AES256-SHA384,DHE-DSS-AES256-GCM-SHA384,DHE-DSS-AES256-SHA256,AES256-GCM-SHA384,AES256-SHA256,ECDHE-ECDSA-AES128-GCM-SHA256,ECDHE-RSA-AES128-GCM-SHA256,ECDHE-ECDSA-AES128-SHA256,ECDHE-RSA-AES128-SHA256,ECDH-ECDSA-AES128-GCM-SHA256,ECDH-RSA-AES128-GCM-SHA256,ECDH-ECDSA-AES128-SHA256,ECDH-RSA-AES128-SHA256,DHE-DSS-AES128-GCM-SHA256,DHE-DSS-AES128-SHA256,AES128-GCM-SHA256,AES128-SHA256,ECDHE-ECDSA-AES256-SHA,ECDHE-RSA-AES256-SHA,DHE-DSS-AES256-SHA,ECDH-ECDSA-AES256-SHA,ECDH-RSA-AES256-SHA,AES256-SHA,ECDHE-ECDSA-AES128-SHA,ECDHE-RSA-AES128-SHA,DHE-DSS-AES128-SHA,ECDH-ECDSA-AES128-SHA,ECDH-RSA-AES128-SHA,AES128-SHA\nbridge.mqtt.aws.keepalive = 60s\nbridge.mqtt.aws.tls_versions = tlsv1.3,tlsv1.2,tlsv1.1,tlsv1\nbridge.mqtt.aws.reconnect_interval = 30s\nbridge.mqtt.aws.retry_interval = 20s\nbridge.mqtt.aws.batch_size = 32\nbridge.mqtt.aws.max_inflight_size = 32\nbridge.mqtt.aws.queue.replayq_dir = data/replayq/emqx_aws_bridge/\nbridge.mqtt.aws.queue.replayq_seg_bytes = 10MB\nbridge.mqtt.aws.queue.max_total_size = 5GB\n",
			"emqx_bridge_pulsar.conf":     "bridge.pulsar.servers = 127.0.0.1:6650\nbridge.pulsar.produce = sync\nbridge.pulsar.sock.sndbuf = 1MB\nbridge.pulsar.hook.client.connected.1     = {\"topic\":\"ClientConnected\"}\nbridge.pulsar.hook.client.disconnected.1  = {\"topic\":\"ClientDisconnected\"}\nbridge.pulsar.hook.session.subscribed.1   = {\"filter\":\"#\", \"topic\":\"SessionSubscribed\"}\nbridge.pulsar.hook.session.unsubscribed.1 = {\"filter\":\"#\", \"topic\":\"SessionUnsubscribed\"}\nbridge.pulsar.hook.message.publish.1      = {\"filter\":\"#\", \"topic\":\"MessagePublish\"}\nbridge.pulsar.hook.message.delivered.1      = {\"filter\":\"#\", \"topic\":\"MessageDelivered\"}\nbridge.pulsar.hook.message.acked.1        = {\"filter\":\"#\", \"topic\":\"MessageAcked\"}\n",
			"emqx_bridge_rabbit.conf":     "bridge.rabbit.1.server = 127.0.0.1:5672\nbridge.rabbit.1.pool_size = 8\nbridge.rabbit.1.username = guest\nbridge.rabbit.1.password = guest\nbridge.rabbit.1.timeout = 5s\nbridge.rabbit.1.virtual_host = /\nbridge.rabbit.1.heartbeat = 30s\nbridge.rabbit.hook.session.subscribed.1 = {\"action\": \"on_session_subscribed\", \"rabbit\": 1, \"exchange\": \"direct:emqx.subscription\"}\nbridge.rabbit.hook.session.unsubscribed.1 = {\"action\": \"on_session_unsubscribed\", \"rabbit\": 1, \"exchange\": \"direct:emqx.subscription\"}\nbridge.rabbit.hook.message.publish.1 = {\"topic\": \"$SYS/#\", \"action\": \"on_message_publish\", \"rabbit\": 1, \"exchange\": \"topic:emqx.$sys\"}\nbridge.rabbit.hook.message.publish.2 = {\"topic\": \"#\", \"action\": \"on_message_publish\", \"rabbit\": 1, \"exchange\": \"topic:emqx.pub\"}\nbridge.rabbit.hook.message.acked.1 = {\"topic\": \"#\", \"action\": \"on_message_acked\", \"rabbit\": 1, \"exchange\": \"topic:emqx.acked\"}\n",
			"emqx_bridge_rocket.conf":     "bridge.rocket.servers = 127.0.0.1:9876\nbridge.rocket.refresh_topic_route_interval = 5S\nbridge.rocket.produce = sync\nbridge.rocket.sock.sndbuf = 1MB\nbridge.rocket.hook.client.connected.1     = {\"topic\":\"ClientConnected\"}\nbridge.rocket.hook.client.disconnected.1  = {\"topic\":\"ClientDisconnected\"}\nbridge.rocket.hook.session.subscribed.1   = {\"filter\":\"#\", \"topic\":\"SessionSubscribed\"}\nbridge.rocket.hook.session.unsubscribed.1 = {\"filter\":\"#\", \"topic\":\"SessionUnsubscribed\"}\nbridge.rocket.hook.message.publish.1      = {\"filter\":\"#\", \"topic\":\"MessagePublish\"}\nbridge.rocket.hook.message.delivered.1    = {\"filter\":\"#\", \"topic\":\"MessageDeliver\"}\nbridge.rocket.hook.message.acked.1        = {\"filter\":\"#\", \"topic\":\"MessageAcked\"}\n",
			"emqx_coap.conf":              "coap.bind.udp.1 = 0.0.0.0:5683\ncoap.enable_stats = off\ncoap.bind.dtls.1 = 0.0.0.0:5684\ncoap.dtls.keyfile = etc/certs/key.pem\ncoap.dtls.certfile = etc/certs/cert.pem\ncoap.dtls.ciphers = ECDHE-ECDSA-AES256-GCM-SHA384,ECDHE-RSA-AES256-GCM-SHA384,ECDHE-ECDSA-AES256-SHA384,ECDHE-RSA-AES256-SHA384,ECDHE-ECDSA-DES-CBC3-SHA,ECDH-ECDSA-AES256-GCM-SHA384,ECDH-RSA-AES256-GCM-SHA384,ECDH-ECDSA-AES256-SHA384,ECDH-RSA-AES256-SHA384,DHE-DSS-AES256-GCM-SHA384,DHE-DSS-AES256-SHA256,AES256-GCM-SHA384,AES256-SHA256,ECDHE-ECDSA-AES128-GCM-SHA256,ECDHE-RSA-AES128-GCM-SHA256,ECDHE-ECDSA-AES128-SHA256,ECDHE-RSA-AES128-SHA256,ECDH-ECDSA-AES128-GCM-SHA256,ECDH-RSA-AES128-GCM-SHA256,ECDH-ECDSA-AES128-SHA256,ECDH-RSA-AES128-SHA256,DHE-DSS-AES128-GCM-SHA256,DHE-DSS-AES128-SHA256,AES128-GCM-SHA256,AES128-SHA256,ECDHE-ECDSA-AES256-SHA,ECDHE-RSA-AES256-SHA,DHE-DSS-AES256-SHA,ECDH-ECDSA-AES256-SHA,ECDH-RSA-AES256-SHA,AES256-SHA,ECDHE-ECDSA-AES128-SHA,ECDHE-RSA-AES128-SHA,DHE-DSS-AES128-SHA,ECDH-ECDSA-AES128-SHA,ECDH-RSA-AES128-SHA,AES128-SHA\n",
			"emqx_conf.conf":              "conf.etc.dir.emqx = etc\nconf.etc.dir.emqx.zones = etc\nconf.etc.dir.emqx.listeners = etc\nconf.etc.dir.emqx.sys_mon = etc\n",
			"emqx_exhook.conf":            "exhook.server.default.url = http://127.0.0.1:9000\n",
			"emqx_exproto.conf":           "exproto.server.http.port = 9100\nexproto.server.https.port = 9101\nexproto.server.https.cacertfile = etc/certs/cacert.pem\nexproto.server.https.certfile = etc/certs/cert.pem\nexproto.server.https.keyfile = etc/certs/key.pem\nexproto.listener.protoname = tcp://0.0.0.0:7993\nexproto.listener.protoname.connection_handler_url = http://127.0.0.1:9001\nexproto.listener.protoname.acceptors = 8\nexproto.listener.protoname.max_connections = 1024000\nexproto.listener.protoname.max_conn_rate = 1000\nexproto.listener.protoname.active_n = 100\nexproto.listener.protoname.idle_timeout = 30s\nexproto.listener.protoname.access.1 = allow all\nexproto.listener.protoname.backlog = 1024\nexproto.listener.protoname.send_timeout = 15s\nexproto.listener.protoname.send_timeout_close = on\nexproto.listener.protoname.nodelay = true\nexproto.listener.protoname.reuseaddr = true\n",
			"emqx_gbt32960.conf":          "gbt32960.frame.max_length = 8192\ngbt32960.proto.retx_interval = 8s\ngbt32960.proto.retx_max_times = 3\ngbt32960.proto.message_queue_len = 10\ngbt32960.listener.tcp = 0.0.0.0:7325\ngbt32960.listener.tcp.acceptors = 8\ngbt32960.listener.tcp.max_connections = 1024000\ngbt32960.listener.tcp.max_conn_rate = 1000\ngbt32960.listener.tcp.idle_timeout = 60s\ngbt32960.listener.tcp.active_n = 100\ngbt32960.listener.tcp.zone = external\ngbt32960.listener.tcp.access.1 = allow all\ngbt32960.listener.tcp.backlog = 1024\ngbt32960.listener.tcp.send_timeout = 15s\ngbt32960.listener.tcp.send_timeout_close = on\ngbt32960.listener.tcp.nodelay = true\ngbt32960.listener.tcp.reuseaddr = true\ngbt32960.listener.ssl = 7326\ngbt32960.listener.ssl.acceptors = 16\ngbt32960.listener.ssl.max_connections = 102400\ngbt32960.listener.ssl.max_conn_rate = 500\ngbt32960.listener.ssl.idle_timeout = 60s\ngbt32960.listener.ssl.active_n = 100\ngbt32960.listener.ssl.zone = external\ngbt32960.listener.ssl.access.1 = allow all\ngbt32960.listener.ssl.handshake_timeout = 15s\ngbt32960.listener.ssl.keyfile = etc/certs/key.pem\ngbt32960.listener.ssl.certfile = etc/certs/cert.pem\ngbt32960.listener.ssl.ciphers = ECDHE-ECDSA-AES256-GCM-SHA384,ECDHE-RSA-AES256-GCM-SHA384,ECDHE-ECDSA-AES256-SHA384,ECDHE-RSA-AES256-SHA384,ECDHE-ECDSA-DES-CBC3-SHA,ECDH-ECDSA-AES256-GCM-SHA384,ECDH-RSA-AES256-GCM-SHA384,ECDH-ECDSA-AES256-SHA384,ECDH-RSA-AES256-SHA384,DHE-DSS-AES256-GCM-SHA384,DHE-DSS-AES256-SHA256,AES256-GCM-SHA384,AES256-SHA256,ECDHE-ECDSA-AES128-GCM-SHA256,ECDHE-RSA-AES128-GCM-SHA256,ECDHE-ECDSA-AES128-SHA256,ECDHE-RSA-AES128-SHA256,ECDH-ECDSA-AES128-GCM-SHA256,ECDH-RSA-AES128-GCM-SHA256,ECDH-ECDSA-AES128-SHA256,ECDH-RSA-AES128-SHA256,DHE-DSS-AES128-GCM-SHA256,DHE-DSS-AES128-SHA256,AES128-GCM-SHA256,AES128-SHA256,ECDHE-ECDSA-AES256-SHA,ECDHE-RSA-AES256-SHA,DHE-DSS-AES256-SHA,ECDH-ECDSA-AES256-SHA,ECDH-RSA-AES256-SHA,AES256-SHA,ECDHE-ECDSA-AES128-SHA,ECDHE-RSA-AES128-SHA,DHE-DSS-AES128-SHA,ECDH-ECDSA-AES128-SHA,ECDH-RSA-AES128-SHA,AES128-SHA\ngbt32960.listener.ssl.reuseaddr = true\n",
			"emqx_jt808.conf":             "jt808.proto.allow_anonymous = true\njt808.proto.dn_topic = jt808/%c/dn\njt808.proto.up_topic = jt808/%c/up\njt808.conn.idle_timeout = 30s\njt808.conn.enable_stats = on\njt808.frame.max_length = 8192\njt808.listener.tcp = 6207\njt808.listener.tcp.acceptors = 4\njt808.listener.tcp.max_clients = 512\n",
			"emqx_lua_hook.conf":          "",
			"emqx_lwm2m.conf":             "lwm2m.lifetime_min = 1s\nlwm2m.lifetime_max = 86400s\nlwm2m.mountpoint = lwm2m/%e/\nlwm2m.topics.command = dn/#\nlwm2m.topics.response = up/resp\nlwm2m.topics.notify = up/notify\nlwm2m.topics.register = up/resp\nlwm2m.topics.update = up/resp\nlwm2m.xml_dir =  etc/lwm2m_xml\nlwm2m.bind.udp.1 = 0.0.0.0:5683\nlwm2m.opts.buffer = 1024KB\nlwm2m.opts.recbuf = 1024KB\nlwm2m.opts.sndbuf = 1024KB\nlwm2m.opts.read_packets = 20\nlwm2m.bind.dtls.1 = 0.0.0.0:5684\nlwm2m.dtls.keyfile = etc/certs/key.pem\nlwm2m.dtls.certfile = etc/certs/cert.pem\nlwm2m.dtls.ciphers = ECDHE-ECDSA-AES256-GCM-SHA384,ECDHE-RSA-AES256-GCM-SHA384,ECDHE-ECDSA-AES256-SHA384,ECDHE-RSA-AES256-SHA384,ECDHE-ECDSA-DES-CBC3-SHA,ECDH-ECDSA-AES256-GCM-SHA384,ECDH-RSA-AES256-GCM-SHA384,ECDH-ECDSA-AES256-SHA384,ECDH-RSA-AES256-SHA384,DHE-DSS-AES256-GCM-SHA384,DHE-DSS-AES256-SHA256,AES256-GCM-SHA384,AES256-SHA256,ECDHE-ECDSA-AES128-GCM-SHA256,ECDHE-RSA-AES128-GCM-SHA256,ECDHE-ECDSA-AES128-SHA256,ECDHE-RSA-AES128-SHA256,ECDH-ECDSA-AES128-GCM-SHA256,ECDH-RSA-AES128-GCM-SHA256,ECDH-ECDSA-AES128-SHA256,ECDH-RSA-AES128-SHA256,DHE-DSS-AES128-GCM-SHA256,DHE-DSS-AES128-SHA256,AES128-GCM-SHA256,AES128-SHA256,ECDHE-ECDSA-AES256-SHA,ECDHE-RSA-AES256-SHA,DHE-DSS-AES256-SHA,ECDH-ECDSA-AES256-SHA,ECDH-RSA-AES256-SHA,AES256-SHA,ECDHE-ECDSA-AES128-SHA,ECDHE-RSA-AES128-SHA,DHE-DSS-AES128-SHA,ECDH-ECDSA-AES128-SHA,ECDH-RSA-AES128-SHA,AES128-SHA\n",
			"emqx_prometheus.conf":        "prometheus.push.gateway.server = http://127.0.0.1:9091\nprometheus.interval = 15000\n",
			"emqx_psk_file.conf":          "psk.file.path = etc/psk.txt\npsk.file.delimiter = :\n",
			"emqx_recon.conf":             "",
			"emqx_sasl.conf":              "",
			"emqx_schema_registry.conf":   "",
			"emqx_sn.conf":                "mqtt.sn.port = 1884\nmqtt.sn.advertise_duration = 15m\nmqtt.sn.gateway_id = 1\nmqtt.sn.enable_stats = off\nmqtt.sn.enable_qos3 = off\nmqtt.sn.idle_timeout = 30s\nmqtt.sn.predefined.topic.0 = reserved\nmqtt.sn.predefined.topic.1 = /predefined/topic/name/hello\nmqtt.sn.predefined.topic.2 = /predefined/topic/name/nice\nmqtt.sn.username = mqtt_sn_user\nmqtt.sn.password = abc\n",
			"emqx_stomp.conf":             "stomp.listener = 61613\nstomp.listener.acceptors = 4\nstomp.listener.max_connections = 512\nstomp.default_user.login = guest\nstomp.default_user.passcode = guest\nstomp.allow_anonymous = true\nstomp.frame.max_headers = 10\nstomp.frame.max_header_length = 1024\nstomp.frame.max_body_length = 8192\n",
			"emqx_tcp.conf":               "tcp.proto.idle_timeout = 15s\ntcp.proto.up_topic = tcp/%c/up\ntcp.proto.dn_topic = tcp/%c/dn\ntcp.proto.max_packet_size = 65535\ntcp.proto.enable_stats = on\ntcp.proto.force_gc_policy = 1000|1MB\ntcp.listener.external = 0.0.0.0:8090\ntcp.listener.external.acceptors = 8\ntcp.listener.external.max_connections = 1024000\ntcp.listener.external.max_conn_rate = 1000\ntcp.listener.external.active_n = 100\ntcp.listener.external.access.1 = allow all\ntcp.listener.external.backlog = 1024\ntcp.listener.external.send_timeout = 15s\ntcp.listener.external.send_timeout_close = on\ntcp.listener.external.nodelay = true\ntcp.listener.external.reuseaddr = true\ntcp.listener.ssl.external = 0.0.0.0:8091\ntcp.listener.ssl.external.acceptors = 8\ntcp.listener.ssl.external.max_connections = 1024000\ntcp.listener.ssl.external.max_conn_rate = 1000\ntcp.listener.ssl.external.active_n = 100\ntcp.listener.ssl.external.access.1 = allow all\ntcp.listener.ssl.external.handshake_timeout = 15s\ntcp.listener.ssl.external.keyfile = etc/certs/key.pem\ntcp.listener.ssl.external.certfile = etc/certs/cert.pem\ntcp.listener.ssl.external.ciphers = ECDHE-ECDSA-AES256-GCM-SHA384,ECDHE-RSA-AES256-GCM-SHA384,ECDHE-ECDSA-AES256-SHA384,ECDHE-RSA-AES256-SHA384,ECDHE-ECDSA-DES-CBC3-SHA,ECDH-ECDSA-AES256-GCM-SHA384,ECDH-RSA-AES256-GCM-SHA384,ECDH-ECDSA-AES256-SHA384,ECDH-RSA-AES256-SHA384,DHE-DSS-AES256-GCM-SHA384,DHE-DSS-AES256-SHA256,AES256-GCM-SHA384,AES256-SHA256,ECDHE-ECDSA-AES128-GCM-SHA256,ECDHE-RSA-AES128-GCM-SHA256,ECDHE-ECDSA-AES128-SHA256,ECDHE-RSA-AES128-SHA256,ECDH-ECDSA-AES128-GCM-SHA256,ECDH-RSA-AES128-GCM-SHA256,ECDH-ECDSA-AES128-SHA256,ECDH-RSA-AES128-SHA256,DHE-DSS-AES128-GCM-SHA256,DHE-DSS-AES128-SHA256,AES128-GCM-SHA256,AES128-SHA256,ECDHE-ECDSA-AES256-SHA,ECDHE-RSA-AES256-SHA,DHE-DSS-AES256-SHA,ECDH-ECDSA-AES256-SHA,ECDH-RSA-AES256-SHA,AES256-SHA,ECDHE-ECDSA-AES128-SHA,ECDHE-RSA-AES128-SHA,DHE-DSS-AES128-SHA,ECDH-ECDSA-AES128-SHA,ECDH-RSA-AES128-SHA,AES128-SHA\ntcp.listener.ssl.external.backlog = 1024\ntcp.listener.ssl.external.send_timeout = 15s\ntcp.listener.ssl.external.send_timeout_close = on\ntcp.listener.ssl.external.nodelay = true\ntcp.listener.ssl.external.reuseaddr = true\n",
			"emqx_web_hook.conf":          "web.hook.url = http://127.0.0.1:80\nweb.hook.headers.content-type = application/json\nweb.hook.body.encoding_of_payload_field = plain\nweb.hook.pool_size = 32\nweb.hook.enable_pipelining = true\n",
		},
	}

	container := sts.Spec.Template.Spec.Containers[0]
	container.Env = append(
		container.Env,
		corev1.EnvVar{
			Name:  "EMQX_PLUGINS__ETC_DIR",
			Value: "/mounted/plugins/etc",
		},
	)
	container.VolumeMounts = append(
		container.VolumeMounts,
		corev1.VolumeMount{
			Name:      cm.Name,
			MountPath: "/mounted/plugins/etc",
		},
	)
	sts.Spec.Template.Spec.Containers[0] = container
	sts.Spec.Template.Spec.Volumes = append(
		sts.Spec.Template.Spec.Volumes,
		corev1.Volume{
			Name: cm.Name,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: cm.Name,
					},
				},
			},
		},
	)

	return cm, sts
}

func generateLoadedPlugins(instance appsv1beta3.Emqx, sts *appsv1.StatefulSet) (*corev1.ConfigMap, *appsv1.StatefulSet) {
	names := appsv1beta3.Names{Object: instance}
	loadedPlugins := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels:    instance.GetLabels(),
			Namespace: instance.GetNamespace(),
			Name:      names.LoadedPlugins(),
		},
		Data: make(map[string]string),
	}

	switch instance.(type) {
	case *appsv1beta3.EmqxBroker:
		loadedPlugins.Data["loaded_plugins"] = "emqx_management.\nemqx_dashboard.\nemqx_retainer.\nemqx_rule_engine.\n"
	case *appsv1beta3.EmqxEnterprise:
		loadedPlugins.Data["loaded_plugins"] = "emqx_management.\nemqx_dashboard.\nemqx_retainer.\nemqx_rule_engine.\nemqx_modules.\n"
	}

	container := sts.Spec.Template.Spec.Containers[0]
	container.VolumeMounts = append(
		container.VolumeMounts,
		corev1.VolumeMount{
			Name:      loadedPlugins.Name,
			MountPath: "/mounted/plugins/data",
		},
	)
	container.Env = append(
		container.Env,
		corev1.EnvVar{
			Name:  "EMQX_PLUGINS__LOADED_FILE",
			Value: "/mounted/plugins/data/loaded_plugins",
		},
	)
	sts.Spec.Template.Spec.Containers[0] = container

	sts.Spec.Template.Spec.Volumes = append(
		sts.Spec.Template.Spec.Volumes,
		corev1.Volume{
			Name: loadedPlugins.Name,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: loadedPlugins.Name,
					},
				},
			},
		},
	)

	return loadedPlugins, sts
}

func generateSvc(instance appsv1beta3.Emqx, sts *appsv1.StatefulSet) (*corev1.Service, *corev1.Service, *appsv1.StatefulSet) {
	names := appsv1beta3.Names{Object: instance}
	serviceTemplate := instance.GetServiceTemplate()

	svc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: serviceTemplate.ObjectMeta,
		Spec:       serviceTemplate.Spec,
	}

	headlessSvcIPFamilyPolicy := corev1.IPFamilyPolicySingleStack
	headlessSvc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels:    instance.GetLabels(),
			Name:      names.HeadlessSvc(),
			Namespace: instance.GetNamespace(),
		},
		Spec: corev1.ServiceSpec{
			Selector:       instance.GetLabels(),
			ClusterIP:      corev1.ClusterIPNone,
			IPFamilyPolicy: &headlessSvcIPFamilyPolicy,
		},
	}

	compile := regexp.MustCompile(".*management.*")
	for _, port := range svc.Spec.Ports {
		if compile.MatchString(port.Name) {
			headlessSvc.Spec.Ports = append(headlessSvc.Spec.Ports, port)
		}
	}
	sts.Spec.ServiceName = headlessSvc.Name

	return headlessSvc, svc, sts
}

func generateAcl(instance appsv1beta3.Emqx, sts *appsv1.StatefulSet) (*corev1.ConfigMap, *appsv1.StatefulSet) {
	names := appsv1beta3.Names{Object: instance}

	var aclString string
	for _, rule := range instance.GetACL() {
		aclString += fmt.Sprintf("%s\n", rule)
	}

	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels:    instance.GetLabels(),
			Namespace: instance.GetNamespace(),
			Name:      names.ACL(),
		},
		Data: map[string]string{"acl.conf": aclString},
	}

	annotations := sts.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations["ACL/Base64EncodeConfig"] = base64.StdEncoding.EncodeToString([]byte(aclString))
	sts.Annotations = annotations

	container := sts.Spec.Template.Spec.Containers[0]
	container.VolumeMounts = append(
		container.VolumeMounts,
		corev1.VolumeMount{
			Name:      cm.Name,
			MountPath: "/mounted/acl",
		},
	)
	container.Env = append(
		container.Env,
		corev1.EnvVar{
			Name:  "EMQX_ACL_FILE",
			Value: "/mounted/acl/acl.conf",
		},
	)
	sts.Spec.Template.Spec.Containers[0] = container

	sts.Spec.Template.Spec.Volumes = append(
		sts.Spec.Template.Spec.Volumes,
		corev1.Volume{
			Name: cm.Name,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: cm.Name,
					},
				},
			},
		},
	)
	return cm, sts
}

func generateLoadedModules(instance appsv1beta3.Emqx, sts *appsv1.StatefulSet) (*corev1.ConfigMap, *appsv1.StatefulSet) {
	names := appsv1beta3.Names{Object: instance}
	var loadedModulesString string
	switch obj := instance.(type) {
	case *appsv1beta3.EmqxBroker:
		modules := &appsv1beta3.EmqxBrokerModuleList{
			Items: obj.Spec.EmqxTemplate.Modules,
		}
		loadedModulesString = modules.String()
	case *appsv1beta3.EmqxEnterprise:
		data, _ := json.Marshal(obj.Spec.EmqxTemplate.Modules)
		loadedModulesString = string(data)
	}
	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels:    instance.GetLabels(),
			Namespace: instance.GetNamespace(),
			Name:      names.LoadedModules(),
		},
		Data: map[string]string{"loaded_modules": loadedModulesString},
	}

	annotations := sts.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations["LoadedModules/Base64EncodeConfig"] = base64.StdEncoding.EncodeToString([]byte(loadedModulesString))
	sts.Annotations = annotations

	container := sts.Spec.Template.Spec.Containers[0]
	container.VolumeMounts = append(
		container.VolumeMounts,
		corev1.VolumeMount{
			Name:      cm.Name,
			MountPath: "/mounted/modules",
		},
	)
	container.Env = append(
		container.Env,
		corev1.EnvVar{
			Name:  "EMQX_MODULES__LOADED_FILE",
			Value: "/mounted/modules/loaded_modules",
		},
	)
	sts.Spec.Template.Spec.Containers[0] = container

	sts.Spec.Template.Spec.Volumes = append(
		sts.Spec.Template.Spec.Volumes,
		corev1.Volume{
			Name: cm.Name,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: cm.Name,
					},
				},
			},
		},
	)
	return cm, sts
}

func generateLicense(instance appsv1beta3.Emqx, sts *appsv1.StatefulSet) (*corev1.Secret, *appsv1.StatefulSet) {
	names := appsv1beta3.Names{Object: instance}
	emqxEnterprise, ok := instance.(*appsv1beta3.EmqxEnterprise)
	if !ok {
		return nil, sts
	}
	if reflect.ValueOf(emqxEnterprise.GetLicense()).IsZero() {
		return nil, sts
	}
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels:    instance.GetLabels(),
			Namespace: instance.GetNamespace(),
			Name:      names.License(),
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{"emqx.lic": emqxEnterprise.GetLicense().Data},
	}
	if emqxEnterprise.GetLicense().StringData != "" {
		secret.StringData = map[string]string{"emqx.lic": emqxEnterprise.GetLicense().StringData}
	}

	container := sts.Spec.Template.Spec.Containers[0]
	container.VolumeMounts = append(
		container.VolumeMounts,
		corev1.VolumeMount{
			Name:      secret.Name,
			MountPath: "/mounted/license",
			ReadOnly:  true,
		},
	)
	container.Env = append(
		container.Env,
		corev1.EnvVar{
			Name:  "EMQX_LICENSE__FILE",
			Value: "/mounted/license/emqx.lic",
		},
	)
	sts.Spec.Template.Spec.Containers[0] = container

	sts.Spec.Template.Spec.Volumes = append(
		sts.Spec.Template.Spec.Volumes,
		corev1.Volume{
			Name: secret.Name,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secret.Name,
				},
			},
		},
	)
	return secret, sts
}

func generateVolume(instance appsv1beta3.Emqx, sts *appsv1.StatefulSet) *appsv1.StatefulSet {
	names := appsv1beta3.Names{Object: instance}

	dataName := names.Data()
	logName := names.Log()

	container := sts.Spec.Template.Spec.Containers[0]
	container.VolumeMounts = append(
		container.VolumeMounts,
		corev1.VolumeMount{
			Name:      dataName,
			MountPath: "/opt/emqx/data",
		},
		corev1.VolumeMount{
			Name:      logName,
			MountPath: "/opt/emqx/log",
		},
	)

	if reflect.ValueOf(instance.GetPersistent()).IsZero() {
		sts.Spec.Template.Spec.Volumes = append(
			sts.Spec.Template.Spec.Volumes,
			generateEmptyDirVolume(dataName),
			generateEmptyDirVolume(logName),
		)
	} else {
		sts.Spec.VolumeClaimTemplates = append(
			sts.Spec.VolumeClaimTemplates,
			generateVolumeClaimTemplate(instance, dataName),
			generateVolumeClaimTemplate(instance, logName),
		)

	}

	container.VolumeMounts = append(container.VolumeMounts, instance.GetExtraVolumeMounts()...)
	sts.Spec.Template.Spec.Containers[0] = container
	sts.Spec.Template.Spec.Volumes = append(sts.Spec.Template.Spec.Volumes, instance.GetExtraVolumes()...)
	return sts
}

func generateEmptyDirVolume(Name string) corev1.Volume {
	return corev1.Volume{
		Name: Name,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
}

func generateVolumeClaimTemplate(instance appsv1beta3.Emqx, Name string) corev1.PersistentVolumeClaim {
	template := instance.GetPersistent()
	pvc := corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      Name,
			Namespace: instance.GetNamespace(),
		},
		Spec: template,
	}
	if pvc.Spec.AccessModes == nil {
		pvc.Spec.AccessModes = []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce}
	}
	if pvc.Spec.VolumeMode == nil {
		fileSystem := corev1.PersistentVolumeFilesystem
		pvc.Spec.VolumeMode = &fileSystem
	}
	return pvc
}

func mergeEnvAndConfig(instance appsv1beta3.Emqx) (ret []corev1.EnvVar) {
	lookup := func(name string, envs []corev1.EnvVar) bool {
		for _, env := range envs {
			if env.Name == name {
				return true
			}
		}
		return false
	}

	envs := instance.GetEnv()
	emqxConfig := instance.GetEmqxConfig()

	for k, v := range emqxConfig {
		key := fmt.Sprintf("EMQX_%s", strings.ToUpper(strings.ReplaceAll(k, ".", "__")))
		if !lookup(key, envs) {
			ret = append(ret, corev1.EnvVar{Name: key, Value: v})
		}
	}

	sort.Slice(ret, func(i, j int) bool {
		return ret[i].Name < ret[j].Name
	})

	return
}

func isPluginInitialized(instance appsv1beta3.Emqx) bool {
	// Init Plugin
	for _, c := range instance.GetStatus().Conditions {
		if c.Type == appsv1beta3.ConditionPluginInitialized && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func generateReloaderContainer(r *EmqxReconciler, instance appsv1beta3.Emqx, sts *appsv1.StatefulSet) corev1.Container {
	username, password, apiPort := r.getManagementField(instance)
	container := sts.Spec.Template.Spec.Containers[0]
	return corev1.Container{
		Name:            ReloaderContainerName,
		Image:           ReloaderContainerImage,
		ImagePullPolicy: instance.GetImagePullPolicy(),
		Env:             container.Env,
		VolumeMounts:    container.VolumeMounts,
		Args: []string{
			"-u", username,
			"-p", password,
			"-P", apiPort,
		},
	}
}

func generateEmqxContainer(instance appsv1beta3.Emqx) corev1.Container {
	return corev1.Container{
		Name:            handler.EmqxContainerName,
		Image:           instance.GetImage(),
		ImagePullPolicy: instance.GetImagePullPolicy(),
		Resources:       instance.GetResource(),
		Env:             mergeEnvAndConfig(instance),
		Args:            instance.GetArgs(),
		ReadinessProbe:  instance.GetReadinessProbe(),
		LivenessProbe:   instance.GetLivenessProbe(),
		StartupProbe:    instance.GetStartupProbe(),
	}
}

func generateAnnotationByContainers(containers []corev1.Container) string {
	containerNames := []string{}
	for _, c := range containers {
		containerNames = append(containerNames, c.Name)
	}
	return strings.Join(containerNames, ",")
}
