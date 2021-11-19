package service

const (
	EMQX_NAME = "emqx"

	EMQX_LIC_NAME    = "emqx-lic"
	EMQX_LIC_DIR     = "/opt/emqx/etc/emqx.lic"
	EMQX_LIC_SUBPATH = "emqx.lic"

	EMQX_DATA_DIR = "/opt/emqx/data"
	EMQX_LOG_DIR  = "/opt/emqx/log"

	EMQX_ACL_CONF_DIR     = "/opt/emqx/etc/acl.conf"
	EMQX_ACL_CONF_SUBPATH = "acl.conf"

	EMQX_LOADED_MODULES_DIR     = "/opt/emqx/data/loaded_modules"
	EMQX_LOADED_MODULES_SUBPATH = "loaded_modules"

	EMQX_LOADED_PLUGINS_DIR     = "/opt/emqx/data/loaded_plugins"
	EMQX_LOADED_PLUGINS_SUBPATH = "loaded_plugins"

	EMQX_MQTT_NAME = "mqtt"
	EMQX_MQTT_PORT = 1883

	EMQX_MQTTS_NAME = "mqtts"
	EMQX_MQTTS_PORT = 8883

	EMQX_WS_NAME = "ws"
	EMQX_WS_PORT = 8083

	EMQX_WSS_NAME = "wss"
	EMQX_WSS_PORT = 8084

	EMQX_DASHBOARD_NAME = "dashboard"
	EMQX_DASHBOARD_PORT = 18083

	EMQX_API_NAME = "api"
	EMQX_API_PORT = 8081
)
