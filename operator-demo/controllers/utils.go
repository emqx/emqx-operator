package controllers

import (
	"fmt"

	"gopkg.in/yaml.v2"
)

func createStringData(s string) string {
	d, err := yaml.Marshal(s)
	if err != nil {
		return ""
	}
	fmt.Printf("========================\n %s\n======================\n", string(d))
	return string(d)
}
