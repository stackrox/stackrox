//go:build test_e2e

package tests

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"k8s.io/client-go/rest"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
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

	// readOnlyAPITokenName is the name of the ready only stackrox API token that is created (and revoked)
	// by this suite.
	readOnlyAPITokenName = "dele-scan-test-read-only"
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
	restCfg            *rest.Config

	readOnlyToken string
}

func TestDelegatedScanning(t *testing.T) {
	suite.Run(t, new(DelegatedScanningSuite))
}

func (ts *DelegatedScanningSuite) SetupSuite() {
	t := ts.T()
	ts.KubernetesSuite.SetupSuite()
	ts.namespace = namespaces.StackRox
	ts.ctx, ts.cleanupCtx, ts.cancel = testContexts(t, "TestDelegatedScanning", 15*time.Minute)

	ts.restCfg = getConfig(t)
	ctx := ts.ctx

	// Create a read only token (needed by some tests)
	_, ts.readOnlyToken = ts.createAPIToken(t, ctx, readOnlyAPITokenName, []string{"Analyst"})

	// Some tests rely on Sensor debug logs to accurately validate expected behaviors.
	ts.origSensorLogLevel, _ = ts.getDeploymentEnvVal(ctx, ts.namespace, sensorDeployment, sensorContainer, logLevelEnvVar)
	if ts.origSensorLogLevel != desiredLogLevel {
		ts.mustSetDeploymentEnvVal(ctx, ts.namespace, sensorDeployment, sensorContainer, logLevelEnvVar, desiredLogLevel)
		t.Logf("Log level env var changed from %q to %q on Sensor", ts.origSensorLogLevel, desiredLogLevel)
	}

	// The changes above may have triggered a Sensor restart, wait for it to be healthy.
	t.Log("Waiting for Sensor to be ready")
	ts.waitUntilK8sDeploymentReady(ctx, ts.namespace, sensorDeployment)
	t.Log("Waiting for Central/Sensor connection to be ready")
	waitUntilCentralSensorConnectionIs(t, ctx, storage.ClusterHealthStatus_HEALTHY)

	// Get the remote cluster to send delegated scans too, will use this to obtain the cluster name, Id, etc.
	t.Log("Getting remote stackrox cluster details")
	ts.remoteCluster = mustGetCluster(t, ctx)
}

