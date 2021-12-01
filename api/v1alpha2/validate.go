package v1alpha2

import (
	"fmt"
)

const maxNameLength = 48

// Validate set the values by default if not defined and checks if the values given are valid
func (emqx EmqxBroker) Validate() error {
	if len(emqx.GetName()) > maxNameLength {
		return fmt.Errorf("name length can't be higher than %d", maxNameLength)
	}

	return nil
}

// Validate set the values by default if not defined and checks if the values given are valid
func (emqx EmqxEnterprise) Validate() error {
	if len(emqx.GetName()) > maxNameLength {
		return fmt.Errorf("name length can't be higher than %d", maxNameLength)
	}

	return nil
}
