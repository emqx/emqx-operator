package v1alpha1

import (
	"errors"
	"fmt"
	"reflect"
)

const (
	maxNameLength = 48

	defaultEmqxNumber = 1
	defaultEmqxImage  = "emqx-ee:4.3.8"
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
		e.Spec.Image = defaultEmqxImage
	}

	// if r.Spec.Config == nil {
	// 	r.Spec.Config = make(map[string]string)
	// }

	// if !e.Spec.DisablePersistence {
	// 	enablePersistence(r.Spec.Config)
	// } else {
	// 	disablePersistence(r.Spec.Config)
	// }

	//Validate the cluster config
	if e.Spec.Cluster.Discovery == "k8s" && e.Spec.Cluster.K8S.IsEmpty() {
		return errors.New("cluster mechanism validated error")
	}
	return nil
}

func (k K8S) IsEmpty() bool {
	return reflect.DeepEqual(k, K8S{})
}
