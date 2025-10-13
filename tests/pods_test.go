//go:build test_e2e || test_compatibility

package tests

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/graphql/resolvers/inputtypes"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
)

type IDStruct struct {
	ID graphql.ID `json:"id"`
}

type Pod struct {
	IDStruct
	Name           string       `json:"name"`
	ContainerCount int32        `json:"containerCount"`
	Started        graphql.Time `json:"started"`
	Events         []Event      `json:"events"`
}

type Event struct {
	IDStruct
	Name      string       `json:"name"`
	Timestamp graphql.Time `json:"timestamp"`
}

func TestPod(testT *testing.T) {
	// https://stack-rox.atlassian.net/browse/ROX-6631
	// - the process events expected in this test are not reliably detected.

	kPod := getPodFromFile(testT, "yamls/multi-container-pod.yaml")
	client := createK8sClient(testT)

	var k8sPod *coreV1.Pod
	// Ensure pod is cleaned up at test end, after all assertions
	defer teardownPod(testT, client, kPod)

	// Retry the entire setup to handle transient K8s API issues, slow pod startup, and Central ingestion lag
	testutils.Retry(testT, 5, 10*time.Second, func(retryT testutils.T) {
		// Ensure pod exists (idempotent - safe to retry even if pod already exists)
		ensurePodExists(retryT, client, kPod)
		// Wait for pod to be fully running with all containers ready
		k8sPod = waitForPodRunning(retryT, client, kPod.GetNamespace(), kPod.GetName())
		testT.Logf("Pod %s is running with all containers ready", kPod.GetName())

		// Now wait for Central to see the deployment
		// This can take time as Sensor needs to detect the pod and report it to Central
		testT.Logf("Waiting for Central to see deployment %s", kPod.GetName())
		waitForDeployment(retryT, kPod.GetName())
		testT.Logf("Central now sees deployment %s", kPod.GetName())
	})

	deploymentID := ""
	var pod Pod
	testutils.Retry(testT, 5, 10*time.Second, func(deplRetryT testutils.T) {
		// Get the test deployment.
		deploymentID = getDeploymentID(deplRetryT, kPod.GetName())
		deplRetryT.Logf("Central sees the deployment under ID %s", deploymentID)

		podCount := getPodCount(deplRetryT, deploymentID)
		deplRetryT.Logf("Pod count: %d", podCount)
		require.Equal(deplRetryT, 1, podCount)

		// Get the test pod.
		pods := getPods(deplRetryT, deploymentID)
		deplRetryT.Logf("Num pods: %d", len(pods))
		require.Len(deplRetryT, pods, 1)
		pod = pods[0]

		deplRetryT.Logf("Pod: %+v", pod)

		// Verify the container count.
		require.Equal(deplRetryT, int32(2), pod.ContainerCount)

		// Verify Pod start time is the creation time.
		deplRetryT.Logf("Creation timestamps comparison: %s vs %s", k8sPod.GetCreationTimestamp().Time.UTC(), pod.Started.UTC())
		require.Equal(deplRetryT, k8sPod.GetCreationTimestamp().Time.UTC(), pod.Started.UTC())
	})

	if os.Getenv("COLLECTION_METHOD") == "NO_COLLECTION" {
		testT.Logf("Skipping parts of TestPod that relate to events because env var \"COLLECTION_METHOD\" is " +
			"set to \"NO_COLLECTION\"")
		return
	}
	testutils.Retry(testT, 30, 5*time.Second, func(retryEventsT testutils.T) {
		events := getEvents(retryEventsT, pod)
		retryEventsT.Logf("Found %d events (expected 4): %+v", len(events), events)

		// Verify we have all 4 expected events
		require.Len(retryEventsT, events, 4, "expected 4 process events")

		// Expecting processes: nginx, sh, date, sleep
		eventNames := sliceutils.Map(events, func(event Event) string { return event.Name })
		expected := []string{"/bin/date", "/bin/sh", "/bin/sleep", "/usr/sbin/nginx"}

		retryEventsT.Logf("Event names: %+v", eventNames)
		retryEventsT.Logf("Expected names: %+v", expected)
		require.ElementsMatch(retryEventsT, eventNames, expected)

		// Verify the pod's timestamp is no later than the timestamp of the earliest event.
		// Find the actual earliest event (GraphQL doesn't guarantee ordering)
		earliestEventTime := events[0].Timestamp.Time
		for _, event := range events[1:] {
			if event.Timestamp.Time.Before(earliestEventTime) {
				earliestEventTime = event.Timestamp.Time
			}
		}
		retryEventsT.Logf("Pod start comparison: %s vs %s (earliest event)", pod.Started, earliestEventTime)
		require.False(retryEventsT, pod.Started.After(earliestEventTime),
			"pod start time (%s) should not be after earliest event time (%s)", pod.Started, earliestEventTime)

		// Verify risk event timeline csv
		retryEventsT.Logf("Verifying CSV export with %d events", len(eventNames))
		verifyRiskEventTimelineCSV(retryEventsT, deploymentID, eventNames)
	})
}

func getDeploymentID(t testutils.T, deploymentName string) string {
	var respData struct {
		Deployments []IDStruct `json:"deployments"`
	}

	makeGraphQLRequest(t, `
		query deployments($query: String) {
			deployments(query: $query) {
				id
				name
			}
		}
	`, map[string]interface{}{
		"query": fmt.Sprintf("Deployment: %s", deploymentName),
	}, &respData, timeout)
	log.Info(respData)
	require.Len(t, respData.Deployments, 1)

	return string(respData.Deployments[0].ID)
}

func getPods(t testutils.T, deploymentID string) []Pod {
	var respData struct {
		Pods []Pod `json:"pods"`
	}

	// Using this to ensure pagination does not fail.
	offset := int32(0)
	limit := int32(10)
	field := "Pod Name"
	pagination := inputtypes.Pagination{
		Offset: &offset,
		Limit:  &limit,
		SortOption: &inputtypes.SortOption{
			Field: &field,
		},
	}

	makeGraphQLRequest(t, `
		query getPods($podsQuery: String, $pagination: Pagination) {
			pods(query: $podsQuery, pagination: $pagination) {
				id
				name
				containerCount
				started
				events {
					id
					name
				}
			}
		}
	`, map[string]interface{}{
		"podsQuery":  fmt.Sprintf("Deployment ID: %s", deploymentID),
		"pagination": pagination,
	}, &respData, timeout)
	log.Infof("%+v", respData)

	return respData.Pods
}

func getPodCount(t testutils.T, deploymentID string) int {
	var respData struct {
		PodCount int32 `json:"podCount"`
	}

	makeGraphQLRequest(t, `
		query getPodCount($podsQuery: String) {
			podCount(query: $podsQuery)
		}
	`, map[string]interface{}{
		"podsQuery": fmt.Sprintf("Deployment ID: %s", deploymentID),
	}, &respData, timeout)
	log.Infof("%+v", respData)

	return int(respData.PodCount)
}

func getEvents(t testutils.T, pod Pod) []Event {
	var respData struct {
		Pod Pod `json:"pod"`
	}

	makeGraphQLRequest(t, `
		query getEvents($podId: ID!) {
			pod(id: $podId) {
				id
				name
				containerCount
				started
				events {
					id
					name
					timestamp
				}
			}
		}
	`, map[string]interface{}{
		"podId": pod.ID,
	}, &respData, timeout)
	log.Infof("Get Events: %+v", respData)

	return respData.Pod.Events
}
