export CRD=Emqx
export group=apps
export version=v1alpha1

kubebuilder init --domain emqx.io

kubebuilder create api --group ${group} --version ${version} --kind ${CRD}