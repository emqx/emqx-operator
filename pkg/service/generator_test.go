package service

import (
	"testing"

	"github.com/emqx/emqx-operator/api/v1alpha2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	replicas int32 = 3
	emqx           = &v1alpha2.EmqxBroker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "emqx",
			Namespace: "emqx-system",
		},
		Spec: v1alpha2.EmqxBrokerSpec{
			Replicas:           &replicas,
			Image:              "emqx/emqx:4.3.10",
			ServiceAccountName: "emqx",
			// Storage: &{},
			// AclConf:
			// 	"{allow, {user, \"dashboard\"}, subscribe, [\"$SYS/#\"]}.
			// 	{allow, {ipaddr, \"127.0.0.1\"}, pubsub, [\"$SYS/#\", \"#\"]}.
			// 	{deny, all, subscribe, [\"$SYS/#\", {eq, \"#\"}]}.
			// 	{allow, all}."
			// LoadedPluginConf: {},
			// LoadedModulesConf: {},
		},
	}
)

// TODO
// unit test for newConfigMapForAcl
func TestNewConfigMapForAcl(t *testing.T) {

	// type args struct {
	// 	t    *v1alpha2.Emqx
	// }

	// TODO
	// tests := []struct {
	// 	ns string
	// 	arg args
	// 	want string
	// }{

	// }
}
