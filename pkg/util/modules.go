package util

import (
	"fmt"

	"github.com/emqx/emqx-operator/apis/apps/v1beta1"
)

func StringEmqxBrokerLoadedModules(modules []v1beta1.EmqxBrokerModules) string {
	var p string
	for _, module := range modules {
		p = fmt.Sprintf("%s{%s, %t}.\n", p, module.Name, module.Enable)
	}
	return p
}
