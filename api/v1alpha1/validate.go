package v1alpha1

import (
	"errors"
	"fmt"
	"reflect"
)

const (
	maxNameLength = 48

	defaultEmqxNumber = 3
)

// Validate set the values by default if not defined and checks if the values given are valid
func (e *Emqx) Validate() error {
	if len(e.Name) > maxNameLength {
		return fmt.Errorf("name length can't be higher than %d", maxNameLength)
	}

	if *e.Spec.Replicas == 0 {
		*e.Spec.Replicas = defaultEmqxNumber
	} else if *e.Spec.Replicas < defaultEmqxNumber {
		return errors.New("number of emqx in spec is less than the minimum")
	}

	if e.Spec.Image == "" {
		return errors.New("image must be specified")
	}

	//Validate the cluster config
	if e.Spec.Cluster.Discovery == "k8s" && e.Spec.Cluster.K8S.IsEmpty() {
		return errors.New("cluster mechanism validated error")
	}
	return nil
}

func (k K8S) IsEmpty() bool {
	return reflect.DeepEqual(k, K8S{})
}
