package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/graphql/resolvers/inputtypes"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Note: Update here if yamls/multi-container-pod.yaml is updated
const (
	deploymentName = "end-to-end-api-test-pod-multi-container"
	podName        = "end-to-end-api-test-pod-multi-container"
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
	// Set up testing environment
	testutils.Retry(testT, 3, 5*time.Second, func(retryT testutils.T) {
		defer teardownDeploymentFromFile(retryT, deploymentName, "yamls/multi-container-pod.yaml")
		setupDeploymentFromFile(retryT, deploymentName, "yamls/multi-container-pod.yaml")

		// Get the test deployment.
		deploymentID := getDeploymentID(retryT, deploymentName)

		podCount := getPodCount(retryT, deploymentID)
		if podCount != 1 {
			log.Infof("Pod count is not 1: %d", podCount)
		}
		require.Equal(retryT, 1, podCount)

		// Get the test pod.
		pods := getPods(retryT, deploymentID)
		if len(pods) != 1 {
			log.Infof("Length Pod count is not 1: %d", len(pods))
		}
		require.Len(retryT, pods, 1)
		pod := pods[0]

		// Verify the container count.
		require.Equal(retryT, int32(2), pod.ContainerCount)

		// Verify the events.
		var loopCount int
		var events []Event
		for {
			events = getEvents(retryT, pod)
			log.Infof("%d: Events: %+v", loopCount, events)
			if len(events) == 4 {
				break
			}
			loopCount++
			require.LessOrEqual(retryT, loopCount, 20)
			time.Sleep(4 * time.Second)
		}

		// Expecting processes: nginx, sh, date, sleep
		eventNames := sliceutils.Map(events, func(event Event) string { return event.Name })
		expected := []string{"/bin/date", "/bin/sh", "/bin/sleep", "/usr/sbin/nginx"}

		log.Infof("Event names: %+v", eventNames)
		log.Infof("Expected name: %+v", expected)
		require.ElementsMatch(retryT, eventNames, expected)

		// Verify the pod's timestamp is no later than the timestamp of the earliest event.
		if pod.Started.After(events[0].Timestamp.Time) {
			log.Infof("Pod start is after event time: %s vs %s", pod.Started, events[0].Timestamp.Time)
		}
		require.False(retryT, pod.Started.After(events[0].Timestamp.Time))

		// Verify risk event timeline csv
		log.Infof("Before CSV CHeck")
		verifyRiskEventTimelineCSV(retryT, deploymentID, eventNames)
		log.Infof("After CSV CHeck")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		k8sPod, err := createK8sClient(testT).CoreV1().Pods("default").Get(ctx, podName, metav1.GetOptions{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Pod",
				APIVersion: "v1",
			},
		})
		if err != nil {
			log.Errorf("Error: %v", err)
		}
		require.NoError(retryT, err)
		// Verify Pod start time is the creation time.
		if !k8sPod.GetCreationTimestamp().Time.UTC().Equal(pod.Started.UTC()) {
			log.Infof("Creation timestamps are different: %s vs %s", k8sPod.GetCreationTimestamp().Time.UTC(), pod.Started.UTC())
		}
		require.Equal(retryT, k8sPod.GetCreationTimestamp().Time.UTC(), pod.Started.UTC())
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
