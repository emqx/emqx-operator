package upgrade

import (
	"testing"
	"time"

	"github.com/onsi/gomega"

	"github.com/stretchr/testify/require"
)

func TestWorkaroundResolution(t *testing.T) {
	recipe := upgradeWorkaround{Description: "synthetic test recipe"}
	wildcard := upgradeWorkaround{Description: "minor-version recipe"}
	book := playbook{
		{Initial: "1.0.0", Upgrade: "2.0.0"}: recipe,
		{Initial: "1.1.x", Upgrade: "2.0.0"}: wildcard,
		{Initial: "1.1.2", Upgrade: "2.0.0"}: recipe,
	}
	for _, tc := range []struct {
		name, initial, upgrade string
		want                   string
	}{
		{"exact tags", "emqx/emqx:1.0.0", "emqx/emqx:2.0.0", recipe.Description},
		{"registry port", "localhost:5000/emqx/emqx:1.0.0", "localhost:5000/emqx/emqx:2.0.0", recipe.Description},
		{"minor wildcard", "emqx/emqx:1.1.0", "emqx/emqx:2.0.0", wildcard.Description},
		{"minor wildcard later patch", "emqx/emqx:1.1.99", "emqx/emqx:2.0.0", wildcard.Description},
		{"minor wildcard prerelease", "emqx/emqx:1.1.3-rc.1", "emqx/emqx:2.0.0", wildcard.Description},
		{"minor wildcard metadata", "emqx/emqx:1.1.3+custom", "emqx/emqx:2.0.0", wildcard.Description},
		{"exact overrides wildcard", "emqx/emqx:1.1.2", "emqx/emqx:2.0.0", recipe.Description},
		{"different minor", "emqx/emqx:1.2.0", "emqx/emqx:2.0.0", ""},
		{"wildcard requires exact target", "emqx/emqx:1.1.0", "emqx/emqx:2.0.1", ""},
		{"reverse path", "emqx/emqx:2.0.0", "emqx/emqx:1.0.0", ""},
		{"different patch", "emqx/emqx:1.0.1", "emqx/emqx:2.0.0", ""},
		{"prerelease", "emqx/emqx:1.0.0", "emqx/emqx:2.0.0-rc.1", ""},
		{"missing tag", "emqx/emqx", "emqx/emqx:2.0.0", ""},
		{"registry port without tag", "localhost:5000/emqx/emqx", "emqx/emqx:2.0.0", ""},
		{"empty tag", "emqx/emqx:1.0.0", "emqx/emqx:", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := book.resolve(tc.initial, tc.upgrade)
			require.Equal(t, tc.want != "", ok)
			if ok {
				require.Equal(t, tc.want, got.Description)
			}
		})
	}
}

func TestWorkaroundPreconditionFalse(t *testing.T) {
	recipe := upgradeWorkaround{
		Precondition: func(instance string) precondition {
			require.Equal(t, "emqx", instance)
			return precondition{Status: condFalse}
		},
		Remediate: func(string) error {
			t.Fatal("false precondition must skip remediation")
			return nil
		},
	}
	require.NoError(t, recipe.apply("emqx"))
}

func TestWorkaroundPreconditionTrue(t *testing.T) {
	calls := 0
	recipe := upgradeWorkaround{
		Precondition: func(instance string) precondition {
			require.Equal(t, "emqx", instance)
			return precondition{Status: condTrue}
		},
		Remediate: func(instance string) error {
			require.Equal(t, "emqx", instance)
			calls++
			return nil
		},
	}
	require.NoError(t, recipe.apply("emqx"))
	require.Equal(t, 1, calls)
}

func TestWorkaroundPreconditionUnknown(t *testing.T) {
	condition := precondition{Status: condUnknown, Message: "cannot read status"}
	recipe := upgradeWorkaround{
		Precondition: func(instance string) precondition {
			require.Equal(t, "emqx", instance)
			return condition
		},
		Remediate: func(string) error {
			t.Fatal("unknown precondition must defer remediation")
			return nil
		},
	}
	err := recipe.apply("emqx")
	var pending precondition
	require.ErrorAs(t, err, &pending)
	require.Equal(t, condition, pending)
	require.ErrorContains(t, err, condition.Message)
}

func TestWorkaroundPreconditionPolling(t *testing.T) {
	condition := precondition{Status: condUnknown, Message: "cannot list core pods"}
	calls := 0
	recipe := upgradeWorkaround{
		Precondition: func(string) precondition { return condition },
		Remediate:    func(string) error { calls++; return nil },
	}

	var failure string
	g := gomega.NewGomega(func(message string, _ ...int) { failure = message })
	g.Eventually(recipe.apply, 20*time.Millisecond, time.Millisecond).
		WithArguments("emqx").Should(gomega.Succeed())
	require.Contains(t, failure, condition.Message)
	require.Zero(t, calls)

	condition.Status = condTrue
	gomega.NewWithT(t).Eventually(recipe.apply, time.Second, time.Millisecond).
		WithArguments("emqx").Should(gomega.Succeed())
	require.Equal(t, 1, calls)
}
