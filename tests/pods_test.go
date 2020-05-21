package tests

import (
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

func TestPod(t *testing.T) {
	// Set up testing environment
	setupDeploymentFromFile(t, deploymentName, "yamls/multi-container-pod.yaml")
	defer teardownDeploymentFromFile(t, deploymentName, "yamls/multi-container-pod.yaml")

	// Get the test deployment.
	deploymentID := getDeploymentID(t, deploymentName)
	require.Equal(t, 1, getPodCount(t, deploymentID))

	// Get the test pod.
	pods := getPods(t, deploymentID)
	require.Len(t, pods, 1)
	pod := pods[0]

	// Verify the container count.
	require.Equal(t, int32(2), pod.ContainerCount)

	// Verify the events.
	testutils.Retry(t, 20, 3*time.Second, func(t testutils.T) {
		events := getEvents(t, pod)
		// Expecting 4 processes: nginx, sh, date, sleep
		require.Len(t, events, 4)
		eventNames := sliceutils.Map(events, func(event Event) string { return event.Name })
		require.ElementsMatch(t, eventNames, []string{"/bin/date", "/bin/sh", "/usr/sbin/nginx", "/bin/sleep"})

		// Verify the pod's timestamp is no later than the timestamp of the earliest event.
		require.False(t, pod.Started.After(events[0].Timestamp.Time))
	})

	k8sPod, err := createK8sClient(t).CoreV1().Pods("default").Get(podName, metav1.GetOptions{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
	})
	require.NoError(t, err)
	// Verify Pod start time is the creation time.
	require.Equal(t, k8sPod.GetCreationTimestamp().Time.UTC(), pod.Started.UTC())
}

func getDeploymentID(t *testing.T, deploymentName string) string {
	var respData struct {
		Deployments []IDStruct `json:"deployments"`
	}

	makeGraphQLRequest(t, `
		query deployments($query: String) {
			deployments(query: $query) {
				id
			}
		}
	`, map[string]interface{}{
		"query": fmt.Sprintf("Deployment: %s", deploymentName),
	}, &respData, timeout)
	log.Info(respData)
	require.Len(t, respData.Deployments, 1)

	return string(respData.Deployments[0].ID)
}

func getPods(t *testing.T, deploymentID string) []Pod {
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

func getPodCount(t *testing.T, deploymentID string) int {
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
	log.Infof("%+v", respData)

	return respData.Pod.Events
}
