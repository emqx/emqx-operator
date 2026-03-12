/*
Copyright 2025.

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

package helm

import (
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/emqx/emqx-operator/test/util"
)

// projectImage is the operator image which should already be built and available.
var (
	operatorImageRepo = "emqx/emqx-operator"
	operatorImageTag  = "0.0.1-helm"
	operatorImage     = operatorImageRepo + ":" + operatorImageTag
)

func TestHelmE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	_, _ = fmt.Fprintf(GinkgoWriter, "Starting Helm e2e test suite\n")
	RunSpecs(t, "Helm e2e suite")
}

var _ = BeforeSuite(func() {
	SetDefaultEventuallyTimeout(3 * time.Minute)
	SetDefaultEventuallyPollingInterval(3 * time.Second)

	By("add emqx Helm repository")
	Expect(util.Run("helm", "repo", "add", "emqx", "https://repos.emqx.io/charts")).To(Succeed())
	Expect(util.Run("helm", "repo", "update")).To(Succeed())

	By("generate Helm chart files")
	Expect(util.Run("make", "helm")).To(Succeed())

	By("build emqx-operator docker image")
	Expect(util.Run("make", "docker-build",
		fmt.Sprintf("OPERATOR_IMAGE=%s", operatorImage),
	)).To(Succeed())

	By("load operator image into Kind cluster")
	Expect(util.LoadImageToKindClusterWithName(operatorImage)).To(Succeed())
})
