package v1alpha2

import (
	"errors"
	"fmt"
)

const (
	maxNameLength = 48

	defaultEmqxBrokerNumber = 3
)

type void struct{}

var (
	voidValue        void
	defaultPortNames = map[string]void{
		"mqtt":      voidValue,
		"mqtts":     voidValue,
		"ws":        voidValue,
		"wss":       voidValue,
		"dashboard": voidValue,
		"api":       voidValue,
	}
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

	if emqx.GetListener() != nil {
		if !validatePortName(emqx) {
			return errors.New("port name must be specified as mqtt, mqtts, ws, wss, dashboard, api ")
		}
	}
	return nil
}

// Validate set the values by default if not defined and checks if the values given are valid
func (emqx EmqxEnterprise) Validate() error {
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

func validatePortName(emqx EmqxBroker) bool {
	for _, port := range emqx.GetListener().Ports {
		if _, ok := defaultPortNames[port.Name]; !ok {
			return false
		}
	}
	return true
}
