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

package v1beta4

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	appsv1beta4 "github.com/emqx/emqx-operator/apis/apps/v1beta4"
	"github.com/emqx/emqx-operator/internal/handler"
	"github.com/sethvargo/go-password/password"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ReloaderContainerName = "reloader"
	defUsername           = "emqx_operator_controller"
)

func generateInitPluginList(instance appsv1beta4.Emqx, existPluginList *appsv1beta4.EmqxPluginList) []client.Object {
	matchedPluginList := []appsv1beta4.EmqxPlugin{}
	for _, existPlugin := range existPluginList.Items {
		selector, _ := labels.ValidatedSelectorFromSet(existPlugin.Spec.Selector)
		if selector.Empty() || !selector.Matches(labels.Set(instance.GetLabels())) {
			continue
		}
		matchedPluginList = append(matchedPluginList, existPlugin)
	}

	isExistPlugin := func(pluginName string, pluginList []appsv1beta4.EmqxPlugin) bool {
		for _, plugin := range pluginList {
			if plugin.Spec.PluginName == pluginName {
				return true
			}
		}
		return false
	}

	pluginList := []client.Object{}
	// Default plugins
	if !isExistPlugin("emqx_eviction_agent", matchedPluginList) {
		emqxRuleEngine := &appsv1beta4.EmqxPlugin{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apps.emqx.io/v1beta4",
				Kind:       "EmqxPlugin",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        fmt.Sprintf("%s-eviction-agent", instance.GetName()),
				Namespace:   instance.GetNamespace(),
				Labels:      instance.GetLabels(),
				Annotations: instance.GetAnnotations(),
			},
			Spec: appsv1beta4.EmqxPluginSpec{
				PluginName: "emqx_eviction_agent",
				Selector:   instance.GetLabels(),
				Config:     map[string]string{},
			},
		}
		pluginList = append(pluginList, emqxRuleEngine)
	}

	if !isExistPlugin("emqx_node_rebalance", matchedPluginList) {
		emqxRuleEngine := &appsv1beta4.EmqxPlugin{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apps.emqx.io/v1beta4",
				Kind:       "EmqxPlugin",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        fmt.Sprintf("%s-node-rebalance", instance.GetName()),
				Namespace:   instance.GetNamespace(),
				Labels:      instance.GetLabels(),
				Annotations: instance.GetAnnotations(),
			},
			Spec: appsv1beta4.EmqxPluginSpec{
				PluginName: "emqx_node_rebalance",
				Selector:   instance.GetLabels(),
				Config:     map[string]string{},
			},
		}
		pluginList = append(pluginList, emqxRuleEngine)
	}

	if !isExistPlugin("emqx_rule_engine", matchedPluginList) {
		emqxRuleEngine := &appsv1beta4.EmqxPlugin{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apps.emqx.io/v1beta4",
				Kind:       "EmqxPlugin",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        fmt.Sprintf("%s-rule-engine", instance.GetName()),
				Namespace:   instance.GetNamespace(),
				Labels:      instance.GetLabels(),
				Annotations: instance.GetAnnotations(),
			},
			Spec: appsv1beta4.EmqxPluginSpec{
				PluginName: "emqx_rule_engine",
				Selector:   instance.GetLabels(),
				Config:     map[string]string{},
			},
		}
		pluginList = append(pluginList, emqxRuleEngine)
	}

	if !isExistPlugin("emqx_retainer", matchedPluginList) {
		emqxRetainer := &appsv1beta4.EmqxPlugin{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apps.emqx.io/v1beta4",
				Kind:       "EmqxPlugin",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        fmt.Sprintf("%s-retainer", instance.GetName()),
				Namespace:   instance.GetNamespace(),
				Labels:      instance.GetLabels(),
				Annotations: instance.GetAnnotations(),
			},
			Spec: appsv1beta4.EmqxPluginSpec{
				PluginName: "emqx_retainer",
				Selector:   instance.GetLabels(),
				Config:     map[string]string{},
			},
		}
		pluginList = append(pluginList, emqxRetainer)
	}

	_, ok := instance.(*appsv1beta4.EmqxEnterprise)
	if ok && !isExistPlugin("emqx_modules", matchedPluginList) {
		emqxModules := &appsv1beta4.EmqxPlugin{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apps.emqx.io/v1beta4",
				Kind:       "EmqxPlugin",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        fmt.Sprintf("%s-modules", instance.GetName()),
				Namespace:   instance.GetNamespace(),
				Labels:      instance.GetLabels(),
				Annotations: instance.GetAnnotations(),
			},
			Spec: appsv1beta4.EmqxPluginSpec{
				PluginName: "emqx_modules",
				Selector:   instance.GetLabels(),
				Config:     map[string]string{},
			},
		}

		pluginList = append(pluginList, emqxModules)
	}

	if ok && !isExistPlugin("emqx_schema_registry", matchedPluginList) {
		emqxSchemaRegistry := &appsv1beta4.EmqxPlugin{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apps.emqx.io/v1beta4",
				Kind:       "EmqxPlugin",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        fmt.Sprintf("%s-schema-registry", instance.GetName()),
				Namespace:   instance.GetNamespace(),
				Labels:      instance.GetLabels(),
				Annotations: instance.GetAnnotations(),
			},
			Spec: appsv1beta4.EmqxPluginSpec{
				PluginName: "emqx_schema_registry",
				Selector:   instance.GetLabels(),
				Config:     map[string]string{},
			},
		}

		pluginList = append(pluginList, emqxSchemaRegistry)
	}

	return pluginList
}

