package v1alpha2

type Labels map[string]string

func (emqx *EmqxBroker) GetLabels() map[string]string {
	return generateLabels(emqx.Name, emqx.Spec.Labels)
}

func (emqx *EmqxEnterprise) GetLabels() map[string]string {
	return generateLabels(emqx.Name, emqx.Spec.Labels)
}

func generateLabels(name string, labels Labels) Labels {
	return mergeLabels(labels, defaultLabels(name))
}

func mergeLabels(allLabels ...Labels) Labels {
	res := map[string]string{}

	for _, labels := range allLabels {
		for k, v := range labels {
			res[k] = v
		}
	}
	return res
}

func defaultLabels(name string) Labels {
	return map[string]string{
		"apps.emqx.io/managed-by": "emqx-operator",
		"apps.emqx.io/version":    "v1alpha2",
		"apps.emqx.io/instance":   name,
	}
}
