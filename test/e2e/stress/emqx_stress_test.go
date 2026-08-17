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

package stress

import (
	"flag"
	"fmt"
	"math/rand"
	"testing"
	"time"

	. "github.com/emqx/emqx-operator/test/e2e"
	. "github.com/emqx/emqx-operator/test/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Stress testing: generate a deterministic, pseudo-random sequence of CR
// changes from a known seed, then apply them in quick succession against a
// fixed EMQX cluster. The reconciliation loop is expected to survive the
// whole sequence and eventually drive the cluster back to a fully ready
// state matching the final desired replica counts.

// Mutation test parameters. Exposed as go-test flags so CI can shard runs
// across different seeds / sequence lengths without recompilation.
var (
	stressSteps  int
	stepInterval time.Duration
	emqxImage    string
)

func init() {
	flag.IntVar(&stressSteps, "stress-steps", 8,
		"Number of individual mutations to apply in sequence")
	flag.DurationVar(&stepInterval, "step-interval", 5*time.Second,
		"Delay between individual changes. Tight intervals are deliberate: "+
			"they stack changes on the reconciler while previous rollouts are "+
			"still in-flight and exercise transitional states.")
	// NOTE: `checkDSReplicationHealthy` needs EMQX 6.0.0 or newer.
	flag.StringVar(&emqxImage, "emqx-image", "emqx/emqx:6.1.1",
		"EMQX container image (with tag) to deploy for the mutation test")
}

// projectImage is the name of the image which will be build and loaded
// with the code source changes to be tested.
const projectImage = "emqx/emqx-operator:0.0.1"

type mutationKind int

const (
	scaleCores mutationKind = iota
	scaleReplicants
)

func (k mutationKind) String() string {
	switch k {
	case scaleCores:
		return "scale-cores"
	case scaleReplicants:
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
	{scaleCores, 2},
	{scaleCores, 3},
	{scaleCores, 4},
	{scaleCores, 5},
	{scaleReplicants, 1},
	{scaleReplicants, 2},
	{scaleReplicants, 3},
	{scaleReplicants, 4},
	{scaleReplicants, 5},
}

// planStress draws `steps` mutations uniformly at random from
// `mutationChoices`. An identical consecutive mutation (same kind and
// same value as the one just emitted) is resampled, so the plan never
// contains a truly no-op patch back-to-back.
func planStress(seed int64, steps int) []mutation {
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
		case scaleCores:
			s.cores = m.value
		case scaleReplicants:
			s.replicants = m.value
		}
	}
	return s
}

// mutationPatch returns the JSON merge-patch that applies `m` to the CR.
func mutationPatch(m mutation) string {
	switch m.kind {
	case scaleCores:
		return fmt.Sprintf(
			`{"spec":{"coreTemplate":{"spec":{"replicas":%d}}}}`,
			m.value,
		)
	case scaleReplicants:
		return fmt.Sprintf(
			`{"spec":{"replicantTemplate":{"spec":{"replicas":%d}}}}`,
			m.value,
		)
	}
	panic(fmt.Sprintf("unknown mutation kind: %v", m.kind))
}

func TestStress(t *testing.T) {
	RegisterFailHandler(Fail)
	// Set the default timeout and interval for async assertions
	SetDefaultEventuallyTimeout(time.Minute * 5)
	SetDefaultEventuallyPollingInterval(time.Second * 3)
	// Run tests
	RunSpecs(t, "Stress")
}

var _ = BeforeSuite(func() {
	By("generate manifests")
	Expect(Run("make", "manifests")).To(Succeed())

	By("build emqx-operator docker image")
	Expect(Run("make", "docker-build-coverage",
		fmt.Sprintf("OPERATOR_IMAGE=%s", projectImage),
	)).To(Succeed())

	By("load emqx-operator docker image into kind cluster")
	Expect(LoadImageToKindClusterWithName(projectImage)).To(Succeed())
})

var _ = Describe("EMQX Cluster / Stress Testing", Label("emqx", "stress"), Ordered, func() {

	const emqxCRBase = "test/e2e/files/resources/emqx.yaml"

	// The plan is captured up-front so the test reports can show the exact
	// sequence that was exercised, which is crucial for reproducing failures
	// discovered by a given seed.
	seed := GinkgoRandomSeed()
	plan := planStress(seed, stressSteps)
	initial := clusterState{cores: 2, replicants: 2}

	BeforeAll(func() {
		if len(plan) == 0 {
			Fail("planned stress sequence is empty")
		} else {
			By(fmt.Sprintf(
				"planned stress sequence (seed=%d, steps=%d, interval=%s):",
				seed, stressSteps, stepInterval,
			))
			for i, m := range plan {
				By(fmt.Sprintf("  [%d] %s=%d", i+1, m.kind, m.value))
			}
		}

		By("deploy EMQX Operator")
		Expect(Run("make", "deploy",
			fmt.Sprintf("OPERATOR_IMAGE=%s", projectImage),
			fmt.Sprintf("KUSTOMIZATION_FILE_PATH=%s", "test/e2e/files/manager"),
		)).To(Succeed())
		Expect(Kubectl("wait", "deployment", "emqx-operator-controller-manager",
			"--for", "condition=Available",
			"--namespace", Namespace,
			"--timeout", "1m",
		)).To(Succeed(), "Timed out waiting for emqx-operator deployment")
	})

	AfterAll(func() {
		By("undeploy EMQX Operator")
		_ = Run("make", "undeploy")
	})

	AfterEach(func() {
		if CurrentSpecReport().Failed() {
			DumpDiagnosticReport(Namespace, "emqx-mutation", CurrentSpecReport().StartTime)
		}
	})

	It("deploy cluster", func() {
		By("create EMQX cluster")
		emqxCR := SpecFromYAMLFile(emqxCRBase).
			WithImage(emqxImage).
			WithCores(initial.cores).
			WithReplicants(initial.replicants).
			WithDS().
			ToJSONDocument()
		Expect(KubectlStdin(emqxCR, "apply", "-f", "-")).To(Succeed())
		By("wait for EMQX cluster to be ready")
		Eventually(EMQXReady).Should(Succeed())
		Eventually(CoresStable).WithArguments(initial.cores).Should(Succeed())
		Eventually(ReplicantsStable).WithArguments(initial.replicants).Should(Succeed())
		Eventually(DSReplicationStable).WithArguments(initial.cores).Should(Succeed())
		Eventually(DSReplicationHealthy).Should(Succeed())
	})

	It("apply mutation sequence", func() {
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
				time.Sleep(stepInterval)
			}
		}

		final := finalState(plan, initial)

		By(fmt.Sprintf(
			"wait for EMQX cluster to settle at final state (cores=%d, replicants=%d)",
			final.cores, final.replicants,
		))
		Eventually(EMQXReady).WithArguments(sequenceStartedAt).Should(Succeed())
		Eventually(CoresStable).WithArguments(final.cores).Should(Succeed())
		Eventually(ReplicantsStable).WithArguments(final.replicants).Should(Succeed())
		Eventually(DSReplicationStable).WithArguments(final.cores).Should(Succeed())
		Eventually(DSReplicationHealthy).Should(Succeed())

		By("verify all node evacuations have completed")
		Eventually(KubectlOut).
			WithArguments("get", "emqx", "emqx", "-o", "jsonpath={.status.nodeEvacuations}").
			Should(BeEmpty(),
				"EMQX cluster still has pending node evacuations after settle")
	})

	It("delete cluster", func() {
		Expect(Kubectl("delete", "emqx", "emqx")).To(Succeed())
	})
})
