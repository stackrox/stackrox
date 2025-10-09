//go:build test_e2e || test_compatibility

package tests

import (
	"context"
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	testT.Skip("Flaky: https://issues.redhat.com/browse/ROX-29771")
	// https://stack-rox.atlassian.net/browse/ROX-6631
	// - the process events expected in this test are not reliably detected.

	kPod := getPodFromFile(testT, "yamls/multi-container-pod.yaml")
	client := createK8sClient(testT)

	// Increased outer retry: 5 attempts instead of 3 to handle transient infrastructure issues
	testutils.Retry(testT, 5, 10*time.Second, func(retryT testutils.T) {
		defer teardownPod(testT, client, kPod)
		createPod(testT, client, kPod)

		// Wait for pod to be fully running before proceeding
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		var k8sPod *coreV1.Pod
		testutils.Retry(retryT, 30, 2*time.Second, func(waitT testutils.T) {
			var err error
			k8sPod, err = client.CoreV1().Pods(kPod.GetNamespace()).Get(ctx, kPod.GetName(), metav1.GetOptions{})
			require.NoError(waitT, err, "failed to get pod %s", kPod.GetName())
			require.Equal(waitT, coreV1.PodRunning, k8sPod.Status.Phase, "pod not in Running phase yet")

			// Ensure all containers are ready before checking for process events
			for _, status := range k8sPod.Status.ContainerStatuses {
				require.True(waitT, status.Ready, "container %s not ready", status.Name)
			}
		})

		testT.Logf("Pod %s is running with all containers ready", k8sPod.Name)

		// Give collector a moment to start detecting processes after containers are ready
		time.Sleep(5 * time.Second)

		// Get the test deployment.
		deploymentID := getDeploymentID(retryT, kPod.GetName())

		podCount := getPodCount(retryT, deploymentID)
		testT.Logf("Pod count: %d", podCount)
		require.Equal(retryT, 1, podCount)

		// Get the test pod.
		pods := getPods(retryT, deploymentID)
		testT.Logf("Num pods: %d", len(pods))
		require.Len(retryT, pods, 1)
		pod := pods[0]

		testT.Logf("Pod: %+v", pod)

		// Verify the container count.
		require.Equal(retryT, int32(2), pod.ContainerCount)

		// Verify Pod start time is the creation time.
		testT.Logf("Creation timestamps comparison: %s vs %s", k8sPod.GetCreationTimestamp().Time.UTC(), pod.Started.UTC())
		require.Equal(retryT, k8sPod.GetCreationTimestamp().Time.UTC(), pod.Started.UTC())

		if os.Getenv("COLLECTION_METHOD") == "NO_COLLECTION" {
			testT.Logf("Skipping parts of TestPod that relate to events because env var \"COLLECTION_METHOD\" is " +
				"set to \"NO_COLLECTION\"")
			return
		}
		testutils.Retry(retryT, 30, 5*time.Second, func(retryEventsT testutils.T) {
			events := getEvents(retryEventsT, pod)
			testT.Logf("Found %d events (expected 4): %+v", len(events), events)

			// Verify we have all 4 expected events
			require.Len(retryEventsT, events, 4, "expected 4 process events")

			// Expecting processes: nginx, sh, date, sleep
			eventNames := sliceutils.Map(events, func(event Event) string { return event.Name })
			expected := []string{"/bin/date", "/bin/sh", "/bin/sleep", "/usr/sbin/nginx"}

			testT.Logf("Event names: %+v", eventNames)
			testT.Logf("Expected names: %+v", expected)
			require.ElementsMatch(retryEventsT, eventNames, expected)

			// Verify the pod's timestamp is no later than the timestamp of the earliest event.
			testT.Logf("Pod start comparison: %s vs %s", pod.Started, events[0].Timestamp.Time)
			require.False(retryEventsT, pod.Started.After(events[0].Timestamp.Time))

			// Verify risk event timeline csv
			testT.Logf("Verifying CSV export with %d events", len(eventNames))
			verifyRiskEventTimelineCSV(retryEventsT, deploymentID, eventNames)
		})
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
