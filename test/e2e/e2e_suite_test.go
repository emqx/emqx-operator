/*
Copyright 2025-2026.

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
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/emqx/emqx-operator/test/util"
)

const (
	// projectImage is the name of the image which will be build and loaded
	// with the code source changes to be tested.
	projectImage = "emqx/emqx-operator:0.0.1"

	// namespace where the project is deployed in
	namespace = Namespace
)

// TestE2E runs the end-to-end (e2e) test suite for the project. These tests execute in an isolated,
// temporary environment to validate project changes with the the purposed to be used in CI jobs.
// The default setup requires Kind, builds/loads the Manager Docker image locally.
func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	// Set the default timeout and interval for async assertions
	SetDefaultEventuallyTimeout(time.Minute * 5)
	SetDefaultEventuallyPollingInterval(time.Second * 3)
	// Run tests
	RunSpecs(t, "e2e suite")
}

var _ = BeforeSuite(func() {
	By("generate manifests")
	Expect(util.Run("make", "manifests")).To(Succeed())

	By("build emqx-operator docker image")
	Expect(util.Run("make", "docker-build-coverage",
		fmt.Sprintf("OPERATOR_IMAGE=%s", projectImage),
	)).To(Succeed())

	By("load emqx-operator docker image into kind cluster")
	Expect(util.LoadImageToKindClusterWithName(projectImage)).To(Succeed())

	By("install Metrics Server")
	Expect(util.InstallMetricsServer()).To(Succeed())
})

var _ = AfterSuite(func() {
	util.UninstallMetricsServer()
})
