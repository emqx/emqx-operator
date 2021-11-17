package v1alpha2

import (
	"errors"
	"fmt"
)

const (
	maxNameLength = 48

	defaultEmqxBrokerNumber = 3
)

// Validate set the values by default if not defined and checks if the values given are valid
func (emqx EmqxBroker) Validate() error {
	if len(emqx.GetName()) > maxNameLength {
		return fmt.Errorf("name length can't be higher than %d", maxNameLength)
	}

	if *emqx.GetReplicas() == 0 {
		emqx.SetReplicas(defaultEmqxBrokerNumber)
	} else if *emqx.GetReplicas() < defaultEmqxBrokerNumber {
		return errors.New("number of emqx in spec is less than the minimum")
	}

	if emqx.GetImage() == "" {
		return errors.New("image must be specified")
	}

	return nil
}
