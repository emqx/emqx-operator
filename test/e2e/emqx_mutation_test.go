/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"flag"
	"fmt"
	"math/rand"
	"time"

	. "github.com/emqx/emqx-operator/test/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Mutation testing: generate a deterministic, pseudo-random sequence of CR
// changes from a known seed, then apply them in quick succession against a
// fixed EMQX cluster. The reconciliation loop is expected to survive the
// whole sequence and eventually drive the cluster back to a fully ready
// state matching the final desired replica counts.
//
// The cluster always runs with Durable Storage enabled:
// core scaling on a DS-enabled cluster exercises shard-replication bookkeeping
// on top of plain StatefulSet rollout, which is where most subtle reconciler
// regressions tend to surface.

// Mutation test parameters. Exposed as go-test flags so CI can shard runs
// across different seeds / sequence lengths without recompilation.
var (
	mutationSeed     int64
	mutationSteps    int
	mutationInterval time.Duration
	mutationImage    string
)

func init() {
	flag.Int64Var(&mutationSeed, "mutation-seed", 1,
		"Seed for the pseudo-random mutation sequence")
	flag.IntVar(&mutationSteps, "mutation-steps", 8,
		"Number of individual mutations to apply in sequence")
	flag.DurationVar(&mutationInterval, "mutation-interval", 5*time.Second,
		"Delay between individual mutations. Tight intervals are deliberate: "+
			"they stack changes on the reconciler while previous rollouts are "+
			"still in-flight and exercise transitional states.")
	// NOTE: `checkDSReplicationHealthy` needs EMQX 6.0.0 or newer.
	flag.StringVar(&mutationImage, "mutation-image", "emqx/emqx:6.1.1",
		"EMQX container image (with tag) to deploy for the mutation test")
}

type mutationKind int

const (
	mutateScaleCores mutationKind = iota
	mutateScaleReplicants
)

func (k mutationKind) String() string {
	switch k {
	case mutateScaleCores:
		return "scale-cores"
	case mutateScaleReplicants:
		return "scale-replicants"
	}
	return "unknown"
}

// mutation is one planned CR change: a kind (which dimension to scale) and
// an absolute target replica count.
type mutation struct {
	kind  mutationKind
	value int
}

// clusterState is the minimal projection of the CR relevant to mutations.
type clusterState struct {
	cores      int
	replicants int
}

// mutationChoices is the static set of mutations the planner draws from.
// Each entry is one concrete (kind, value) pair; to add a new mutation type,
// just append more entries to this slice.
var mutationChoices = []mutation{
	{mutateScaleCores, 2},
	{mutateScaleCores, 3},
	{mutateScaleCores, 4},
	{mutateScaleCores, 5},
	{mutateScaleReplicants, 1},
	{mutateScaleReplicants, 2},
	{mutateScaleReplicants, 3},
	{mutateScaleReplicants, 4},
	{mutateScaleReplicants, 5},
}

// planMutations draws `steps` mutations uniformly at random from
// `mutationChoices`. An identical consecutive mutation (same kind and
// same value as the one just emitted) is resampled, so the plan never
// contains a truly no-op patch back-to-back; non-adjacent duplicates are
// kept as-is because they still exercise the reconciler against a
// different intermediate state.
func planMutations(seed int64, steps int) []mutation {
	rnd := rand.New(rand.NewSource(seed))
	out := make([]mutation, 0, steps)
	for len(out) < steps {
		m := mutationChoices[rnd.Intn(len(mutationChoices))]
		if len(out) > 0 && out[len(out)-1] == m {
			continue
		}
		out = append(out, m)
	}
	return out
}

// finalState walks the plan to compute the cluster's desired state after
// the last mutation is applied, starting from `initial`.
func finalState(plan []mutation, initial clusterState) clusterState {
	s := initial
	for _, m := range plan {
		switch m.kind {
		case mutateScaleCores:
			s.cores = m.value
		case mutateScaleReplicants:
			s.replicants = m.value
		}
	}
	return s
}

// mutationPatch returns the JSON merge-patch that applies `m` to the CR.
func mutationPatch(m mutation) string {
	switch m.kind {
	case mutateScaleCores:
		return fmt.Sprintf(
			`{"spec":{"coreTemplate":{"spec":{"replicas":%d}}}}`,
			m.value,
		)
	case mutateScaleReplicants:
		return fmt.Sprintf(
			`{"spec":{"replicantTemplate":{"spec":{"replicas":%d}}}}`,
			m.value,
		)
	}
	panic(fmt.Sprintf("unknown mutation kind: %v", m.kind))
}

