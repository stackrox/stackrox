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
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/netutil"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		retryT.Logf("Pod %s is running with all containers ready", kPod.GetName())

		// Now wait for Central to see the deployment
		// This can take time as Sensor needs to detect the pod and report it to Central
		retryT.Logf("Waiting for Central to see deployment %s", kPod.GetName())
		waitForDeploymentInCentral(retryT, kPod.GetName())
		retryT.Logf("Central now sees deployment %s", kPod.GetName())
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

	k8sPod, deploymentID, pod, cleanup := setupMultiContainerPodTest(testT)
	defer cleanup()

	const eventRetries = 30
	attempt := 0
	prevEventNamesLen := -1
	testutils.Retry(testT, eventRetries, 5*time.Second, func(retryEventsT testutils.T) {
		attempt++

		events := getEvents(retryEventsT, pod)
		retryEventsT.Logf("Found %d events: %+v", len(events), events)

		// Use "at least" semantics: verify required processes exist, but allow extras.
		// Rationale: nginx spawns worker processes (creating duplicate /usr/sbin/nginx events),
		// and docker-entrypoint scripts may create short-lived utility processes
		// (/docker-entrypoint.sh, /usr/bin/find, /bin/grep, etc.) that get captured.
		// This approach makes the test robust against image changes and process lifecycle variations.

		eventNames := sliceutils.Map(events, func(event Event) string { return event.Name })
		retryEventsT.Logf("Event names: %+v", eventNames)

		// Diagnostics should not influence the test outcome. We run it best-effort and log any failures
		// instead of failing the test.
		//
		// Trigger diagnostics when eventNames length changes (but skip the noisy zero-length case).
		// This provides snapshots for both successful runs and flakes, enabling comparisons across runs.
		curLen := len(eventNames)
		if curLen > 0 && curLen != prevEventNamesLen {
			retryEventsT.Logf("pod.events length changed: %d -> %d (attempt %d/%d)", prevEventNamesLen, curLen, attempt, eventRetries)
			dumpTestPodDiagnostics(retryEventsT, k8sPod.GetNamespace(), k8sPod.GetName(), deploymentID, pod, events)
			prevEventNamesLen = curLen
		}

		// Required processes from both containers
		requiredProcesses := []string{"/usr/sbin/nginx", "/bin/sh", "/bin/date", "/bin/sleep"}
		require.Subsetf(retryEventsT, eventNames, requiredProcesses,
			"Pod: required processes: %v not found in events: %v", requiredProcesses, eventNames)

		// Verify the pod's timestamp is no later than the timestamp of the earliest event
		verifyStartTimeBeforeEvents(retryEventsT, pod.Started, events, "Pod")

		// Verify risk event timeline csv
		retryEventsT.Logf("Verifying CSV export with %d events", len(eventNames))
		verifyRiskEventTimelineCSV(retryEventsT, deploymentID, eventNames)
	})
}

