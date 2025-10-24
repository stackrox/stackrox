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
	"github.com/stackrox/rox/generated/storage"
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

// setupMultiContainerPodTest handles common pod setup: create K8s pod, wait for running,
// wait for Central ingestion, get deployment ID and pod data.
// Returns: k8sPod, deploymentID, pod, cleanup function
func setupMultiContainerPodTest(t *testing.T) (*coreV1.Pod, string, Pod, func()) {
	kPod := getPodFromFile(t, "yamls/multi-container-pod.yaml")
	client := createK8sClient(t)

	var k8sPod *coreV1.Pod
	cleanup := func() { teardownPod(t, client, kPod) }

	// Retry the entire setup to handle transient K8s API issues, slow pod startup, and Central ingestion lag
	testutils.Retry(t, 5, 10*time.Second, func(retryT testutils.T) {
		// Ensure pod exists (idempotent - safe to retry even if pod already exists)
		ensurePodExists(retryT, client, kPod)
		// Wait for pod to be fully running with all containers ready
		k8sPod = waitForPodRunning(retryT, client, kPod.GetNamespace(), kPod.GetName())
		t.Logf("Pod %s is running with all containers ready", kPod.GetName())

		// Now wait for Central to see the deployment
		// This can take time as Sensor needs to detect the pod and report it to Central
		t.Logf("Waiting for Central to see deployment %s", kPod.GetName())
		waitForDeployment(retryT, kPod.GetName())
		t.Logf("Central now sees deployment %s", kPod.GetName())
	})

	deploymentID := ""
	var pod Pod
	testutils.Retry(t, 5, 10*time.Second, func(deplRetryT testutils.T) {
		deploymentID = getDeploymentID(deplRetryT, kPod.GetName())
		deplRetryT.Logf("Central sees the deployment under ID %s", deploymentID)

		// Verify Central sees exactly 1 pod for this deployment
		podCount := getPodCountInCentral(deplRetryT, deploymentID)
		require.Equal(deplRetryT, 1, podCount, "Central should see exactly 1 pod for deployment %s", deploymentID)

		pods := getPods(deplRetryT, deploymentID)
		require.Len(deplRetryT, pods, 1)
		pod = pods[0]
		require.Equal(deplRetryT, int32(2), pod.ContainerCount)
		deplRetryT.Logf("Creation timestamps comparison: %s vs %s",
			k8sPod.GetCreationTimestamp().Time.UTC(), pod.Started.UTC())
		require.Equal(deplRetryT, k8sPod.GetCreationTimestamp().Time.UTC(), pod.Started.UTC())
	})

	return k8sPod, deploymentID, pod, cleanup
}

// skipIfNoCollection skips the test if COLLECTION_METHOD=NO_COLLECTION is set
func skipIfNoCollection(t *testing.T) {
	if os.Getenv("COLLECTION_METHOD") == "NO_COLLECTION" {
		t.Logf("Skipping test that relates to events because env var \"COLLECTION_METHOD\" is set to \"NO_COLLECTION\"")
		t.SkipNow()
	}
}

// waitForSensorHealthy waits for Sensor to be healthy both in Kubernetes and as reported by Central.
// This ensures the Collector->Sensor->Central event pipeline is ready before tests that depend on process events.
func waitForSensorHealthy(t *testing.T) {
	ctx := context.Background()
	client := createK8sClient(t)

	t.Log("Waiting for Sensor to be healthy before starting test")

	// Create a minimal KubernetesSuite to use existing helper methods
	ks := &KubernetesSuite{
		k8s: client,
	}
	ks.SetT(t)

	// Wait for Sensor deployment to be ready in Kubernetes
	ks.waitUntilK8sDeploymentReady(ctx, "stackrox", "sensor")

	// Wait for Central to report healthy connection with Sensor
	waitUntilCentralSensorConnectionIs(t, ctx, storage.ClusterHealthStatus_HEALTHY)

	t.Log("Sensor is healthy and ready")
}

// verifyStartTimeBeforeEvents verifies that a start time is not after the earliest event timestamp
func verifyStartTimeBeforeEvents(t testutils.T, startTime graphql.Time, events []Event, contextMsg string) {
	if len(events) == 0 {
		return
	}
	earliestEventTime := events[0].Timestamp.Time
	for _, event := range events[1:] {
		if event.Timestamp.Time.Before(earliestEventTime) {
			earliestEventTime = event.Timestamp.Time
		}
	}
	t.Logf("%s start comparison: %s vs %s (earliest event)", contextMsg, startTime, earliestEventTime)
	require.False(t, startTime.After(earliestEventTime),
		"%s: start time (%s) should not be after earliest event time (%s)",
		contextMsg, startTime, earliestEventTime)
}

func TestPod(testT *testing.T) {
	skipIfNoCollection(testT)

	// Wait for Sensor to be healthy to ensure the event collection pipeline is ready
	// after any previous tests that may have restarted Sensor.
	waitForSensorHealthy(testT)

	_, deploymentID, pod, cleanup := setupMultiContainerPodTest(testT)
	defer cleanup()

	testutils.Retry(testT, 30, 5*time.Second, func(retryEventsT testutils.T) {
		events := getEvents(retryEventsT, pod)
		retryEventsT.Logf("Found %d events: %+v", len(events), events)

		// Use "at least" semantics: verify required processes exist, but allow extras.
		// Rationale: nginx spawns worker processes (creating duplicate /usr/sbin/nginx events),
		// and docker-entrypoint scripts may create short-lived utility processes
		// (/docker-entrypoint.sh, /usr/bin/find, /bin/grep, etc.) that get captured.
		// This approach makes the test robust against image changes and process lifecycle variations.

		eventNames := sliceutils.Map(events, func(event Event) string { return event.Name })
		retryEventsT.Logf("Event names: %+v", eventNames)

		// Required processes from both containers
		// TODO(ROX-31331): Collector cannot reliably detect /bin/sh /bin/date or /bin/sleep in ubuntu image,
		// thus not including it in the required processes.
		requiredProcesses := []string{"/usr/sbin/nginx"}
		require.Subsetf(retryEventsT, eventNames, requiredProcesses,
			"Pod: required processes: %v not found in events: %v", requiredProcesses, eventNames)

		// Verify the pod's timestamp is no later than the timestamp of the earliest event
		verifyStartTimeBeforeEvents(retryEventsT, pod.Started, events, "Pod")

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

// getPodCountInCentral queries Central via GraphQL to get the number of pods for a deployment.
// This ensures Central has properly ingested the pod from Sensor.
func getPodCountInCentral(t testutils.T, deploymentID string) int {
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
	log.Infof("Pod count in Central for deployment %s: %d", deploymentID, respData.PodCount)

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
