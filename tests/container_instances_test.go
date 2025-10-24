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
	skipIfNoCollection(testT)

	// Wait for Sensor to be healthy to ensure the event collection pipeline is ready
	// after any previous tests that may have restarted Sensor.
	waitForSensorHealthy(testT)

	_, deploymentID, pod, cleanup := setupMultiContainerPodTest(testT)
	defer cleanup()

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
		require.Subsetf(retryEventsT, firstContainerEvents, requiredFirstContainer,
			"First container: required processes: %v not found in events: %v", requiredFirstContainer, firstContainerEvents)

		secondContainerEvents :=
			sliceutils.Map(groupedContainers[1].Events, func(event Event) string { return event.Name })
		retryEventsT.Logf("Second container (%s) events: %+v", groupedContainers[1].Name, secondContainerEvents)

		// Second container: ubuntu running a loop with date and sleep
		// TODO(ROX-31331): Collector cannot reliably detect /bin/sh /bin/date or /bin/sleep in ubuntu image,
		// thus not including it in the required processes.
		// If this flakes again, see ROX-31331 and follow-up on the discussion in the ticket.
		requiredSecondContainer := []string{"/bin/sh"}
		require.Subsetf(retryEventsT, secondContainerEvents, requiredSecondContainer,
			"Second container: required processes: %v not found in events: %v", requiredSecondContainer, secondContainerEvents)

		// Verify container start times are not after their earliest events
		verifyStartTimeBeforeEvents(retryEventsT, groupedContainers[0].StartTime, groupedContainers[0].Events, "Container 0")
		verifyStartTimeBeforeEvents(retryEventsT, groupedContainers[1].StartTime, groupedContainers[1].Events, "Container 1")

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