func generateDefaultPluginsConfig(instance appsv1beta4.Emqx) *corev1.ConfigMap {
	names := appsv1beta4.Names{Object: instance}

	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        names.PluginsConfig(),
			Namespace:   instance.GetNamespace(),
			Labels:      instance.GetLabels(),
			Annotations: instance.GetAnnotations(),
		},
		Data: map[string]string{
			"emqx_node_rebalance.conf":    "",
			"emqx_eviction_agent.conf":    "",
			"emqx_modules.conf":           "",
			"emqx_management.conf":        "management.listener.http = 8081\nmanagement.default_application.id = admin\nmanagement.default_application.secret = public",
			"emqx_dashboard.conf":         "dashboard.listener.http = 18083\ndashboard.default_user.login = admin\ndashboard.default_user.password = public",
			"emqx_rule_engine.conf":       "",
			"emqx_retainer.conf":          "",
			"emqx_telemetry.conf":         "",
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

	return cm
}

func generateLicense(instance appsv1beta4.Emqx) *corev1.Secret {
	enterprise, ok := instance.(*appsv1beta4.EmqxEnterprise)
	if !ok {
		return nil
	}
	names := appsv1beta4.Names{Object: instance}
	license := enterprise.Spec.License
	if len(license.Data) == 0 && len(license.StringData) == 0 {
		return nil
	}

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        names.License(),
			Namespace:   instance.GetNamespace(),
			Labels:      instance.GetLabels(),
			Annotations: instance.GetAnnotations(),
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{"emqx.lic": license.Data},
	}
	if license.StringData != "" {
		secret.StringData = map[string]string{"emqx.lic": license.StringData}
	}
	return secret
}

func generateEmqxACL(instance appsv1beta4.Emqx) *corev1.ConfigMap {
	names := appsv1beta4.Names{Object: instance}

	var aclString string
	for _, rule := range instance.GetSpec().GetTemplate().Spec.EmqxContainer.EmqxACL {
		aclString += fmt.Sprintf("%s\n", rule)
	}

	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        names.ACL(),
			Namespace:   instance.GetNamespace(),
			Labels:      instance.GetLabels(),
			Annotations: instance.GetAnnotations(),
		},
		Data: map[string]string{"acl.conf": aclString},
	}
}

func generateHeadlessService(instance appsv1beta4.Emqx) *corev1.Service {
	names := appsv1beta4.Names{Object: instance}

	headlessSvc := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        names.HeadlessSvc(),
			Namespace:   instance.GetNamespace(),
			Labels:      instance.GetLabels(),
			Annotations: instance.GetAnnotations(),
		},
		Spec: corev1.ServiceSpec{
			Selector:                 instance.GetLabels(),
			Type:                     corev1.ServiceTypeClusterIP,
			ClusterIP:                corev1.ClusterIPNone,
			PublishNotReadyAddresses: true,
			Ports: []corev1.ServicePort{
				{
					Name:       "http-management-8081",
					Port:       8081,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(8081),
				},
			},
		},
	}
	return headlessSvc
}

