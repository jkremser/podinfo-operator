/*
Copyright 2021.

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

package test

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	http_helper "github.com/gruntwork-io/terratest/modules/http-helper"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/retry"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeployPodinfo(t *testing.T) {
	t.Parallel()

	kubeNamespacePath := "../examples/namespace.yaml"
	options := k8s.NewKubectlOptions("", "", "test-terratest")
	defer k8s.KubectlDelete(t, options, kubeNamespacePath)
	k8s.KubectlApply(t, options, kubeNamespacePath)

	podinfoPath := "../examples/podinfo.yaml"
	k8s.KubectlApply(t, options, podinfoPath)

	k8s.WaitUntilServiceAvailable(t, options, "podinfo-sample-fe", 10, 1*time.Second)
	service := k8s.GetService(t, options, "podinfo-sample-fe")
	url := fmt.Sprintf("http://%s", k8s.GetServiceEndpoint(t, options, service, 80))

	// this doesn't work because the replica set name is suffixed by some random chars (like pods)
	// backend_rs := k8s.GetReplicaSet(t, options, "podinfo-sample-be")
	// assert.Equal(t, 2, backend_rs.Spec.Replicas)
	backendLabels := metav1.ListOptions{
		LabelSelector: "app=podinfo-sample-be",
	}
	desiredBackendReplicas := 2
	k8s.WaitUntilNumPodsCreated(t, options, backendLabels, desiredBackendReplicas, 30, 2*time.Second)

	frontendLabels := metav1.ListOptions{
		LabelSelector: "app=podinfo-sample-fe",
	}
	desiredFrontendReplicas := 1
	k8s.WaitUntilNumPodsCreated(t, options, frontendLabels, desiredFrontendReplicas, 30, 2*time.Second)

	if _, ci := os.LookupEnv("GITHUB_ACTION"); !ci {
		http_helper.HttpGetWithRetryWithCustomValidation(
			t,
			url,
			nil,
			30,
			2*time.Second,
			func(statusCode int, body string) bool {
				isOk := statusCode == 200
				customMsg := strings.Contains(body, "Hello Podinfo")
				return isOk && customMsg
			},
		)
	}

	// update of custom resource
	changedPodinfoPath := "../examples/podinfo-changed.yaml"
	k8s.KubectlApply(t, options, changedPodinfoPath)

	k8s.WaitUntilServiceAvailable(t, options, "podinfo-sample-fe", 10, 2*time.Second)

	desiredBackendReplicas = 3
	// this takes more time, because the pods in the terminating state will still be listed (they still match the label)
	k8s.WaitUntilNumPodsCreated(t, options, backendLabels, desiredBackendReplicas, 50, 3*time.Second)

	desiredFrontendReplicas = 2
	k8s.WaitUntilNumPodsCreated(t, options, frontendLabels, desiredFrontendReplicas, 50, 3*time.Second)

	if _, ci := os.LookupEnv("GITHUB_ACTION"); !ci {
		http_helper.HttpGetWithRetryWithCustomValidation(
			t,
			url,
			nil,
			30,
			2*time.Second,
			func(statusCode int, body string) bool {
				isOk := statusCode == 200
				customMsg := strings.Contains(body, "Hello Terratest")
				return isOk && customMsg
			},
		)
	}

	// cr deleted
	k8s.KubectlDelete(t, options, changedPodinfoPath)
	waitUntilServiceIsGone(t, options, "podinfo-sample-fe", 40, 3*time.Second)
	_, err := k8s.GetReplicaSetE(t, options, "podinfo-sample-be")
	if err != nil {
		require.Error(t, err)
	}
}

func waitUntilServiceIsGone(t *testing.T, options *k8s.KubectlOptions, serviceName string, retries int, sleepBetweenRetries time.Duration) {
	statusMsg := fmt.Sprintf("Wait for service %s to be deleted.", serviceName)
	message := retry.DoWithRetry(
		t,
		statusMsg,
		retries,
		sleepBetweenRetries,
		func() (string, error) {
			_, err := k8s.GetServiceE(t, options, serviceName)
			if err == nil {
				return "Service is still there", errors.New("Service is still there")
			}

			return "Service is now gone", nil
		},
	)
	logger.Logf(t, message)
}