func (ts *DelegatedScanningSuite) TearDownSuite() {
	t := ts.T()
	ctx := ts.cleanupCtx
	ns := ts.namespace

	// Collect logs if any test failed, do this first in case other tear down tasks clear logs via pod restarts.
	if t.Failed() {
		ts.logf("Test failed. Collecting k8s artifacts before cleanup.")
		// TODO: DAVE uncomment me before PR review
		// collectLogs(t, ts.namespace, "delegated-scanning-failure")
	}

	ts.revokeAPIToken(t, ctx, readOnlyAPITokenName)

	// Reset the log level back to its original value so that other tests are not impacted by the additional logging.
	if ts.origSensorLogLevel != desiredLogLevel {
		if ts.origSensorLogLevel != "" {
			ts.mustSetDeploymentEnvVal(ctx, ns, sensorDeployment, sensorContainer, logLevelEnvVar, ts.origSensorLogLevel)
			t.Logf("Log level reverted back to %q on Sensor", ts.origSensorLogLevel)
		} else {
			ts.mustDeleteDeploymentEnvVar(ctx, ns, sensorDeployment, logLevelEnvVar)
			t.Logf("Log level env var removed from Sensor")
		}
	}

	// Ensure central/sensor are in a good state before moving to next test's to avoid flakes.
	t.Log("Waiting for Sensor to be ready (On Tear Down)")
	ts.waitUntilK8sDeploymentReady(ctx, ns, sensorDeployment)
	t.Log("Waiting for Central/Sensor connection to be ready (On Tear Down)")
	waitUntilCentralSensorConnectionIs(t, ctx, storage.ClusterHealthStatus_HEALTHY)

	ts.cancel()
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

		ts.waitUntilLog(ctx, ts.namespace, sensorPodLabels, sensorContainer, "contain delegated registry config upsert",
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

	conn := centralgrpc.GRPCConnectionToCentral(t)
	service := v1.NewImageServiceClient(conn)

	sensorPod, err := ts.getSensorPod(ctx, ts.namespace)
	require.NoError(t, err)

	// If delegated scanning has been enabled for 'All Registries', scans in Sensor
	// may be rate limited, retry the scan when rate limited.
	maxTries := 30
	betweenAttemptsFunc := func(num int) {
		t.Logf("Too many parallel scans, trying again, attempt %d/%d", num, maxTries)
	}

	scanImgReq := func(imgStr string) *v1.ScanImageRequest {
		return &v1.ScanImageRequest{
			ImageName: imgStr,
			Force:     true,
			Cluster:   ts.remoteCluster.GetName(),
		}
	}

	t.Run("ad-hoc scan without auth", func(t *testing.T) {
		fromLine, err := ts.getNumLogLines(ctx, ts.namespace, sensorPod.GetName(), sensorPod.Spec.Containers[0].Name)
		require.NoError(t, err)
		ts.logf("Only matching sensor logs after line %d from pod %s", fromLine, sensorPod.GetName())

		// This image was chosen because it requires no auth and is small (no other reason).
		imgStr := "registry.access.redhat.com/ubi9/ubi-minimal:9.4-1194"
		var img *storage.Image
		scanImageFunc := func() error {
			img, err = service.ScanImage(ctx, scanImgReq(imgStr))

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
		ts.validateImageScan(t, img)

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

		fromLine, err := ts.getNumLogLines(ctx, ts.namespace, sensorPod.GetName(), sensorPod.Spec.Containers[0].Name)
		require.NoError(t, err)
		ts.logf("Only matching sensor logs after line %d from pod %s", fromLine, sensorPod.GetName())

		// Push an to the OCP internal registry.
		ocpImgBuilder := deleScanOCPTestImageBuilder{
			t:         t,
			ctx:       ts.ctx,
			namespace: ts.namespace,
			restCfg:   ts.restCfg,
		}
		name := "test-01"
		fromImage := "registry.access.redhat.com/ubi9/ubi-minimal:9.4-1227"
		imgStr := ocpImgBuilder.BuildOCPInternalImage(name, fromImage)

		// Scan the image.
		img, err := service.ScanImage(ctx, scanImgReq(imgStr))
		require.NoError(t, err)
		require.Equal(t, imgStr, img.GetName().GetFullName())
		ts.validateImageScan(t, img)

		// Verify the request made it to the cluster.
		reStr := fmt.Sprintf(`Image "%s".* enriched with metadata using pull source`, imgStr)
		ts.waitUntilLog(ctx, ts.namespace, sensorPodLabels, sensorContainer, "contain the image scan",
			containsLineMatchingAfter(regexp.MustCompile(reStr), fromLine),
		)
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

type DaveSuite struct {
	KubernetesSuite
	namespace string
	restCfg   *rest.Config
	ctx       context.Context
}

func TestDaveDeleScan(t *testing.T) {
	suite.Run(t, new(DaveSuite))
}

func (ts *DaveSuite) SetupSuite() {
	ts.KubernetesSuite.SetupSuite()
	ts.namespace = namespaces.StackRox
	ts.restCfg = getConfig(ts.T())
	ts.ctx = context.Background()
}

func (ts *DaveSuite) TestDeleScan() {
	ocpImgBuilder := deleScanOCPTestImageBuilder{
		t:         ts.T(),
		ctx:       ts.ctx,
		namespace: ts.namespace,
		restCfg:   ts.restCfg,
	}

	name := "dave"
	fromImage := "registry.access.redhat.com/ubi9/ubi-minimal:9.4-1227"
	ref := ocpImgBuilder.BuildOCPInternalImage(name, fromImage)

	ts.T().Logf("Image: %v", ref)
}

// validateImageScan will fail the test if the image's scan was not completed
// successfully.
func (ts *DelegatedScanningSuite) validateImageScan(t *testing.T, img *storage.Image) {
	require.NotNil(t, img.GetScan(), "image scan is nil, check logs for errors, image notes: %v", img.GetNotes())
	require.NotEmpty(t, img.GetScan().GetComponents(), "image scan has no components, check central logs for errors, this can happen if indexing succeeds but matching fails, ROX-17472 will make this an error in the future")
	require.True(t, img.GetIsClusterLocal())
}

func (ts *DelegatedScanningSuite) createReadOnlyToken(t *testing.T) {

}
