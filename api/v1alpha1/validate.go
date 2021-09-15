package v1alpha1

import (
	"errors"
	"reflect"

	"github.com/cloudflare/cfssl/log"
)

//The discovery mechanism of emqx cluster only be one of the dns or k8s.
func (e *Emqx) Validate() error {

	//Validate the cluster config
	if e.Spec.Cluster.Discovery == "k8s" && e.Spec.Cluster.K8S.IsEmpty() {
		log.Error("cluster discovery mechanism must be completed")
		return errors.New("cluster mechanism validated error")
	}
	return nil
}

func (k K8S) IsEmpty() bool {
	return reflect.DeepEqual(k, K8S{})
}
