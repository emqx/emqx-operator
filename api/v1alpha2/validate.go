package v1alpha2

import (
	"fmt"
	"reflect"
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

	if reflect.ValueOf(emqx.GetReplicas()).IsZero() {
		emqx.SetReplicas(defaultEmqxBrokerNumber)
	}

	return nil
}

// Validate set the values by default if not defined and checks if the values given are valid
func (emqx EmqxEnterprise) Validate() error {
	if len(emqx.GetName()) > maxNameLength {
		return fmt.Errorf("name length can't be higher than %d", maxNameLength)
	}

	if reflect.ValueOf(emqx.GetReplicas()).IsZero() {
		emqx.SetReplicas(defaultEmqxBrokerNumber)
	}

	return nil
}
