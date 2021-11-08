export CRD=EmqxBroker
export group=apps
export version=v1alpha2

kubebuilder init --domain emqx.io

kubebuilder create api --group ${group} --version ${version} --kind ${CRD}