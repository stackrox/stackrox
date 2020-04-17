package tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/require"
)

// Note: Update here if yamls/multi-container-pod.yaml is updated
const podName = "end-to-end-api-test-pod-multi-container"

type ContainerNameGroup struct {
	IDStruct
	Name   string  `json:"name"`
	Events []Event `json:"events"`
}

func TestContainerInstances(t *testing.T) {
	// Set up testing environment
	setupDeploymentFromFile(t, podName, "yamls/multi-container-pod.yaml")
	defer teardownDeploymentFromFile(t, podName, "yamls/multi-container-pod.yaml")

	// Get the test pod.
	podID := getPodID(t, podName)

	// Retry to ensure all processes start up.
	testutils.Retry(t, 20, 3*time.Second, func(t testutils.T) {
		// Get the container groups.
		groupedContainers := getGroupedContainerInstances(t, podID)

		// Verify the number of containers.
		require.Len(t, groupedContainers, 2)
		// Verify default sort is by name.
		names := sliceutils.Map(groupedContainers, func(g ContainerNameGroup) string { return g.Name })
		require.Equal(t, names, []string{"1st", "2nd"})
		// Verify the events.
		// Expecting 1 process: nginx
		require.Len(t, groupedContainers[0].Events, 1)
		events := sliceutils.Map(groupedContainers[0].Events, func(event Event) string { return event.Name })
		require.ElementsMatch(t, events, []string{"nginx"})
		// Expecting 3 processes: sh, date, sleep
		require.Len(t, groupedContainers[1].Events, 3)
		events = sliceutils.Map(groupedContainers[1].Events, func(event Event) string { return event.Name })
		require.ElementsMatch(t, events, []string{"sh", "date", "sleep"})
	})
}

func getPodID(t *testing.T, podName string) string {
	var respData struct {
		Pods []IDStruct `json:"pods"`
	}

	makeGraphQLRequest(t, `
		query pods($query: String) {
			pods(query: $query) {
				id
			}
		}
	`, map[string]interface{}{
		"query": fmt.Sprintf("Pod Name: %s", podName),
	}, &respData, timeout)
	log.Info(respData)
	require.Len(t, respData.Pods, 1)

	return string(respData.Pods[0].ID)
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
				events {
					id
					name
				}
			}
		}
	`, map[string]interface{}{
		"containersQuery": fmt.Sprintf("Pod ID: %s", podID),
	}, &respData, timeout)
	log.Info(respData)

	return respData.GroupedContainerInstances
}
