package e2e

import (
	"encoding/json"
	"fmt"
	"strings"

	crdv2 "github.com/emqx/emqx-operator/api/v2"
	. "github.com/emqx/emqx-operator/test/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// upgradeWorkaround describes version-specific manual remediation steps that
// must be applied during a rolling upgrade to unblock the operator.
// See docs/en_US/upgrade-workarounds.md for the human-readable version.
type upgradeWorkaround struct {
	Description string
	// Prepare is called after the initial cluster is deployed but before the update.
	// It applies any configuration changes needed before the upgrade by modifying
	// CR spec.
	Prepare func(emqxCR *[]byte) error
	// Precondition is polled after the image patch. Returns true when the
	// upgrade is stuck in the expected way and remediation should run.
	Precondition func(instance string) metav1.ConditionStatus
	// Remediate applies the manual fix to unblock the upgrade.
	Remediate func(instance string) error
}

func resolvePlaybook(imageInitial, imageUpgrade string) *upgradeWorkaround {
	if imageTag(imageInitial) == "6.1.0" && imageTag(imageUpgrade) == "6.2.0" {
		return &workaround610620
	}
	return nil
}

// workaround610620 provides 2 remedies for the broken update path:
//  1. Opens DS databases that are missing on new 6.2.0 nodes when upgrading from
//     a 6.1.0 cluster. Without this step the operator cannot move shard replicas
//     and the upgrade hangs.
//  2. Additionally, overrides some application environment settings so that DS
//     recover availability as soon as 6.2.0 nodes are the majority of the cluster.
var workaround610620 = upgradeWorkaround{

	Description: "open missing DS databases on new nodes",

	Prepare: func(emqxCR *[]byte) error {
		patched := PatchDocument(
			*emqxCR,
			[]byte(`{"spec": {"coreTemplate":
				{"spec": {"args": [
					"/opt/emqx/bin/emqx", "foreground",
					"-ra", "machine_upgrade_strategy", "quorum"
				]}}
			}}`),
		)
		*emqxCR = patched
		return nil
	},

	Precondition: func(instance string) metav1.ConditionStatus {
		var status crdv2.EMQXStatus
		out, _ := KubectlOut("get", "emqx", instance, "-o", "jsonpath={.status}")
		err := json.Unmarshal([]byte(out), &status)
		if err != nil || len(status.CoreNodes) == 0 || len(status.DSReplication.DBs) == 0 {
			return metav1.ConditionUnknown
		}
		totalTransitions := 0
		affectedTransitions := 0
		affectedDBs := map[string]bool{
			"mq_state":             true,
			"mq_message_regular":   true,
			"mq_message_lastvalue": true,
		}
		for _, db := range status.DSReplication.DBs {
			totalTransitions += int(db.NumTransitions) / int(db.NumShards)
			if affectedDBs[db.Name] {
				affectedTransitions += int(db.NumTransitions) / int(db.NumShards)
			}
		}
		if totalTransitions == 0 || totalTransitions > affectedTransitions {
			return metav1.ConditionUnknown
		}
		if affectedTransitions < len(affectedDBs)*2 {
			return metav1.ConditionFalse
		}
		return metav1.ConditionTrue
	},

	Remediate: func(instance string) error {
		var podList corev1.PodList
		out, err := KubectlOut("get", "pod",
			"--selector", coreSelector(instance),
			"--field-selector", "status.phase=Running",
			"-o", "json",
		)
		if err != nil {
			return err
		}
		err = json.Unmarshal([]byte(out), &podList)
		if err != nil {
			return err
		}
		for _, pod := range podList.Items {
			for _, cmd := range []string{
				"emqx_mq_message_db:open()",
				"emqx_mq_state_storage:open_db()",
			} {
				err = Kubectl("exec", pod.Name, "--", "emqx", "eval", cmd)
				if err != nil {
					return err
				}
			}
		}
		return nil
	},
}

func coreSelector(instance string) string {
	return fmt.Sprintf("apps.emqx.io/instance=%s,apps.emqx.io/db-role=core", instance)
}

func imageTag(image string) string {
	parts := strings.SplitN(image, ":", 2)
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}
