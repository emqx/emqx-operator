package controllers

const (
	EMQX_NAME                = "emqx"
	EMQX_LIC_NAME            = "emqx-lic"
	EMQX_LOG_NAME            = "emqx-log-dir"
	EMQX_DATA_NAME           = "emqx-data-dir"
	EMQX_ACL_CONF_NAME       = "emqx-acl"
	EMQX_LOADED_MODULES_NAME = "emqx-loaded-modules"
	EMQX_LOADED_PLUGINS_NAME = "emqx-loaded-plugins"
	// emqxenvName            = "cloud-env"
	EMQX_LIC_DIR     = "/opt/emqx/etc/emqx.lic"
	EMQX_LIC_SUBPATH = "emqx.lic"

	EMQX_DATA_DIR = "/opt/emqx/data/mnesia"

	EMQX_LOG_DIR = "/opt/emqx/log"

	EMQX_ACL_CONF_DIR     = "/opt/emqx/etc/acl.conf"
	EMQX_ACL_CONF_SUBPATH = "acl.conf"

	EMQX_LOADED_MODULES_DIR     = "/opt/emqx/data/loaded_modules"
	EMQX_LOADED_MODULES_SUBPATH = "loaded_modules"

	EMQX_LOADED_PLUGINS_DIR     = "/opt/emqx/data/loaded_plugins"
	EMQX_LOADED_PLUGINS_SUBPATH = "loaded_plugins"

	// emqxloadmodulesSubpath = "loaded_modules"

	SERVICE_TCP_NAME = "tcp"
	SERVICE_TCP_PORT = 1883

	SERVICE_TCPS_NAME = "tcps"
	SERVICE_TCPS_PORT = 8883

	SERVICE_WS_NAME = "ws"
	SERVICE_WS_PORT = 8083

	SERVICE_WSS_NAME = "wss"
	SERVICE_WSS_PORT = 8084

	SERVICE_DASHBOARD_NAME = "dashboard"
	SERVICE_DASHBOARD_PORT = 18083
)
