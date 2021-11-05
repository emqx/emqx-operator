package v1alpha1

import (
	"errors"
	"fmt"
)

const (
	maxNameLength = 48

	defaultEmqxBrokerNumber = 3
)

// Validate set the values by default if not defined and checks if the values given are valid
func (e *EmqxBroker) Validate() error {
	if len(e.Name) > maxNameLength {
		return fmt.Errorf("name length can't be higher than %d", maxNameLength)
	}

	if *e.Spec.Replicas == 0 {
		*e.Spec.Replicas = defaultEmqxBrokerNumber
	} else if *e.Spec.Replicas < defaultEmqxBrokerNumber {
		return errors.New("number of emqx in spec is less than the minimum")
	}

	if e.Spec.Image == "" {
		return errors.New("image must be specified")
	}

	return nil
}