func dumpTestPodDiagnostics(t testutils.T, podNamespace, podName, deploymentID string, pod Pod, podEvents []Event) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	t.Logf("=== TestPod diagnostics (on failure) ===")
	t.Logf("Central pod ID: %s, Pod name: %s, DeploymentID: %s", pod.ID, pod.Name, deploymentID)

	// K8s pod state (helps detect stale pod reuse / container restarts / container IDs).
	k8s := createK8sClient(t)
	if k8sPod, err := k8s.CoreV1().Pods(podNamespace).Get(ctx, podName, metaV1.GetOptions{}); err != nil {
		t.Logf("K8s: failed to get pod %s/%s: %v", podNamespace, podName, err)
	} else {
		t.Logf("K8s pod %s/%s: uid=%s phase=%s creation=%s deletion=%v",
			podNamespace, podName, k8sPod.UID, k8sPod.Status.Phase, k8sPod.CreationTimestamp.UTC(), k8sPod.DeletionTimestamp)
		for _, cs := range k8sPod.Status.ContainerStatuses {
			startedAt := ""
			if cs.State.Running != nil {
				startedAt = cs.State.Running.StartedAt.UTC().String()
			}
			t.Logf("K8s container %q: ready=%v restart=%d image=%q imageID=%q containerID=%q startedAt=%q state=%+v",
				cs.Name, cs.Ready, cs.RestartCount, cs.Image, cs.ImageID, cs.ContainerID, startedAt, cs.State)
		}
	}

	// Sensor restart evidence.
	if sensorPods, err := k8s.CoreV1().Pods("stackrox").List(ctx, metaV1.ListOptions{LabelSelector: "app=sensor"}); err != nil {
		t.Logf("K8s: failed to list sensor pods: %v", err)
	} else {
		for _, sp := range sensorPods.Items {
			t.Logf("K8s sensor pod: %s uid=%s phase=%s", sp.Name, sp.UID, sp.Status.Phase)
			for _, cs := range sp.Status.ContainerStatuses {
				if cs.Name != "sensor" {
					continue
				}
				startedAt := ""
				if cs.State.Running != nil {
					startedAt = cs.State.Running.StartedAt.UTC().String()
				}
				t.Logf("K8s sensor container: pod=%s restart=%d startedAt=%q containerID=%q", sp.Name, cs.RestartCount, startedAt, cs.ContainerID)
			}
		}
	}

	// Central view (cluster health, process service) must be best-effort: never fail the test due to transient
	// network flakes while gathering diagnostics.
	if conn, err := tryGRPCConnectionToCentral(ctx); err != nil {
		t.Logf("Central: skipping gRPC diagnostics (unable to connect): %v", err)
	} else {
		defer conn.Close()

		// Central view of cluster health (includes last contact).
		clustersSvc := v1.NewClustersServiceClient(conn)
		if clustersResp, err := clustersSvc.GetClusters(ctx, &v1.GetClustersRequest{}); err != nil {
			t.Logf("Central: GetClusters failed: %v", err)
		} else if len(clustersResp.GetClusters()) == 1 {
			c := clustersResp.GetClusters()[0]
			hs := c.GetHealthStatus()
			t.Logf("Central cluster health: sensor=%s collector=%s overall=%s lastContact=%v",
				hs.GetSensorHealthStatus(), hs.GetCollectorHealthStatus(), hs.GetOverallHealthStatus(), hs.GetLastContact())
		} else {
			t.Logf("Central: expected 1 cluster, got %d", len(clustersResp.GetClusters()))
		}

		// ProcessService view (bypasses GraphQL Pod.events aggregation).
		procSvc := v1.NewProcessServiceClient(conn)
		if procsResp, err := procSvc.GetGroupedProcessByDeploymentAndContainer(ctx, &v1.GetProcessesByDeploymentRequest{DeploymentId: deploymentID}); err != nil {
			t.Logf("Central: ProcessService GetGroupedProcessByDeploymentAndContainer failed: %v", err)
		} else {
			t.Logf("Central: ProcessService groups=%d (name+container)", len(procsResp.GetGroups()))
			for _, g := range procsResp.GetGroups() {
				t.Logf("Central process group: container=%q name=%q timesExecuted=%d suspicious=%v",
					g.GetContainerName(), g.GetName(), g.GetTimesExecuted(), g.GetSuspicious())
			}
		}
	}

	// What GraphQL pod.events returned on the failing attempt.
	eventNames := sliceutils.Map(podEvents, func(e Event) string { return e.Name })
	t.Logf("GraphQL pod.events count=%d names=%v", len(podEvents), eventNames)
}

func tryGRPCConnectionToCentral(ctx context.Context) (*grpc.ClientConn, error) {
	// Avoid centralgrpc helpers here: they call require/assert internally and would fail the test.
	// This function is used only for diagnostics and must be best-effort.

	// In CI tests this is typically provided as API_ENDPOINT (see pkg/testutils/centralgrpc),
	// but we also accept ROX_ENDPOINT for convenience.
	endpoint := os.Getenv("API_ENDPOINT")
	if endpoint == "" {
		endpoint = os.Getenv("ROX_ENDPOINT")
	}
	if endpoint == "" {
		return nil, fmt.Errorf("missing central endpoint env (API_ENDPOINT or ROX_ENDPOINT)")
	}

	host, _, _, err := netutil.ParseEndpoint(endpoint)
	if err != nil {
		return nil, fmt.Errorf("parsing endpoint %q: %w", endpoint, err)
	}

	opts := clientconn.Options{
		TLS: clientconn.TLSConfigOptions{
			InsecureSkipVerify: true,
			ServerName:         host,
		},
	}

	// Best-effort basic auth (optional for diagnostics; if missing, calls may still succeed depending on config).
	user := os.Getenv("ROX_USERNAME")
	pass := os.Getenv("ROX_ADMIN_PASSWORD")
	if user != "" && pass != "" {
		opts.ConfigureBasicAuth(user, pass)
	}

	dialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return clientconn.GRPCConnection(dialCtx, mtls.CentralSubject, endpoint, opts)
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
	t.Logf("%+v", respData)
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
	t.Logf("%+v", respData)

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
	t.Logf("Pod count in Central for deployment %s: %d", deploymentID, respData.PodCount)

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
	t.Logf("Get Events: %+v", respData)

	return respData.Pod.Events
}
