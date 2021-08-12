package v1alpha1

//+kubebuilder:validation:Optional
type Plugins struct {

	//+kubebuilder:default:="etc/plugins"
	EtcDir string `json:"etc_dir,omitempty"`

	//+kubebuilder:default:="etc/loaded_plugins"
	LoadedFile string `json:"loaded_file,omitempty"`

	//+kubebuilder:default:="plugins/"
	ExpandPluginsDir string `json:"expand_plugins_dir,omitempty"`
}
