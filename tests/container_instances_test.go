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
	// https://stack-rox.atlassian.net/browse/ROX-6493
	// - the process events expected in this test are not reliably detected.
	testutils.Retry(testT, 3, 5*time.Second, func(retryT testutils.T) {
		// Set up testing environment
		defer teardownDeploymentFromFile(retryT, deploymentName, "yamls/multi-container-pod.yaml")
		setupDeploymentFromFile(retryT, deploymentName, "yamls/multi-container-pod.yaml")

		// Get the test pod.
		deploymentID := getDeploymentID(retryT, deploymentName)
		pods := getPods(retryT, deploymentID)
		require.Len(retryT, pods, 1)
		pod := pods[0]

		// Retry to ensure all processes start up.
		testutils.Retry(retryT, 20, 4*time.Second, func(retryEventsT testutils.T) {
			// Get the container groups.
			groupedContainers := getGroupedContainerInstances(retryEventsT, string(pod.ID))

			// Verify the number of containers.
			require.Len(retryEventsT, groupedContainers, 2)
			// Verify default sort is by name.
			names := sliceutils.Map(groupedContainers, func(g ContainerNameGroup) string { return g.Name })
			require.Equal(retryEventsT, names, []string{"1st", "2nd"})
			// Verify the events.
			// Expecting 1 process: nginx
			require.Len(retryEventsT, groupedContainers[0].Events, 1)
			firstContainerEvents :=
				sliceutils.Map(groupedContainers[0].Events, func(event Event) string { return event.Name })
			require.ElementsMatch(retryEventsT, firstContainerEvents, []string{"/usr/sbin/nginx"})
			// Expecting 3 processes: sh, date, sleep
			require.Len(retryEventsT, groupedContainers[1].Events, 3)
			secondContainerEvents :=
				sliceutils.Map(groupedContainers[1].Events, func(event Event) string { return event.Name })
			require.ElementsMatch(retryEventsT, secondContainerEvents, []string{"/bin/sh", "/bin/date", "/bin/sleep"})

			// Verify the container group's timestamp is no later than the timestamp of the first event
			require.False(retryEventsT, groupedContainers[0].StartTime.After(groupedContainers[0].Events[0].Timestamp.Time))
			require.False(retryEventsT, groupedContainers[1].StartTime.After(groupedContainers[1].Events[0].Timestamp.Time))

			// Number of events expected should be the aggregate of the above

			verifyRiskEventTimelineCSV(retryEventsT, deploymentID, append(firstContainerEvents, secondContainerEvents...))
		})
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
