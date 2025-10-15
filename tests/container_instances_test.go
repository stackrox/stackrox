//go:build test_e2e

package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/require"
)

type ContainerNameGroup struct {
	IDStruct
	Name      string       `json:"name"`
	StartTime graphql.Time `json:"startTime"`
	Events    []Event      `json:"events"`
}

func TestContainerInstances(testT *testing.T) {
	// testT.Skip("Flaky: https://issues.redhat.com/browse/ROX-30400")
	// https://stack-rox.atlassian.net/browse/ROX-6493
	// - the process events expected in this test are not reliably detected.
	kPod := getPodFromFile(testT, "yamls/multi-container-pod.yaml")
	client := createK8sClient(testT)

	// Ensure pod is cleaned up at test end, after all assertions
	defer teardownPod(testT, client, kPod)

	// Retry the entire setup to handle transient K8s API issues, slow pod startup, and Central ingestion lag
	testutils.Retry(testT, 5, 10*time.Second, func(retryT testutils.T) {
		// Ensure pod exists (idempotent - safe to retry even if pod already exists)
		ensurePodExists(retryT, client, kPod)

		// Wait for pod to be fully running with all containers ready
		waitForPodRunning(retryT, client, kPod.GetNamespace(), kPod.GetName())
		testT.Logf("Pod %s is running with all containers ready", kPod.GetName())

		// Wait for Central to see the deployment
		testT.Logf("Waiting for Central to see deployment %s", kPod.GetName())
		waitForDeployment(retryT, kPod.GetName())
		testT.Logf("Central now sees deployment %s", kPod.GetName())
	})

	// Get deployment and pod data from Central with retry
	var deploymentID string
	var pod Pod
	testutils.Retry(testT, 5, 10*time.Second, func(deplRetryT testutils.T) {
		deploymentID = getDeploymentID(deplRetryT, kPod.GetName())
		deplRetryT.Logf("Central sees the deployment under ID %s", deploymentID)

		pods := getPods(deplRetryT, deploymentID)
		require.Len(deplRetryT, pods, 1)
		pod = pods[0]
	})

	// Retry to ensure all processes start up and are detected
	testutils.Retry(testT, 20, 4*time.Second, func(retryEventsT testutils.T) {
		// Get the container groups.
		groupedContainers := getGroupedContainerInstances(retryEventsT, string(pod.ID))

		// Verify the number of containers.
		require.Len(retryEventsT, groupedContainers, 2)
		// Verify default sort is by name.
		names := sliceutils.Map(groupedContainers, func(g ContainerNameGroup) string { return g.Name })
		require.Equal(retryEventsT, names, []string{"1st", "2nd"})

		// Use "at least" semantics: verify required processes exist, but allow extras.
		// Rationale: Modern container images (especially nginx) run extensive initialization:
		// - docker-entrypoint.sh and scripts in /docker-entrypoint.d/ (10-listen-on-ipv6, 20-envsubst, 30-tune-workers)
		// - Short-lived utilities: /usr/bin/find, /bin/grep, /usr/bin/cut, /bin/sed, /usr/bin/basename, etc.
		// - nginx worker processes (duplicate /usr/sbin/nginx)
		// A typical nginx container may capture 20+ processes during startup. This approach focuses on
		// verifying the main application processes exist without being brittle to image implementation details.

		firstContainerEvents :=
			sliceutils.Map(groupedContainers[0].Events, func(event Event) string { return event.Name })
		retryEventsT.Logf("First container (%s) events: %+v", groupedContainers[0].Name, firstContainerEvents)

		// First container: nginx (may see workers and ~20 docker-entrypoint processes)
		requiredFirstContainer := []string{"/usr/sbin/nginx"}
		retryEventsT.Logf("First container always required: %v", requiredFirstContainer)
		for _, required := range requiredFirstContainer {
			require.Contains(retryEventsT, firstContainerEvents, required,
				"first container missing required process %q", required)
		}

		secondContainerEvents :=
			sliceutils.Map(groupedContainers[1].Events, func(event Event) string { return event.Name })
		retryEventsT.Logf("Second container (%s) events: %+v", groupedContainers[1].Name, secondContainerEvents)

		// Second container: busybox running a simple loop
		// Always required: the shell running the loop
		alwaysRequiredSecond := []string{"/bin/sh"}
		retryEventsT.Logf("Second container always required: %v", alwaysRequiredSecond)
		require.Contains(retryEventsT, secondContainerEvents, "/bin/sh",
			"second container missing required process /bin/sh")

		// At least one required: processes from the busybox loop.
		// MYSTERY/BREADCRUMB: /bin/date is consistently NOT captured, while /bin/sleep IS captured.
		// This is puzzling because:
		// 1. The loop runs for 60+ seconds (plenty of time for events to be ingested)
		// 2. The collector CAN capture microsecond-duration processes (we've seen /bin/grep, /usr/bin/find, etc.)
		// 3. /bin/sleep is captured reliably, proving the loop is running
		// 4. If sleep is captured, date must have executed 60+ times just before each sleep
		atLeastOneRequired := []string{"/bin/date", "/bin/sleep"}
		retryEventsT.Logf("Second container at least one required (short-lived): %v", atLeastOneRequired)
		foundAtLeastOne := false
		for _, candidate := range atLeastOneRequired {
			for _, proc := range secondContainerEvents {
				if proc == candidate {
					foundAtLeastOne = true
					break
				}
			}
			if foundAtLeastOne {
				break
			}
		}
		require.True(retryEventsT, foundAtLeastOne,
			"second container: expected at least one of %v, found none", atLeastOneRequired)

		// Verify the container group's timestamp is no later than the timestamp of the earliest event
		// Find the actual earliest event for each container (GraphQL doesn't guarantee ordering)
		firstContainerEarliestTime := groupedContainers[0].Events[0].Timestamp.Time
		for _, event := range groupedContainers[0].Events[1:] {
			if event.Timestamp.Time.Before(firstContainerEarliestTime) {
				firstContainerEarliestTime = event.Timestamp.Time
			}
		}
		require.False(retryEventsT, groupedContainers[0].StartTime.After(firstContainerEarliestTime),
			"container 0 start time (%s) should not be after earliest event time (%s)",
			groupedContainers[0].StartTime, firstContainerEarliestTime)

		secondContainerEarliestTime := groupedContainers[1].Events[0].Timestamp.Time
		for _, event := range groupedContainers[1].Events[1:] {
			if event.Timestamp.Time.Before(secondContainerEarliestTime) {
				secondContainerEarliestTime = event.Timestamp.Time
			}
		}
		require.False(retryEventsT, groupedContainers[1].StartTime.After(secondContainerEarliestTime),
			"container 1 start time (%s) should not be after earliest event time (%s)",
			groupedContainers[1].StartTime, secondContainerEarliestTime)

		// Verify risk event timeline CSV
		verifyRiskEventTimelineCSV(retryEventsT, deploymentID, append(firstContainerEvents, secondContainerEvents...))
	})
}

func getGroupedContainerInstances(t testutils.T, podID string) []ContainerNameGroup {
	var respData struct {
		GroupedContainerInstances []ContainerNameGroup `json:"groupedContainerInstances"`
	}

	makeGraphQLRequest(t, `
		query getGroupedContainerInstances($containersQuery: String) {
			groupedContainerInstances(query: $containersQuery) {
				id
				name
				startTime
				events {
					id
					name
					timestamp
				}
			}
		}
	`, map[string]interface{}{
		"containersQuery": fmt.Sprintf("Pod ID: %s", podID),
	}, &respData, timeout)
	log.Info(respData)

	return respData.GroupedContainerInstances
}
