package v1beta1

//+kubebuilder:object:generate=true
type TelegrafTemplate struct {
	Name  *string `json:"name,omitempty"`
	Image *string `json:"image,omitempty"`
	Conf  *string `json:"conf,omitempty"`
}

func generateTelegrafTemplate(telegrafTemplate *TelegrafTemplate) *TelegrafTemplate {
	if telegrafTemplate.Name == nil {
		name := "telegraf"
		telegrafTemplate.Name = &name
	}
	return telegrafTemplate
}

func (telegrafTemplate *TelegrafTemplate) GetName() string {
	return *telegrafTemplate.Name
}

func (telegrafTemplate *TelegrafTemplate) GetImage() string {
	return *telegrafTemplate.Image
}

func (telegrafTemplate *TelegrafTemplate) GetConf() string {
	return *telegrafTemplate.Conf
}
