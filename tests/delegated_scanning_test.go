//go:build test_e2e

package tests

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	buildv1 "github.com/openshift/api/build/v1"
	buildv1client "github.com/openshift/client-go/build/clientset/versioned/typed/build/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/retry"
	pkgTar "github.com/stackrox/rox/pkg/tar"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common/scan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Delegated Scanning tests that verify the expected behavior for scans executed in
// the secured cluster(s).
//
// These tests should NOT be run in parallel with other tests.  Changes are made to
// the delegated scanning config and OCP mirroring CRs (ie: ImageContentSourcePolicy,
// ImageDigestMirrorSet, ImageTagMirrorSet) which may break other unrelated tests.
//
// These tests DO NOT validate scan results contain specific packages or vulnerabilities,
// they instead focus on the foundational capabilities to index images in secured
// clusters.
//
// These tests require the following env vars to be set in order to contact ACS/K8s:
//   ROX_USERNAME, ROX_PASSWORD, API_ENDPOINT, KUBECONFIG

const (
	logLevelEnvVar = "LOGLEVEL"

	// desiredLogLevel is the desired value of Sensor's log level env var for executing these tests.
	desiredLogLevel = "Debug"
)

type DelegatedScanningSuite struct {
	KubernetesSuite
	ctx        context.Context
	cleanupCtx context.Context
	cancel     func()

	// origSensorLogLevel is the value of Sensor's log level env var prior to any changes / this suite executing.
	origSensorLogLevel string
	remoteCluster      *storage.Cluster
	namespace          string
}

func (ts *DelegatedScanningSuite) SetupSuite() {
	ts.namespace = namespaces.StackRox
	ts.KubernetesSuite.SetupSuite()
	ts.ctx, ts.cleanupCtx, ts.cancel = testContexts(ts.T(), "TestDelegatedScanning", 15*time.Minute)

	// Some tests rely on Sensor debug logs to accurately validate expected behaviors.
	ts.origSensorLogLevel = ts.getDeploymentEnvVal(ts.ctx, ts.namespace, sensorDeployment, sensorContainer, logLevelEnvVar)
	if ts.origSensorLogLevel != desiredLogLevel {
		ts.mustSetDeploymentEnvVal(ts.ctx, ts.namespace, sensorDeployment, sensorContainer, logLevelEnvVar, desiredLogLevel)
		ts.T().Logf("Log level env var changed from %q to %q on Sensor", ts.origSensorLogLevel, desiredLogLevel)
	}

	// The changes above may have triggered a Sensor restart, wait for it to be healthy.
	ts.T().Log("Waiting for Sensor to be ready")
	ts.waitUntilK8sDeploymentReady(ts.ctx, ts.namespace, sensorDeployment)
	ts.T().Log("Waiting for Central/Sensor connection to be ready")
	waitUntilCentralSensorConnectionIs(ts.T(), ts.ctx, storage.ClusterHealthStatus_HEALTHY)

	// Get the remote cluster to send delegated scans too, will use this to obtain the cluster name, Id, etc.
	ts.remoteCluster = mustGetCluster(ts.T(), ts.ctx)
}

func (ts *DelegatedScanningSuite) TearDownSuite() {
	// Collect logs if any test failed, do this first in case other tear down tasks clear logs via pod restarts.
	if ts.T().Failed() {
		ts.logf("Test failed. Collecting k8s artifacts before cleanup.")
		// TODO: DAVE uncomment me before PR review
		// collectLogs(ts.T(), ts.namespace, "delegated-scanning-failure")
	}

	// Reset the log level back to its original value so that other tests are not impacted by the additional logging.
	if ts.origSensorLogLevel != desiredLogLevel {
		if ts.origSensorLogLevel != "" {
			ts.mustSetDeploymentEnvVal(ts.cleanupCtx, ts.namespace, sensorDeployment, sensorContainer, logLevelEnvVar, ts.origSensorLogLevel)
			ts.T().Logf("Log level reverted back to %q on Sensor", ts.origSensorLogLevel)
		} else {
			ts.mustDeleteDeploymentEnvVar(ts.cleanupCtx, ts.namespace, sensorDeployment, logLevelEnvVar)
			ts.T().Logf("Log level env var removed from Sensor")
		}
	}

	// Ensure central/sensor are in a good state before moving to next test's to avoid flakes.
	ts.T().Log("Waiting for Sensor to be ready (On Tear Down)")
	ts.waitUntilK8sDeploymentReady(ts.ctx, ts.namespace, sensorDeployment)
	ts.T().Log("Waiting for Central/Sensor connection to be ready (On Tear Down)")
	waitUntilCentralSensorConnectionIs(ts.T(), ts.cleanupCtx, storage.ClusterHealthStatus_HEALTHY)

	ts.cancel()
}

