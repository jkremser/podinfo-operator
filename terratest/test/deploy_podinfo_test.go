// +build kubeall kubernetes
// NOTE: See the notes in the other Kubernetes example tests for why this build tag is included.

package test

import (
	"fmt"
	"testing"
	"time"

	http_helper "github.com/gruntwork-io/terratest/modules/http-helper"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/retry"
	"github.com/gruntwork-io/terratest/modules/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeployPodinfo(t *testing.T) {
	t.Parallel()

	kubeNamespacePath := "../examples/namespace.yml"
	options := k8s.NewKubectlOptions("", "", "test-terratest")
	defer k8s.KubectlDelete(t, options, kubeNamespacePath)
	k8s.KubectlApply(t, options, kubeNamespacePath)

	podinfoPath := "../examples/podinfo.yml"
	k8s.KubectlApply(t, options, podinfoPath)

	k8s.WaitUntilServiceAvailable(t, options, "podinfo-sample-fe", 10, 1*time.Second)
	service := k8s.GetService(t, options, "podinfo-sample-fe")
	url := fmt.Sprintf("http://%s", k8s.GetServiceEndpoint(t, options, service, 5000))

	backend_rs := k8s.GetReplicaSet(t, options, "podinfo-sample-be")
	assert.Equal(t, 2, backend_rs.Replicas)
	// or k8s.WaitUntilNumPodsCreated w/ some label

	frontend_rs := k8s.GetReplicaSet(t, options, "podinfo-sample-fe")
	assert.Equal(t, 1, frontend_rs.Replicas)

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
	changedPodinfoPath := "../examples/podinfo-changed.yml"
	k8s.KubectlApply(t, options, changedPodinfoPath)

	k8s.WaitUntilServiceAvailable(t, options, "podinfo-sample-fe", 10, 1*time.Second)

	backend_rs = k8s.GetReplicaSet(t, options, "podinfo-sample-be")
	assert.Equal(t, 3, backend_rs.Replicas)

	frontend_rs = k8s.GetReplicaSet(t, options, "podinfo-sample-fe")
	assert.Equal(t, 2, frontend_rs.Replicas)

	http_helper.HttpGetWithRetry(t, url, nil, 200, "Hello Terratest", 30, 3*time.Second)

	// cr deleted
	k8s.KubectlDelete(t, options, changedPodinfoPath)
	waitUntilServiceIsGone(t, options, "podinfo-sample-fe", 10, 1*time.Second)
	_, err := k8s.GetReplicaSet(t, options, "podinfo-sample-be")
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
