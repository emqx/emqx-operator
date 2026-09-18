package upgrade

import (
	"encoding/json"
	"fmt"
	"strings"

	crd "github.com/emqx/emqx-operator/api/v3beta1"
	"github.com/emqx/emqx-operator/test/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

var bootstrapAuthWorkaround = upgradeWorkaround{
	Description: "reload bootstrap API keys on a remaining 6.2 core",
	Precondition: func(instance string) precondition {
		_, condition := lookupBootstrapAuthTarget(instance)
		return condition
	},
	Remediate: func(instance string) error {
		// Re-read node status rather than reusing an earlier target.
		node, condition := lookupBootstrapAuthTarget(instance)
		if condition.Status == condFalse {
			return nil
		}
		if condition.Status != condTrue {
			return condition
		}
		out, err := util.KubectlOut("exec", node.PodName, "-c", crd.DefaultContainerName,
			"--", "emqx", "eval", "emqx_mgmt_auth:try_init_bootstrap_file().")
		if err != nil {
			return err
		}
		if strings.TrimSpace(out) != "ok" {
			return fmt.Errorf("bootstrap API key reload on %s did not return ok", node.PodName)
		}
		return nil
	},
}

// Read the mounted credentials inside the pod so they never enter test logs.
const bootstrapAuthProbe = `
curl --silent --show-error --max-time 10 \
    --user "$(sed -n '/^emqx-operator-controller:/p' /opt/emqx/etc/bootstrap_api_keys)" \
    http://127.0.0.1:18083/api/v5/listeners
`

func lookupBootstrapAuthTarget(instance string) (*crd.EMQXNode, precondition) {
	var emqx crd.EMQX
	out, _ := util.KubectlOut("get", "emqx", instance, "-o", "json")
	err := json.Unmarshal([]byte(out), &emqx)
	if err != nil {
		return nil, preconditionUnknown(err.Error())
	}
	ready := emqx.Status.GetCondition(crd.Ready)
	versions := coreNodeVersions(emqx.Status.NodesWithRole(crd.RoleCore))
	node := preferredBootstrapAuthNode(emqx.Status.NodesWithRole(crd.RoleCore))
	if ready != nil && ready.Status == metav1.ConditionTrue {
		return nil, precondition{Status: condFalse}
	}
	if !versions.Has("6.3.0") || node == nil {
		return nil, preconditionUnknown("expecting both 6.2.x and 6.3.0 running cores")
	}
	out, err = util.KubectlOut("exec", node.PodName,
		"-c", crd.DefaultContainerName,
		"--", "sh", "-ec", bootstrapAuthProbe)
	if err != nil {
		return nil, preconditionUnknown(err.Error())
	}
	if strings.Contains(out, "BAD_API_KEY_OR_SECRET") {
		return node, precondition{Status: condTrue}
	}
	return node, precondition{Status: condFalse}
}

// coreNodeVersions lists versions of running cores with known pods.
func coreNodeVersions(nodes []crd.EMQXNode) sets.Set[string] {
	versions := sets.New[string]()
	for _, node := range nodes {
		if node.Status == "running" && node.PodName != "" && node.Version != "" {
			versions.Insert(node.Version)
		}
	}
	return versions
}

// preferredBootstrapAuthNode selects the first running 6.2.x core by pod name.
func preferredBootstrapAuthNode(nodes []crd.EMQXNode) *crd.EMQXNode {
	var target *crd.EMQXNode
	for i := range nodes {
		node := &nodes[i]
		if node.Status != "running" || node.PodName == "" || minorVersion(node.Version) != "6.2.x" {
			continue
		}
		if target == nil || node.PodName < target.PodName {
			target = node
		}
	}
	return target
}