func TestDelegatedScanning(t *testing.T) {
	suite.Run(t, new(DelegatedScanningSuite))
}

// TestDelegatedScanning_Config verifies that changes made to the delegated registry config
// stick and are propagated to the secured clusters.
func (ts *DelegatedScanningSuite) TestDelegatedScanning_Config() {
	t := ts.T()
	ctx := ts.ctx

	conn := centralgrpc.GRPCConnectionToCentral(t)
	service := v1.NewDelegatedRegistryConfigServiceClient(conn)

	// Get the current config so that we can undo any changes when
	// tests are finished.
	origCfg, err := service.GetConfig(ctx, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		// Be a good 'cluster' citizen and return the config back to
		// its original value when tests finish.
		_, _ = service.UpdateConfig(ctx, origCfg)
	})

	// Ensure that at least one valid secured cluster exists, which is required for
	// delegating scans, otherwise fail.
	getClustersResp, err := service.GetClusters(ctx, nil)
	require.NoError(t, err)
	require.Greater(t, len(getClustersResp.Clusters), 0)

	cluster := getClustersResp.Clusters[0]
	require.NotEmpty(t, cluster.GetId())
	require.NotEmpty(t, cluster.GetName())

	// Verify the API returns the expected values when the config
	// is set to it's default values.
	t.Run("config has expected default values", func(t *testing.T) {
		_, err = service.UpdateConfig(ctx, nil)
		require.NoError(t, err)

		cfg, err := service.GetConfig(ctx, nil)
		require.NoError(t, err)
		assert.Equal(t, "", cfg.DefaultClusterId, "default cluster id should be empty")
		assert.Equal(t, v1.DelegatedRegistryConfig_NONE, cfg.EnabledFor)
		assert.Len(t, cfg.Registries, 0)
	})

	// Verify the API returns the same values that were sent in, at the time
	// this test was written the config was always read from Central DB
	// (not from memory), so this test verifies that the config
	// is persisted to the DB and read back accurately.
	t.Run("update config and get same values back", func(t *testing.T) {
		want := &v1.DelegatedRegistryConfig{
			EnabledFor:       v1.DelegatedRegistryConfig_ALL,
			DefaultClusterId: cluster.GetId(),
			Registries: []*v1.DelegatedRegistryConfig_DelegatedRegistry{
				{Path: "quay.io/rhacs-eng/qa:dele-scan", ClusterId: ""},
			},
		}

		got, err := service.UpdateConfig(ctx, want)
		require.NoError(t, err)
		if !assert.True(t, got.EqualVT(want)) {
			t.Logf("\n got: %v\nwant: %v", got, want)
		}

		got, err = service.GetConfig(ctx, nil)
		require.NoError(t, err)
		if !assert.True(t, got.EqualVT(want)) {
			t.Logf("\n got: %v\nwant: %v", got, want)
		}
	})

	// After making a change to the config confirm (via log inspection)
	// that the change is propagated to the secured cluster.
	t.Run("config propagated to secured cluster", func(t *testing.T) {
		// Create a unique random path within the config that will later
		// be sought within Sensor logs.
		path := fmt.Sprintf("example.com/%s", uuid.NewV4().String())
		cfg := &v1.DelegatedRegistryConfig{
			EnabledFor:       v1.DelegatedRegistryConfig_ALL,
			DefaultClusterId: cluster.GetId(),
			Registries: []*v1.DelegatedRegistryConfig_DelegatedRegistry{
				{Path: path, ClusterId: ""},
			},
		}

		_, err := service.UpdateConfig(ctx, cfg)
		require.NoError(t, err)

		ts.waitUntilLog(ctx, ts.namespace, sensorPodLabels, sensorContainer, "contain delegate registry config upsert",
			containsLineMatching(regexp.MustCompile(fmt.Sprintf("Upserted delegated registry config.*%s", path))),
		)
	})
}