//nolint:errcheck
var _ = Describe("EMQX Cluster / Mutation Testing", Label("emqx", "mutation"), Ordered, func() {

	const emqxCRBasic = "test/e2e/files/resources/emqx.yaml"

	initial := clusterState{cores: 2, replicants: 2}

	BeforeAll(func() {
		By("create manager namespace")
		Expect(Kubectl("create", "ns", namespace)).To(Succeed())

		By("install CRDs")
		Expect(Run("make", "install")).To(Succeed())

		By("deploy emqx-operator")
		Expect(Run("make", "deploy",
			fmt.Sprintf("OPERATOR_IMAGE=%s", projectImage),
			fmt.Sprintf("KUSTOMIZATION_FILE_PATH=%s", "test/e2e/files/manager"),
		)).To(Succeed())
		Expect(Kubectl("wait", "deployment", "emqx-operator-controller-manager",
			"--for", "condition=Available",
			"--namespace", namespace,
			"--timeout", "1m",
		)).To(Succeed(), "Timed out waiting for emqx-operator deployment")
	})

	AfterAll(func() {
		By("undeploy emqx-operator")
		_ = Run("make", "undeploy")

		By("uninstall CRDs")
		_ = Run("make", "uninstall")

		By("delete manager namespace")
		_ = Kubectl("delete", "ns", namespace)
	})

	AfterEach(func() {
		if CurrentSpecReport().Failed() {
			DumpDiagnosticReport(namespace, "emqx-mutation", CurrentSpecReport().StartTime)
		}
	})

	// The plan is captured up-front so the test reports can show the exact
	// sequence that was exercised, which is crucial for reproducing failures
	// discovered by a given seed.
	plan := planMutations(mutationSeed, mutationSteps)

	It("deploy cluster", func() {
		By(fmt.Sprintf(
			"planned mutation sequence (seed=%d, steps=%d, interval=%s, image=%s):",
			mutationSeed, mutationSteps, mutationInterval, mutationImage,
		))
		for i, m := range plan {
			By(fmt.Sprintf("  [%d] %s=%d", i+1, m.kind, m.value))
		}

		By("create EMQX cluster")
		emqxCR := PatchDocument(
			FromYAMLFile(emqxCRBasic),
			withImage(mutationImage),
			withCores(initial.cores),
			withReplicants(initial.replicants),
			withConfig(configDS()),
		)
		Expect(KubectlStdin(emqxCR, "apply", "-f", "-")).To(Succeed())
		By("wait for EMQX cluster to be ready")
		Eventually(checkEMQXReady).Should(Succeed())
		Eventually(checkEMQXStatus).WithArguments(initial.cores).Should(Succeed())
		Eventually(checkReplicantStatus).WithArguments(initial.replicants).Should(Succeed())
		Eventually(checkDSReplicationStatus).WithArguments(initial.cores).Should(Succeed())
		Eventually(checkDSReplicationHealthy).Should(Succeed())
	})

	It("apply mutation sequence", func() {
		if len(plan) == 0 {
			Skip("mutation plan is empty")
		}

		// MQTTX holds ~25 long-lived MQTT connections against the cluster
		// service. That load is what turns a plain core scale-down into a
		// real node evacuation (the operator drains connections from the
		// doomed node via EMQX's load-rebalance API before deleting the
		// pod), so keeping a workload attached throughout the mutation
		// sequence is what actually exercises `.status.nodeEvacuations`.
		By("create MQTT workload")
		Expect(Kubectl("apply", "-f", "test/e2e/files/resources/mqttx.yaml")).To(Succeed())
		defer Kubectl("delete", "-f", "test/e2e/files/resources/mqttx.yaml") //nolint:errcheck
		Expect(Kubectl("wait", "pod",
			"--selector=app=mqttx",
			"--for=condition=Ready",
			"--timeout=1m",
		)).To(Succeed(), "Timed out waiting for MQTTX to be ready")

		sequenceStartedAt := metav1.Now()
		for i, m := range plan {
			By(fmt.Sprintf("[%d/%d] apply %s=%d", i+1, len(plan), m.kind, m.value))
			Expect(Kubectl("patch", "emqx", "emqx",
				"--type", "merge",
				"--patch", mutationPatch(m),
			)).To(Succeed(), "Failed to apply mutation %d (%s=%d)", i+1, m.kind, m.value)
			// Deliberately do _not_ wait for readiness between mutations: the
			// point of the test is to stack changes while reconciliation is
			// still in-flight. A fixed delay gives the controller a chance to
			// observe each change but keeps the queue pressured.
			if i < len(plan)-1 {
				time.Sleep(mutationInterval)
			}
		}

		final := finalState(plan, initial)

		By(fmt.Sprintf(
			"wait for EMQX cluster to settle at final state (cores=%d, replicants=%d)",
			final.cores, final.replicants,
		))
		Eventually(checkEMQXReady).
			WithArguments(sequenceStartedAt).
			Should(Succeed())
		Eventually(checkEMQXStatus).
			WithArguments(final.cores).
			Should(Succeed())
		Eventually(checkReplicantStatus).
			WithArguments(final.replicants).
			Should(Succeed())
		Eventually(checkDSReplicationHealthy).
			Should(Succeed())
		Eventually(checkDSReplicationStatus).
			WithArguments(final.cores).
			Should(Succeed())

		By("verify all node evacuations have completed")
		Eventually(KubectlOut).
			WithArguments("get", "emqx", "emqx", "-o", "jsonpath={.status.nodeEvacuations}").
			Should(BeEmpty(),
				"EMQX cluster still has pending node evacuations after settle")
	})

	It("delete cluster", func() {
		Expect(Kubectl("delete", "emqx", "emqx")).To(Succeed())
		Expect(Kubectl("get", "emqx", "emqx")).To(HaveOccurred(), "EMQX cluster still exists")
	})
})
