package v1beta1

type Labels map[string]string

func generateLabels(emqx Emqx) Labels {
	var metaLabels Labels
	var specLabels Labels

	if broker, ok := emqx.(*EmqxBroker); ok {
		metaLabels = broker.Labels
		specLabels = broker.Spec.Labels
	}
	if enterprise, ok := emqx.(*EmqxEnterprise); ok {
		metaLabels = enterprise.Labels
		specLabels = enterprise.Spec.Labels
	}
	if metaLabels == nil {
		metaLabels = make(Labels)
	}
	if specLabels == nil {
		specLabels = make(Labels)
	}

	for key, value := range metaLabels {
		specLabels[key] = value

	}

	specLabels["apps.emqx.io/managed-by"] = "emqx-operator"
	specLabels["apps.emqx.io/instance"] = emqx.GetName()
	return specLabels
}