// TestDelegatedScanning_ImageIntegrations tests various aspects of image integrations
// (such as syncing) related to delegated scanning.
func (ts *DelegatedScanningSuite) TestDelegatedScanning_ImageIntegrations() {
	t := ts.T()
	ctx := ts.ctx

	conn := centralgrpc.GRPCConnectionToCentral(t)
	service := v1.NewImageIntegrationServiceClient(conn)

	// This test verifies there are no autogenerated image integrations created for the
	// OCP internal registry. This doesn't directly test Delegated Scanning, however
	// the presence of these integrations is a symptom that indicates the detection of
	// OCP internal registry secrets is broken and will result in delegated scanning failures
	// for images from the OCP Internal registries (ex: historic regression ROX-25526).
	//
	// Scan failures are not used to detect this regression because our test environment
	// consists of a single cluster/namespace Stackrox installation (as opposed to multi-cluster).
	// In a single cluster installation scanning images from the OCP internal registry may,
	// by luck, succeed if an autogenerated integration with the appropriate registry access
	// exists at the time of scan.
	t.Run("no autogenerated integrations are created for the OCP internal registry", func(t *testing.T) {
		if !isOpenshift() {
			t.Skip("Skipping test - not an OCP cluster")
		}

		iis, err := service.GetImageIntegrations(ctx, nil)
		require.NoError(t, err)

		ocpRegistryHostname := "image-registry.openshift-image-registry.svc"

		for _, ii := range iis.GetIntegrations() {
			if !ii.GetAutogenerated() {
				continue
			}

			assert.NotContains(t, ii.GetDocker().GetEndpoint(), ocpRegistryHostname,
				fmt.Sprintf("integration %q (ID: %s) should NOT exist, this indicates possible OCP internal registry scan regression, verify that the detection of OCP pull secrets in sensor/kubernetes/listener/resources/secrets.go is working as expected", ii.GetName(), ii.GetId()))
		}
	})

	// Created, updated, and deleted image integrations should be sent to every secured cluster, this
	// gives users control over the credentials used for delegated scanning.
	t.Run("image integrations are kept synced with the secured clusters", func(t *testing.T) {
		ii := &storage.ImageIntegration{
			Name: fmt.Sprintf("dele-scan-test-%s", uuid.NewV4().String()),
			Type: types.DockerType,
			IntegrationConfig: &storage.ImageIntegration_Docker{
				Docker: &storage.DockerConfig{
					Endpoint: "example.com",
				},
			},
			SkipTestIntegration: true,
			Categories: []storage.ImageIntegrationCategory{
				storage.ImageIntegrationCategory_REGISTRY,
			},
		}

		// Create image integration.
		rii, err := service.PostImageIntegration(ctx, ii)
		require.NoError(t, err)
		id := rii.GetId()

		ts.waitUntilLog(ctx, ts.namespace, sensorPodLabels, sensorContainer, "contain the upserted integration",
			// Requires debug logging.
			containsLineMatching(regexp.MustCompile(fmt.Sprintf("Upserted registry integration.*%s", id))),
		)

		// Update image integration.
		rii.GetDocker().Insecure = !rii.GetDocker().Insecure
		_, err = service.UpdateImageIntegration(ctx, &v1.UpdateImageIntegrationRequest{Config: rii, UpdatePassword: false})
		require.NoError(t, err)

		ts.waitUntilLog(ctx, ts.namespace, sensorPodLabels, sensorContainer, "contain the upserted integration",
			// Requires debug logging.
			containsMultipleLinesMatching(regexp.MustCompile(fmt.Sprintf("Upserted registry integration.*%s", id)), 2),
		)

		// Delete the image integration.
		_, err = service.DeleteImageIntegration(ctx, &v1.ResourceByID{Id: rii.GetId()})
		require.NoError(t, err)

		ts.waitUntilLog(ctx, ts.namespace, sensorPodLabels, sensorContainer, "contain the deleted integration",
			// Requires debug logging.
			containsLineMatching(regexp.MustCompile(fmt.Sprintf("Deleted registry integration.*%s", id))),
		)
	})

	// This ensure that autogenerated integrations are NOT sync'd with the secured clusters.
	t.Run("autogenerated integrations are NOT synced with the secured clusters", func(t *testing.T) {
		ii := &storage.ImageIntegration{
			Autogenerated: true,
			Name:          fmt.Sprintf("dele-scan-test-%s", uuid.NewV4().String()),
			Type:          types.DockerType,
			IntegrationConfig: &storage.ImageIntegration_Docker{
				Docker: &storage.DockerConfig{
					Endpoint: "example.com",
				},
			},
			SkipTestIntegration: true,
			Categories: []storage.ImageIntegrationCategory{
				storage.ImageIntegrationCategory_REGISTRY,
			},
		}

		// Create image integration.
		rii, err := service.PostImageIntegration(ctx, ii)
		require.NoError(t, err)
		id := rii.GetId()

		ts.checkLogsMatch(ctx, ts.namespace, sensorPodLabels, sensorContainer, "contain the upserted integration",
			// Requires debug logging.
			containsNoLinesMatching(regexp.MustCompile(fmt.Sprintf("Upserted registry integration.*%s", id))),
		)
	})
}