func generateStatefulSet(instance appsv1beta4.Emqx) *appsv1.StatefulSet {
	names := appsv1beta4.Names{Object: instance}

	emqxTemplate := instance.GetSpec().GetTemplate()

	reloaderContainer := corev1.Container{
		Name:            ReloaderContainerName,
		Image:           instance.GetSpec().GetReloaderImage(),
		ImagePullPolicy: emqxTemplate.Spec.EmqxContainer.Image.PullPolicy,
		Args: []string{
			"-u", "admin",
			"-p", "public",
			"-P", "8081",
		},
		EnvFrom:      emqxTemplate.Spec.EmqxContainer.DeepCopy().EnvFrom,
		Env:          mergeEnvAndConfig(instance),
		VolumeMounts: emqxTemplate.Spec.EmqxContainer.DeepCopy().VolumeMounts,
	}

	emqxContainer := corev1.Container{
		Name:            EmqxContainerName,
		Image:           appsv1beta4.GetEmqxImage(instance),
		ImagePullPolicy: emqxTemplate.Spec.EmqxContainer.Image.PullPolicy,
		Command:         emqxTemplate.Spec.EmqxContainer.Command,
		Args:            emqxTemplate.Spec.EmqxContainer.Args,
		WorkingDir:      emqxTemplate.Spec.EmqxContainer.WorkingDir,
		Ports:           emqxTemplate.Spec.EmqxContainer.Ports,
		EnvFrom:         emqxTemplate.Spec.EmqxContainer.EnvFrom,
		Env:             mergeEnvAndConfig(instance),
		Resources:       emqxTemplate.Spec.EmqxContainer.Resources,
		VolumeMounts: append(emqxTemplate.Spec.EmqxContainer.VolumeMounts, corev1.VolumeMount{
			Name:      "emqx-log",
			MountPath: "/opt/emqx/log",
		}),
		VolumeDevices:            emqxTemplate.Spec.EmqxContainer.VolumeDevices,
		LivenessProbe:            emqxTemplate.Spec.EmqxContainer.LivenessProbe,
		ReadinessProbe:           emqxTemplate.Spec.EmqxContainer.ReadinessProbe,
		StartupProbe:             emqxTemplate.Spec.EmqxContainer.StartupProbe,
		Lifecycle:                emqxTemplate.Spec.EmqxContainer.Lifecycle,
		TerminationMessagePath:   emqxTemplate.Spec.EmqxContainer.TerminationMessagePath,
		TerminationMessagePolicy: emqxTemplate.Spec.EmqxContainer.TerminationMessagePolicy,
		SecurityContext:          emqxTemplate.Spec.EmqxContainer.SecurityContext,
		Stdin:                    emqxTemplate.Spec.EmqxContainer.Stdin,
		StdinOnce:                emqxTemplate.Spec.EmqxContainer.StdinOnce,
		TTY:                      emqxTemplate.Spec.EmqxContainer.TTY,
	}

	podTemplate := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      emqxTemplate.Labels,
			Annotations: emqxTemplate.Annotations,
		},
		Spec: corev1.PodSpec{
			ReadinessGates: []corev1.PodReadinessGate{
				{
					ConditionType: appsv1beta4.PodOnServing,
				},
			},
			Affinity:            emqxTemplate.Spec.Affinity,
			Tolerations:         emqxTemplate.Spec.Tolerations,
			NodeName:            emqxTemplate.Spec.NodeName,
			NodeSelector:        emqxTemplate.Spec.NodeSelector,
			ServiceAccountName:  emqxTemplate.Spec.ServiceAccountName,
			ImagePullSecrets:    emqxTemplate.Spec.ImagePullSecrets,
			InitContainers:      emqxTemplate.Spec.InitContainers,
			EphemeralContainers: emqxTemplate.Spec.EphemeralContainers,
			Containers: append([]corev1.Container{
				emqxContainer, reloaderContainer,
			}, emqxTemplate.Spec.ExtraContainers...),

			SecurityContext: emqxTemplate.Spec.PodSecurityContext,
			Volumes: append(emqxTemplate.Spec.Volumes, corev1.Volume{
				Name: "emqx-log",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			}),
		},
	}
	podTemplate.Annotations = handler.SetManagerContainerAnnotation(podTemplate.Annotations, podTemplate.Spec.Containers)

	sts := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        instance.GetName(),
			Namespace:   instance.GetNamespace(),
			Labels:      instance.GetLabels(),
			Annotations: instance.GetAnnotations(),
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: names.HeadlessSvc(),
			Replicas:    instance.GetSpec().GetReplicas(),
			Selector: &metav1.LabelSelector{
				MatchLabels: podTemplate.Labels,
			},
			PodManagementPolicy: appsv1.ParallelPodManagement,
			Template:            podTemplate,
		},
	}

	if sts.Annotations != nil {
		// Delete needless annotations from EMQX Custom Resource
		delete(sts.Annotations, "kubectl.kubernetes.io/last-applied-configuration")
	}

	if instance.GetSpec().GetPersistent() == nil {
		sts.Spec.Template.Spec.Containers[0].VolumeMounts = append(
			sts.Spec.Template.Spec.Containers[0].VolumeMounts,
			corev1.VolumeMount{
				Name:      names.Data(),
				MountPath: "/opt/emqx/data",
			},
		)
		sts.Spec.Template.Spec.Volumes = append(
			sts.Spec.Template.Spec.Volumes,
			corev1.Volume{
				Name: names.Data(),
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		)
	} else {
		sts.Spec.Template.Spec.Containers[0].VolumeMounts = append(
			sts.Spec.Template.Spec.Containers[0].VolumeMounts,
			corev1.VolumeMount{
				Name:      instance.GetSpec().GetPersistent().ObjectMeta.Name,
				MountPath: "/opt/emqx/data",
			},
		)
		sts.Spec.VolumeClaimTemplates = append(
			sts.Spec.VolumeClaimTemplates,
			corev1.PersistentVolumeClaim{
				ObjectMeta: instance.GetSpec().GetPersistent().ObjectMeta,
				Spec:       instance.GetSpec().GetPersistent().Spec,
			},
		)
	}

	return sts
}

