package constants

const (
	//  The default value for EMQ X Cluster Name
	EMQX_NAME = "emqx"

	// The constant value for configmap
	EMQX_LIC_NAME    = "emqx-lic"
	EMQX_LIC_DIR     = "/opt/emqx/etc/emqx.lic"
	EMQX_LIC_SUBPATH = "emqx.lic"

	EMQX_DATA_DIR = "/opt/emqx/data"
	EMQX_LOG_DIR  = "/opt/emqx/log"

	// The constant key-value for labels
	OPERATOR_NAME        = "emqx-operator"
	LABEL_MANAGED_BY_KEY = "apps.emqx.io/managed-by"
	LABEL_NAME_KEY       = "emqx-operator/v1alpha2"
)