// TestDelegatedScanning_AdHocRequests test delegating image scans via the API using the
// cluster parameter. The API is used by ad-hoc scanning mechanisms, such as roxctl and
// Jenkins, and the cluster parameter will take precedence over the delegated scanning config.
func (ts *DelegatedScanningSuite) TestDelegatedScanning_AdHocRequests() {
	t := ts.T()
	ctx := ts.ctx

	cluster := mustGetCluster(t, ctx)
	conn := centralgrpc.GRPCConnectionToCentral(t)
	service := v1.NewImageServiceClient(conn)

	sensorPod, err := ts.getSensorPod(ctx, ts.namespace)
	require.NoError(t, err)

	// If delegated scanning has been enabled for all registries, scans in Sensor
	// may be rate limited, retry the scan when rate limited.
	maxTries := 30
	betweenAttemptsFunc := func(num int) {
		t.Logf("Too many parallel scans, trying again, attempt %d/%d", num, maxTries)
	}

	t.Run("ad-hoc scan without auth", func(t *testing.T) {
		fromLine, err := ts.getNumLogLines(ctx, ts.namespace, sensorPod.GetName(), sensorPod.Spec.Containers[0].Name)
		require.NoError(t, err)
		ts.logf("Only matching sensor logs after line %d from pod %s", fromLine, sensorPod.GetName())

		// This image was chosen because it requires no auth and is small (no other reason).
		imgStr := "registry.access.redhat.com/ubi9/ubi-minimal:9.4-1194"
		var img *storage.Image
		scanImageFunc := func() error {
			img, err = service.ScanImage(ctx, &v1.ScanImageRequest{
				ImageName: imgStr,
				Force:     true,
				Cluster:   cluster.GetName(),
			})

			if err != nil && strings.Contains(err.Error(), scan.ErrTooManyParallelScans.Error()) {
				err = retry.MakeRetryable(err)
			}

			return err
		}

		err = retry.WithRetry(scanImageFunc,
			retry.BetweenAttempts(betweenAttemptsFunc),
			retry.Tries(maxTries),
			retry.WithExponentialBackoff(),
			retry.OnlyRetryableErrors(),
		)
		require.NoError(t, err)
		require.Equal(t, imgStr, img.GetName().GetFullName())
		require.NotEmpty(t, img.GetScan().GetComponents())
		require.True(t, img.GetIsClusterLocal())

		reStr := fmt.Sprintf(`Image "%s".* enriched with metadata using pull source`, imgStr)
		ts.waitUntilLog(ctx, ts.namespace, sensorPodLabels, sensorContainer, "contain the image scan",
			containsLineMatchingAfter(regexp.MustCompile(reStr), fromLine),
		)
	})

	// Delete user
	// Delete role
	t.Run("fail scan image with read only user", func(t *testing.T) {
		// Make API request using that user (perhaps NOT via GRPC)
	})

	t.Run("image watch", func(t *testing.T) {
		// set config and do image watch
		// fromLine, err := ts.getNumLogLines(ctx, ts.namespace, sensorPod.GetName(), sensorPod.Spec.Containers[0].Name)
		// require.NoError(t, err)
		// ts.logf("Only matching sensor logs after line %d from pod %s", fromLine, sensorPod.GetName())

		// // This image was chosen because it requires no auth and is small (no other reason).
		// imgStr := "registry.access.redhat.com/ubi9/ubi-minimal:9.4-1194"
		// service := v1.NewImageServiceClient(conn)
		// _, err = service.WatchImage(ctx, &v1.WatchImageRequest{Name: imgStr})
		// require.NoError(t, err)

		// _, err = service.UnwatchImage(ctx, &v1.UnwatchImageRequest{Name: imgStr})
		// assert.NoError(t, err)
	})

	t.Run("scan image from OCP internal registry", func(t *testing.T) {
		if !isOpenshift() {
			t.Skip("Skipping test - not an OCP cluster")
		}

	})

	// Create role for just image scanning
	// Create user that uses that role
	// Get token for user that uses that role
	t.Run("scan image from OCP internal registry with minimally scoped user", func(t *testing.T) {
		if !isOpenshift() {
			t.Skip("Skipping test - not an OCP cluster")
		}
		// Make API request using that user (perhaps NOT via GRPC)
	})

}