func updateStatefulSetForPluginsConfig(sts *appsv1.StatefulSet, pluginsConfig *corev1.ConfigMap) *appsv1.StatefulSet {
	return updateEnvAndVolumeForSts(sts,
		corev1.EnvVar{
			Name:  "EMQX_PLUGINS__ETC_DIR",
			Value: "/mounted/plugins/etc",
		},
		corev1.VolumeMount{
			Name:      pluginsConfig.Name,
			MountPath: "/mounted/plugins/etc",
		},
		corev1.Volume{
			Name: pluginsConfig.Name,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: pluginsConfig.Name,
					},
				},
			},
		},
	)
}

func updateStatefulSetForLicense(sts *appsv1.StatefulSet, license *corev1.Secret) *appsv1.StatefulSet {
	if license == nil {
		return sts
	}
	fileName := "emqx.lic"
	for k := range license.Data {
		fileName = k
		break
	}

	return updateEnvAndVolumeForSts(sts,
		corev1.EnvVar{
			Name:  "EMQX_LICENSE__FILE",
			Value: filepath.Join("/mounted/license", fileName),
		},
		corev1.VolumeMount{
			Name:      license.Name,
			MountPath: "/mounted/license",
			ReadOnly:  true,
		},
		corev1.Volume{
			Name: license.Name,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: license.Name,
				},
			},
		},
	)
}

func updateStatefulSetForACL(sts *appsv1.StatefulSet, acl *corev1.ConfigMap) *appsv1.StatefulSet {
	return updateEnvAndVolumeForSts(sts,
		corev1.EnvVar{
			Name:  "EMQX_ACL_FILE",
			Value: "/mounted/acl/acl.conf",
		},
		corev1.VolumeMount{
			Name:      acl.Name,
			MountPath: "/mounted/acl",
		},
		corev1.Volume{
			Name: acl.Name,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: acl.Name,
					},
				},
			},
		},
	)
}

