export CRD=EmqxBroker
export group=apps
export version=v1beta1

kubebuilder init --domain emqx.io

kubebuilder create api --group ${group} --version ${version} --kind ${CRD}