// TestDelegatedScanning_OCPInternalRegistry ...
func (ts *DelegatedScanningSuite) TestDelegatedScanning_OCPInternalRegistry() {
}

// TestDelegatedScanning_ViaConfig ...
func (ts *DelegatedScanningSuite) TestDelegatedScanning_Deployments() {
	t := ts.T()

	// setup
	// set the delegated scanning config to scan images from specific registries/repos
	// delete any test deployments
	// delete any images that exist for the test deployments (to reduce flakes)

	// teardown
	// revert the delegated scanning config, perhaps via defer

	// t.Run("image watch without auth", func(t *testing.T) {
	// 	fromLine, err := ts.getNumLogLines(ctx, ts.namespace, sensorPod.GetName(), sensorPod.Spec.Containers[0].Name)
	// 	require.NoError(t, err)
	// 	ts.logf("Only matching sensor logs after line %d from pod %s", fromLine, sensorPod.GetName())

	// 	// This image was chosen because it requires no auth and is small (no other reason).
	// 	imgStr := "registry.access.redhat.com/ubi9/ubi-minimal:9.4-1194"
	// 	service := v1.NewImageServiceClient(conn)
	// 	_, err = service.WatchImage(ctx, &v1.WatchImageRequest{Name: imgStr})
	// 	require.NoError(t, err)

	// 	_, err = service.UnwatchImage(ctx, &v1.UnwatchImageRequest{Name: imgStr})
	// 	assert.NoError(t, err)
	// })
	t.Run("scan deployment for OCP internal registry", func(t *testing.T) {})
	t.Run("scan deployment all registries", func(t *testing.T) {})
	t.Run("scan deployment specific registries", func(t *testing.T) {})
}

// TestDelegatedScansUsingMirrors will setup mirroring on a cluster and validate
// that delegated scanning is able to scan images from mirrors defined by the various
// mirroring CRs (ie: ImageContentSourcePolicy, ImageDigestMirrorSet, ImageTagMirrorSet)
func (ts *DelegatedScanningSuite) TestDelegatedScanning_Mirrors() {
	t := ts.T()

	t.Run("Scan deployment from mirror defined by ImageContentSourcePolicy", func(t *testing.T) {})
	t.Run("Scan ad-hoc from mirror defined by ImageContentSourcePolicy", func(t *testing.T) {})
	t.Run("Scan deployment from mirror defined by ImageDigestMirrorSet", func(t *testing.T) {})
	t.Run("Scan deployment from mirror defined by ImageTagMirrorSet", func(t *testing.T) {})
}