func updateEnvAndVolumeForSts(sts *appsv1.StatefulSet, envVar corev1.EnvVar, volumeMount corev1.VolumeMount, volume corev1.Volume) *appsv1.StatefulSet {
	emqxContainerIndex := 0
	reloaderContainerIndex := 1

	isNotExistVolume := func(volume corev1.Volume) bool {
		for _, v := range sts.Spec.Template.Spec.Volumes {
			if v.Name == volume.Name {
				return false
			}
		}
		return true
	}

	isNotExistVolumeVolumeMount := func(volumeMount corev1.VolumeMount) bool {
		for _, v := range sts.Spec.Template.Spec.Containers[emqxContainerIndex].VolumeMounts {
			if v.Name == volumeMount.Name {
				return false
			}
		}
		return true
	}

	isNotExistEnv := func(envVar corev1.EnvVar) bool {
		for _, v := range sts.Spec.Template.Spec.Containers[emqxContainerIndex].Env {
			if v.Name == envVar.Name {
				return false
			}
		}
		return true
	}

	if isNotExistVolume(volume) {
		sts.Spec.Template.Spec.Volumes = append(
			sts.Spec.Template.Spec.Volumes,
			volume,
		)
	}

	if isNotExistVolumeVolumeMount(volumeMount) {
		sts.Spec.Template.Spec.Containers[emqxContainerIndex].VolumeMounts = append(
			sts.Spec.Template.Spec.Containers[emqxContainerIndex].VolumeMounts,
			volumeMount,
		)
		sts.Spec.Template.Spec.Containers[reloaderContainerIndex].VolumeMounts = append(
			sts.Spec.Template.Spec.Containers[reloaderContainerIndex].VolumeMounts,
			volumeMount,
		)
	}

	if isNotExistEnv(envVar) {
		sts.Spec.Template.Spec.Containers[emqxContainerIndex].Env = append(
			sts.Spec.Template.Spec.Containers[emqxContainerIndex].Env,
			envVar,
		)
		sts.Spec.Template.Spec.Containers[reloaderContainerIndex].Env = append(
			sts.Spec.Template.Spec.Containers[reloaderContainerIndex].Env,
			envVar,
		)
	}

	return sts
}

func mergeEnvAndConfig(instance appsv1beta4.Emqx, extraEnvs ...corev1.EnvVar) []corev1.EnvVar {
	lookup := func(name string, envs []corev1.EnvVar) bool {
		for _, env := range envs {
			if env.Name == name {
				return true
			}
		}
		return false
	}

	container := instance.GetSpec().GetTemplate().Spec.EmqxContainer

	envs := append(container.DeepCopy().Env, extraEnvs...)
	emqxConfig := container.DeepCopy().EmqxConfig

	for k, v := range emqxConfig {
		key := fmt.Sprintf("EMQX_%s", strings.ToUpper(strings.ReplaceAll(k, ".", "__")))
		if !lookup(key, envs) {
			envs = append(envs, corev1.EnvVar{Name: key, Value: v})
		}
	}

	sort.Slice(envs, func(i, j int) bool {
		return envs[i].Name < envs[j].Name
	})
	return envs
}

func generateBootstrapUserSecret(instance appsv1beta4.Emqx) *corev1.Secret {
	names := appsv1beta4.Names{Object: instance}

	bootstrapUsers := ""
	for _, apiKey := range instance.GetSpec().GetTemplate().Spec.EmqxContainer.BootstrapAPIKeys {
		bootstrapUsers += apiKey.Key + ":" + apiKey.Secret + "\n"
	}

	defPassword, _ := password.Generate(64, 10, 0, true, true)
	bootstrapUsers += defUsername + ":" + defPassword

	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        names.BootstrapUser(),
			Namespace:   instance.GetNamespace(),
			Labels:      instance.GetLabels(),
			Annotations: instance.GetAnnotations(),
		},
		StringData: map[string]string{
			"bootstrap_user": bootstrapUsers,
		},
	}
}

func updateStatefulSetForBootstrapUser(sts *appsv1.StatefulSet, bootstrapUser *corev1.Secret) *appsv1.StatefulSet {
	return updateEnvAndVolumeForSts(sts,
		corev1.EnvVar{
			Name:  "EMQX_MANAGEMENT__BOOTSTRAP_APPS_FILE",
			Value: "/opt/emqx/data/bootstrap_user",
		},
		corev1.VolumeMount{
			Name:      "bootstrap-user",
			MountPath: "/opt/emqx/data/bootstrap_user",
			SubPath:   "bootstrap_user",
			ReadOnly:  true,
		},
		corev1.Volume{
			Name: "bootstrap-user",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: bootstrapUser.Name,
				},
			},
		},
	)
}
