package util

import (
	"fmt"
	"reflect"
)

func IsNil(i interface{}) bool {
	return reflect.ValueOf(i).IsZero()
}

func GenerateHeadelssServiceName(name string) string {
	return fmt.Sprintf("%s-%s", name, "headless")
}
