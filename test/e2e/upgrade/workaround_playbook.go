package upgrade

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/version"
)

// UpgradePlaybook contains manual workarounds exercised by the upgrade E2E suite.
//
// To add one, register its initial and target image tags here and document its
// symptoms, manual commands, and verification in docs/en_US/upgrade-workarounds.md.
// * Exact pairs take precedence over initial minor-series keys such as "6.2.x".
// * Target tag is always exact.
//
// Validate the affected pair on the isolated Kind cluster:
//
//	make test-e2e-upgrade \
//	  TEST_E2E_UPGRADE_IMAGE_INITIAL=emqx/emqx:6.2.2 \
//	  TEST_E2E_UPGRADE_IMAGE_UPGRADE=emqx/emqx:6.3.0 \
//	  TEST_E2E_DIAGNOSTIC_REPORT_PATH=test/_reports
var UpgradePlaybook = playbook{
	// Trouble:
	// During a mixed-version upgrade, a 6.3.0 node overwrites the shared
	// bootstrap API key records with records that 6.2 cores cannot use.
	// The older cores return BAD_API_KEY_OR_SECRET, stalling reconciliation.
	// Workaround:
	// Run `emqx_mgmt_auth:try_init_bootstrap_file()` on a remaining 6.2 core
	// to reload its bootstrap file and replace the shared records, restoring
	// API access so the operator can continue the upgrade.
	{Initial: "6.2.x", Upgrade: "6.3.0"}: bootstrapAuthWorkaround,
}

// upgradeWorkaround codifies manual intervention for one known upgrade path.
type upgradeWorkaround struct {
	Description string
	// Prepare optionally modifies the desired CR before the image update is
	// applied. The supplied document already contains the target image.
	Prepare func(emqxCR *[]byte) error
	// Precondition returns True to remediate, False to skip, or Unknown with
	// diagnostic details to retry within a bounded timeout. Nil means True.
	Precondition func(instance string) precondition
	// Remediate applies an idempotent manual fix. It may be retried and must
	// tolerate a previous attempt that completed only some of its steps.
	Remediate func(instance string) error
}

type preconditionStatus string

const (
	condTrue    preconditionStatus = "True"
	condFalse   preconditionStatus = "False"
	condUnknown preconditionStatus = "Unknown"
)

// precondition describes whether a workaround applies to the observed state.
// Message contains the observation or error details.
type precondition struct {
	Status  preconditionStatus
	Message string
}

func preconditionUnknown(message string) precondition {
	return precondition{Status: condUnknown, Message: message}
}

func (c precondition) Error() string {
	return fmt.Sprintf("precondition %s: %s", c.Status, c.Message)
}

type upgradePath struct {
	Initial string
	Upgrade string
}

type playbook map[upgradePath]upgradeWorkaround

func (p playbook) resolve(imageInitial, imageUpgrade string) (upgradeWorkaround, bool) {
	initial, upgrade := imageTag(imageInitial), imageTag(imageUpgrade)
	if initial == "" || upgrade == "" {
		return upgradeWorkaround{}, false
	}
	if recipe, ok := p[upgradePath{Initial: initial, Upgrade: upgrade}]; ok {
		return recipe, true
	}
	recipe, ok := p[upgradePath{Initial: minorVersion(initial), Upgrade: upgrade}]
	return recipe, ok
}

// minorVersion groups semantic versions by major and minor, including
// prereleases and versions with build metadata.
func minorVersion(tag string) string {
	v, err := version.ParseSemantic(tag)
	if err != nil || v.String() != tag {
		return ""
	}
	return fmt.Sprintf("%d.%d.x", v.Major(), v.Minor())
}

func imageTag(image string) string {
	colon := strings.LastIndexByte(image, ':')
	if colon <= strings.LastIndexByte(image, '/') || colon <= 0 {
		return ""
	}
	return image[colon+1:]
}

func (w upgradeWorkaround) apply(instance string) error {
	condition := precondition{Status: condTrue}
	if w.Precondition != nil {
		condition = w.Precondition(instance)
	}
	switch condition.Status {
	case condTrue:
		return w.Remediate(instance)
	case condFalse:
		return nil
	default:
		return condition
	}
}
