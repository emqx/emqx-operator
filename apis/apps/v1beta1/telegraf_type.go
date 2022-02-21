package v1beta1

//+kubebuilder:object:generate=true
type TelegrafTemplate struct {
	Name  string  `json:"name,omitempty"`
	Image string  `json:"image,omitempty"`
	Conf  *string `json:"conf,omitempty"`
}

func generateTelegrafTemplate(telegrafTemplate *TelegrafTemplate) *TelegrafTemplate {
	if telegrafTemplate.Name == "" {
		name := "telegraf"
		telegrafTemplate.Name = name
	}
	return telegrafTemplate
}
