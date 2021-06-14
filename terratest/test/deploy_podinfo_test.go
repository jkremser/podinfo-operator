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
	"fmt"
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
	url := fmt.Sprintf("http://%s", k8s.GetServiceEndpoint(t, options, service, 5000))

	// this doesn't work because the replica set name is suffixed by some random chars (like pods)
	// backend_rs := k8s.GetReplicaSet(t, options, "podinfo-sample-be")
	// assert.Equal(t, 2, backend_rs.Spec.Replicas)
	backendLabels := metav1.ListOptions{
		LabelSelector: "app=podinfo-sample-be",
	}
	desiredBackendReplicas := 2
	k8s.WaitUntilNumPodsCreated(t, options, backendLabels, desiredBackendReplicas, 30, 1*time.Second)

	frontendLabels := metav1.ListOptions{
		LabelSelector: "app=podinfo-sample-fe",
	}
	desiredFrontendReplicas := 1
	k8s.WaitUntilNumPodsCreated(t, options, frontendLabels, desiredFrontendReplicas, 30, 1*time.Second)

	http_helper.HttpGetWithRetry(t, url, nil, 200, "Hello Podinfo", 30, 3*time.Second)
	// http_helper.HttpGetWithRetryWithCustomValidation(
	//     t,
	//     url,
	//     30,
	//     3*time.Second,
	//     func(statusCode int, body string) bool {
	//         isOk := statusCode == 200
	//         customMsg := strings.Contains(body, "Hello Podinfo")
	//         return isOk && customMsg
	//     },
	// )

	// update of custom resource
	changedPodinfoPath := "../examples/podinfo-changed.yaml"
	k8s.KubectlApply(t, options, changedPodinfoPath)

	k8s.WaitUntilServiceAvailable(t, options, "podinfo-sample-fe", 10, 1*time.Second)

	desiredBackendReplicas = 3
	k8s.WaitUntilNumPodsCreated(t, options, backendLabels, desiredBackendReplicas, 30, 1*time.Second)

	desiredFrontendReplicas = 2
	k8s.WaitUntilNumPodsCreated(t, options, frontendLabels, desiredFrontendReplicas, 30, 1*time.Second)

	http_helper.HttpGetWithRetry(t, url, nil, 200, "Hello Terratest", 30, 3*time.Second)

	// cr deleted
	k8s.KubectlDelete(t, options, changedPodinfoPath)
	waitUntilServiceIsGone(t, options, "podinfo-sample-fe", 10, 1*time.Second)
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
			service, err := k8s.GetServiceE(t, options, serviceName)
			if err != nil {
				return "", err
			}

			if k8s.IsServiceAvailable(service) {
				return "", k8s.NewServiceNotAvailableError(service)
			}
			return "Service is now gone", nil
		},
	)
	logger.Logf(t, message)
}
