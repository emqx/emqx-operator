package controllers

const (
	EMQX_NAME      = "emqx"
	EMQX_LIC_NAME  = "emqx-lic"
	EMQX_LOG_NAME  = "emqx-log-dir"
	EMQX_DATA_NAME = "emqx-data-dir"
	// EMQX_LOAD_MODULESName = "emqx-loaded-modules"
	// emqxenvName            = "cloud-env"
	EMQX_LIC_DIR     = "/opt/emqx/etc/emqx.lic"
	EMQX_LIC_SUBPATH = "emqx.lic"
	EMQX_DATA_DIR    = "/opt/emqx/data/mnesia"
	EMQX_LOG_DIR     = "/opt/emqx/log"
	// emqxloadmodulesDir     = "/opt/emqx/data/loaded_modules"
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