func (ts *DelegatedScanningSuite) TestDelegatedScanning_DAVE() {
	t := ts.T()
	ctx := ts.ctx

	restClient := getConfig(t)
	buildV1Client, err := buildv1client.NewForConfig(restClient)
	require.NoError(t, err)

	ns := ts.namespace
	name := "dele-scan-01"

	dir := "testdata/delegatedscanning/binary-build-01"
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	require.NoError(t, pkgTar.FromPath(dir, tw))
	utils.IgnoreError(tw.Close)

	result := &buildv1.Build{}
	c := buildV1Client.RESTClient()
	t.Logf("Posting build")
	err = c.Post().
		Namespace(ns).
		Resource("buildconfigs").
		Name(name).
		SubResource("instantiatebinary").
		Body(buf).
		Do(ctx).
		Into(result)

	require.NoError(t, err)

	dataB, err := json.MarshalIndent(result, "", "  ")
	require.NoError(t, err)
	t.Logf("BuildResult: %v", string(dataB))

	t.Logf("Streaming build logs for %q", result.GetName())
	err = streamBuildLogs(ctx, c, result, ts.namespace)
	require.NoError(t, err)

	t.Logf("Waiting for build %q to complete", result.GetName())
	err = waitForBuildComplete(ctx, buildV1Client.Builds(ns), result.GetName())
	require.NoError(t, err)
}

// streamBuildLogs inspired by https://github.com/openshift/oc/blob/67813c212f6625919fa42524a27c399be653a51f/pkg/cli/startbuild/startbuild.go#L507
func streamBuildLogs(ctx context.Context, buildRestClient rest.Interface, build *buildv1.Build, namespace string) error {
	opts := buildv1.BuildLogOptions{
		Follow: true,
		NoWait: false,
	}
	scheme := runtime.NewScheme()
	buildv1.AddToScheme(scheme)
	var err error
	for {
		rd, err := buildRestClient.Get().
			Namespace(namespace).
			Resource("builds").
			Name(build.GetName()).
			SubResource("log").
			VersionedParams(&opts, runtime.NewParameterCodec(scheme)).Stream(ctx)
		if err != nil {
			fmt.Printf("unable to stream the build logs: %v\n", err)
			time.Sleep(1 * time.Second)
			continue
		}
		defer rd.Close()

		if _, err := io.Copy(os.Stdout, rd); err != nil {
			fmt.Printf("unable to stream the build logs: %v\n", err)
		}
		break
	}
	return err
}

// waitForBuildComplete inspred by https://github.com/openshift/oc/blob/67813c212f6625919fa42524a27c399be653a51f/pkg/cli/startbuild/startbuild.go#L1067
func waitForBuildComplete(ctx context.Context, c buildv1client.BuildInterface, name string) error {
	isOK := func(b *buildv1.Build) bool {
		return b.Status.Phase == buildv1.BuildPhaseComplete
	}
	isFailed := func(b *buildv1.Build) bool {
		return b.Status.Phase == buildv1.BuildPhaseFailed ||
			b.Status.Phase == buildv1.BuildPhaseCancelled ||
			b.Status.Phase == buildv1.BuildPhaseError
	}

	for {
		list, err := c.List(ctx, metav1.ListOptions{FieldSelector: fields.Set{"metadata.name": name}.AsSelector().String()})
		if err != nil {
			return err
		}
		for i := range list.Items {
			if name == list.Items[i].Name && isOK(&list.Items[i]) {
				return nil
			}
			if name != list.Items[i].Name || isFailed(&list.Items[i]) {
				return fmt.Errorf("the build %s/%s status is %q", list.Items[i].Namespace, list.Items[i].Name, list.Items[i].Status.Phase)
			}
		}

		rv := list.ResourceVersion
		w, err := c.Watch(ctx, metav1.ListOptions{FieldSelector: fields.Set{"metadata.name": name}.AsSelector().String(), ResourceVersion: rv})
		if err != nil {
			return err
		}
		defer w.Stop()

		for {
			val, ok := <-w.ResultChan()
			if !ok {
				// reget and re-watch
				break
			}
			if e, ok := val.Object.(*buildv1.Build); ok {
				if name == e.Name && isOK(e) {
					return nil
				}
				if name != e.Name || isFailed(e) {
					return fmt.Errorf("the build %s/%s status is %q", e.Namespace, name, e.Status.Phase)
				}
			}
		}
	}

}
