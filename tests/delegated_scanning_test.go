//go:build test_e2e

package tests

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/docker/config"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	pkgUtils "github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common/scan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

// Delegated Scanning tests verify the expected behavior for scans executed in
// the Secured Cluster(s).
//
// These tests should NOT be run in parallel with other tests. Changes are made to
// the delegated scanning config and OCP mirroring CRs (ie: ImageContentSourcePolicy,
// ImageDigestMirrorSet, ImageTagMirrorSet) which may break other unrelated tests.
//
// These tests DO NOT validate that scan results contain specific packages or vulnerabilities,
// (we rely on the scanner specific tests for that). These tests instead focus on the foundational
// capabilities of delegating scans to Secured Clusters.
//
// These tests require the following env vars to be set:
// - API_ENDPOINT               - location of the StackRox API.
// - ROX_USERNAME               - for admin access to the StackRox API.
// - ROX_ADMIN_PASSWORD         - for admin access to the StackRox API.
// - KUBECONFIG                 - for inspecting logs, creating deploys, etc.
// - QUAY_RHACS_ENG_RO_USERNAME - for reading quay.io/rhacs-eng images.
// - QUAY_RHACS_ENG_RO_PASSWORD - for reading quay.io/rhacs-eng images.
// - ORCHESTRATOR_FLAVOR        - to determine if running on OCP or not.

const (
	// deleScanLogLevelEnvVar the name of Sensor's env var that controls the logging level.
	deleScanLogLevelEnvVar = "LOGLEVEL"

	// deleScanDesiredLogLevel the desired value of Sensor's log level env var for executing these tests.
	deleScanDesiredLogLevel = "Debug"

	// deleScanDefaultLogWaitTimeout the amount of time that the various delegated scanning tests will wait for
	// a log entry to be found.
	deleScanDefaultLogWaitTimeout = 30 * time.Second

	// deleScanDefaultMaxRetries the number of retries when pulling images from the StackRox API or executing
	// scans.
	deleScanDefaultMaxRetries = 30

	// deleScanDefaultRetryDelay the time to wait between retries.
	deleScanDefaultRetryDelay = 5 * time.Second

	// deleScanArtifactsDir the base directory to store test artifacts on failure.
	deleScanArtifactsDir = "dele-scan"
)

// This block contains names for resources created during testing, consistent names
// make cleanup easier.
const (
	deleScanAPITokenName      = "dele-scan-api-token" //nolint:gosec // G101
	deleScanRoleName          = "dele-scan-role"
	deleScanPermissionSetName = "dele-scan-permission-set"
)

var (
	denyAllAccessScope  = accesscontrol.DefaultAccessScopeIDs[accesscontrol.DenyAllAccessScope]
	allowAllAccessScope = accesscontrol.DefaultAccessScopeIDs[accesscontrol.UnrestrictedAccessScope]
)

type DelegatedScanningSuite struct {
	KubernetesSuite
	ctx        context.Context
	cleanupCtx context.Context
	cancel     func()
	restCfg    *rest.Config

	// origSensorLogLevel the value of Sensor's log level env var prior to any changes.
	origSensorLogLevel string

	// remoteCluster represents the Secured Cluster that scans will be delegated to.
	remoteCluster *v1.DelegatedRegistryCluster

	// namespace holds the current k8s namespace that StackRox is installed in and where deployments
	// for testing will be created.
	namespace string

	// Credentials for accessing images from quay.io/rhacs-eng
	quayROUsername string
	quayROPassword string

	// failureHandled is used to ensure cleanup activities are not executed multiple times by the suite.
	failureHandled atomic.Bool

	// deleScanUtils a utility used to setup various aspects of these tests, such as
	// creating images in the OCP image registry or creating mirroring CRs.
	deleScanUtils *deleScanTestUtils

	// ocpInternalImage the image that was created in the OCP internal registry during setup.
	ocpInternalImage deleScanTestImage

	// Images used by various tests, it is ideal to use a unique image for each test
	// scanning images from deployments.
	ubi9Image      deleScanTestImage
	ubi9ImageB     deleScanTestImage
	nginxImage     deleScanTestImage
	httpdImage     deleScanTestImage
	memcachedImage deleScanTestImage
}

// TestDelegatedScanning the entrypoint for all delegated scanning tests.
func TestDelegatedScanning(t *testing.T) {
	suite.Run(t, new(DelegatedScanningSuite))
}

func (ts *DelegatedScanningSuite) SetupSuite() {
	ts.KubernetesSuite.SetupSuite()
	t := ts.T()

	// Ensure failures during setup are handled.
	t.Cleanup(ts.handleFailure)

	ts.ctx, ts.cleanupCtx, ts.cancel = testContexts(t, "TestDelegatedScanning", 30*time.Minute)
	ctx := ts.ctx
	ts.restCfg = getConfig(t)
	ts.namespace = namespaces.StackRox

	// ubi9Image* references images that are small (scans fast), do not require auth, and
	// have at least one vulnerability.
	//
	// A unique image for each deployment test is ideal, otherwise there is a chance a scan
	// triggered from a previous deployment is considered 'new'.
	ts.ubi9Image = NewDeleScanTestImage(t, "registry.access.redhat.com/ubi9/ubi-minimal:9.4-1194@sha256:73f7dcacb460dad137a58f24668470a5a2e47378838a0190eef0ab532c6e8998")
	ts.ubi9ImageB = NewDeleScanTestImage(t, "registry.access.redhat.com/ubi9/ubi-minimal:9.4-1227@sha256:35a12657ce1bcb2b7667f4e6e0147186c1e0172cc43ece5452ab85afd6532791")

	// Dockerfiles for these are located at qa-tests-backend/test-images/delegated-scanning/*.
	//
	// These images were chosen at random due on smallish size, are self running, contain
	// at least one vulnerability, and do not overlap with other images tested in this suite.
	ts.nginxImage = NewDeleScanTestImage(t, "quay.io/rhacs-eng/qa:dele-scan-nginx@sha256:68b418b74715000e41a894428bd787442945592486a08d4cbea89a9b4fa03302")
	ts.httpdImage = NewDeleScanTestImage(t, "quay.io/rhacs-eng/qa:dele-scan-httpd@sha256:489576ec07d6d8d64690bedb4cf1eeb366a8f03f8530367c3eee0c71579b5f5e")
	ts.memcachedImage = NewDeleScanTestImage(t, "itms.invalid/rhacs-eng/qa:dele-scan-memcached@sha256:1cf25340014838bef90aa9d19eaef725a0b4986af3c8e8a6be3203c2cef8cb61")

	// Ensure rhacs-eng repo user/pass is avail for accessing private images.
	ts.quayROUsername = mustGetEnv(t, "QUAY_RHACS_ENG_RO_USERNAME")
	ts.quayROPassword = mustGetEnv(t, "QUAY_RHACS_ENG_RO_PASSWORD")

	// Get a reference to the Secured Cluster to send delegated scans too.
	// If a valid remote cluster is NOT available all tests in this suite will fail.
	logf(t, "Getting remote StackRox cluster details")
	envVal, _ := ts.getDeploymentEnvVal(ctx, ts.namespace, sensorDeployment, sensorContainer, env.LocalImageScanningEnabled.EnvVar())

	// Verify the StackRox installation supports delegated scanning, Central and Sensor
	// must have an active connection for this check to succeed, so wait for that connection.
	ts.waitForHealthyCentralSensorConn()

	conn := centralgrpc.GRPCConnectionToCentral(t)
	service := v1.NewDelegatedRegistryConfigServiceClient(conn)
	clustersResp, err := service.GetClusters(ctx, &v1.Empty{})
	require.NoError(t, err)
	require.Len(t, clustersResp.GetClusters(), 1, "Expected a single valid cluster for executing delegated scans")
	cluster := clustersResp.GetClusters()[0]
	require.True(t, cluster.GetIsValid(),
		"Local scanning is not enabled in the connected secured cluster (%q is %q on Sensor, cluster id: %q, name: %q, is valid: %t)",
		env.LocalImageScanningEnabled.EnvVar(),
		envVal,
		cluster.GetId(),
		cluster.GetName(),
		cluster.GetIsValid(),
	)
	ts.remoteCluster = cluster

	// Enable Sensor debug logs, some tests need this to accurately validate expected behaviors.
	ts.origSensorLogLevel, _ = ts.getDeploymentEnvVal(ctx, ts.namespace, sensorDeployment, sensorContainer, deleScanLogLevelEnvVar)
	if ts.origSensorLogLevel != deleScanDesiredLogLevel {
		ts.mustSetDeploymentEnvVal(ctx, ts.namespace, sensorDeployment, sensorContainer, deleScanLogLevelEnvVar, deleScanDesiredLogLevel)
		logf(t, "Log level env var changed from %q to %q on Sensor", ts.origSensorLogLevel, deleScanDesiredLogLevel)

		ts.waitForHealthyCentralSensorConn()
	}

	// Pre-clean any roles, permissions, etc. from previous runs.
	logf(t, "Deleteing resources from previous tests")
	ts.resetAccess(ctx)

	// Reset the delegated scanning config to default value.
	ts.resetConfig(ctx)

	// Initialize the delegated scanning utility.
	ts.deleScanUtils = NewDeleScanTestUtils(t, ts.restCfg, ts.listK8sAPIResources())

	if isOpenshift() {
		logf(t, "Building and pushing an image to the OCP internal registry")
		ts.ocpInternalImage = ts.deleScanUtils.BuildOCPInternalImage(t, ts.ctx, ts.namespace, "dele-scan-test-01", ts.ubi9Image.TagRef())
		logf(t, "OCP Internal Image: %q", ts.ocpInternalImage.cImage.GetName().GetFullName())
	}
}

func (ts *DelegatedScanningSuite) TearDownSuite() {
	t := ts.T()
	ctx := ts.cleanupCtx

	// Handle data collection on test failures prior to resetting the log level.
	// Changing the log level may result in pod restarts and logs lost.
	ts.handleFailure()

	// Reset the delegated scanning config back so that other tests are not impacted.
	ts.resetConfig(ctx)

	// Reset the log level back to its original value so that other e2e tests are
	// not impacted by the additional logging.
	if ts.origSensorLogLevel != deleScanDesiredLogLevel {
		if ts.origSensorLogLevel != "" {
			ts.mustSetDeploymentEnvVal(ctx, ts.namespace, sensorDeployment, sensorContainer, deleScanLogLevelEnvVar, ts.origSensorLogLevel)
			logf(t, "Log level reverted back to %q on Sensor", ts.origSensorLogLevel)
		} else {
			ts.mustDeleteDeploymentEnvVar(ctx, ts.namespace, sensorDeployment, deleScanLogLevelEnvVar)
			logf(t, "Log level env var removed from Sensor")
		}
	}

	ts.cancel()
}

func (ts *DelegatedScanningSuite) AfterTest(suiteName string, testName string) {
	t := ts.T()

	if t.Failed() {
		// Collect artifacts after each failed test so that we don't lose logs
		// when subsequent tests trigger pod restarts.
		dir := filepath.Join(deleScanArtifactsDir, testName)
		logf(t, "Test failed, collecting artifacts into %q", dir)
		collectLogs(t, ts.namespace, dir)
	}
}

// handleFailure is a catch all for handling test suite failures, invoked via t.Cleanup in SetupSuite AND
// as part of TearDownSuite. We cannot handle failures solely in TearDownSuite because a failure in SetupSuite
// prevents TearDownSuite from executing. Subsequent invocations of this method after the first will be no-ops.
func (ts *DelegatedScanningSuite) handleFailure() {
	if ts.failureHandled.Swap(true) {
		return
	}

	t := ts.T()
	logf(t, "Handling failures (if any)")

	if t.Failed() {
		dir := filepath.Join(deleScanArtifactsDir, "Final")
		ts.logf("Test(s) failed, collecting artifacts before final cleanup into %q", dir)
		collectLogs(t, ts.namespace, dir)
	}
}

// TestConfig verifies that changes made to the delegated registry config
// stick and are propagated to the Secured Clusters.
func (ts *DelegatedScanningSuite) TestConfig() {
	t := ts.T()
	ctx := ts.ctx

	conn := centralgrpc.GRPCConnectionToCentral(t)
	service := v1.NewDelegatedRegistryConfigServiceClient(conn)

	// Verify the API returns the expected values when the config
	// is set to it's default value.
	ts.Run("config has expected default values", func() {
		t := ts.T()

		_, err := service.UpdateConfig(ctx, &v1.DelegatedRegistryConfig{})
		require.NoError(t, err)

		cfg, err := service.GetConfig(ctx, &v1.Empty{})
		require.NoError(t, err)
		assert.Equal(t, "", cfg.DefaultClusterId)
		assert.Equal(t, v1.DelegatedRegistryConfig_NONE, cfg.EnabledFor)
		assert.Len(t, cfg.Registries, 0)
	})

	// Verify the API returns the same values that were sent in. At the time
	// this test was written the config was always read from Central DB
	// (as opposed to in-mem). This test verifies that the config
	// is persisted to the DB and read back accurately.
	ts.Run("update config and get same values back", func() {
		t := ts.T()

		want := &v1.DelegatedRegistryConfig{
			EnabledFor:       v1.DelegatedRegistryConfig_ALL,
			DefaultClusterId: ts.remoteCluster.GetId(),
			Registries: []*v1.DelegatedRegistryConfig_DelegatedRegistry{
				{Path: "quay.io/rhacs-eng/qa:dele-scan", ClusterId: ""},
			},
		}

		got, err := service.UpdateConfig(ctx, want)
		require.NoError(t, err)
		if !assert.True(t, got.EqualVT(want)) {
			logf(t, "\n got: %v\nwant: %v", got, want)
		}

		got, err = service.GetConfig(ctx, &v1.Empty{})
		require.NoError(t, err)
		if !assert.True(t, got.EqualVT(want)) {
			logf(t, "\n got: %v\nwant: %v", got, want)
		}
	})

	// After making a change to the config confirm (via log inspection)
	// that the change is propagated to the Secured Cluster.
	ts.Run("config propagated to secured cluster", func() {
		t := ts.T()

		// Create a unique random path within the config that will later
		// be sought within Sensor logs.
		path := fmt.Sprintf("example.com/%s", uuid.NewV4().String())
		cfg := &v1.DelegatedRegistryConfig{
			EnabledFor:       v1.DelegatedRegistryConfig_ALL,
			DefaultClusterId: ts.remoteCluster.GetId(),
			Registries: []*v1.DelegatedRegistryConfig_DelegatedRegistry{
				{Path: path, ClusterId: ""},
			},
		}

		fromByte := ts.getSensorLastLogBytePos(ctx)

		_, err := service.UpdateConfig(ctx, cfg)
		require.NoError(t, err)

		ts.waitUntilLog(ctx, "contain delegated registry config upsert",
			containsLineMatchingAfter(regexp.MustCompile(fmt.Sprintf("Upserted delegated registry config.*%s", path)), fromByte),
		)
	})
}

// TestImageIntegrations tests various aspects of image integrations
// (such as syncing).
func (ts *DelegatedScanningSuite) TestImageIntegrations() {
	t := ts.T()
	ctx := ts.ctx

	conn := centralgrpc.GRPCConnectionToCentral(t)
	service := v1.NewImageIntegrationServiceClient(conn)

	// This test verifies there are no autogenerated image integrations created for the
	// OCP internal registry. This doesn't directly test delegated scanning, however
	// the presence of these integrations is a symptom indicating the detection of
	// OCP internal registry secrets is broken, which will result in delegated scanning failures
	// (ex: regression ROX-25526).
	//
	// Scan failures are not used to detect this regression because our test environment
	// consists of a single cluster/namespace StackRox installation (as opposed to multi-cluster).
	// In a single cluster installation scanning images from the OCP internal registry may,
	// by luck, succeed if an autogenerated integration with appropriate access
	// exists at the time of scan.
	ts.Run("no autogenerated integrations are created for the OCP internal registry", func() {
		t := ts.T()

		ts.skipIfNotOpenShift()

		iis, err := service.GetImageIntegrations(ctx, &v1.GetImageIntegrationsRequest{})
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

	// Created, updated, and deleted image integrations should be sent to every Secured Cluster, this
	// gives users control over the credentials used for delegated scanning.
	ts.Run("image integrations are synced with the secured clusters", func() {
		t := ts.T()

		fromByte := ts.getSensorLastLogBytePos(ctx)

		ii := ts.createImageIntegration(&storage.DockerConfig{
			Endpoint: "example.com",
		}, false, false)

		// Create image integration.
		id := ii.GetId()

		ts.waitUntilLog(ctx, "contain the upserted integration",
			// Requires debug logging.
			containsLineMatchingAfter(regexp.MustCompile(fmt.Sprintf("Upserted registry integration.*%s", id)), fromByte),
		)

		// Update image integration.
		ii.GetDocker().Insecure = !ii.GetDocker().Insecure
		_, err := service.UpdateImageIntegration(ctx, &v1.UpdateImageIntegrationRequest{Config: ii, UpdatePassword: false})
		require.NoError(t, err)

		ts.waitUntilLog(ctx, "contain the upserted integration",
			// Requires debug logging.
			containsMultipleLinesMatchingAfter(regexp.MustCompile(fmt.Sprintf("Upserted registry integration.*%s", id)), 2, fromByte),
		)

		// Delete the image integration.
		_, err = service.DeleteImageIntegration(ctx, &v1.ResourceByID{Id: id})
		require.NoError(t, err)

		ts.waitUntilLog(ctx, "contain the deleted integration",
			// Requires debug logging.
			containsLineMatchingAfter(regexp.MustCompile(fmt.Sprintf("Deleted registry integration.*%s", id)), fromByte),
		)
	})

	// Ensure that autogenerated integrations are NOT sync'd with the Secured Clusters.
	ts.Run("autogenerated integrations are NOT synced with the secured clusters", func() {
		fromByte := ts.getSensorLastLogBytePos(ctx)

		ii := ts.createImageIntegration(&storage.DockerConfig{
			Endpoint: "example.com",
		}, true, true)

		ts.checkLogsMatch(ctx, "contain the upserted integration",
			// Requires debug logging.
			containsNoLinesMatchingAfter(regexp.MustCompile(fmt.Sprintf("Upserted registry integration.*%s", ii.GetId())), fromByte),
		)
	})
}

// TestAdHocScans tests delegating image scans via the API. The API is
// used by ad-hoc scanning mechanisms, such as roxctl and Jenkins.
func (ts *DelegatedScanningSuite) TestAdHocScans() {
	t := ts.T()
	ctx := ts.ctx

	// For readability.
	withClusterFlag := true

	scanImgReq := func(imageStr string, withClusterFlag bool) *v1.ScanImageRequest {
		cluster := pkgUtils.IfThenElse(withClusterFlag, ts.remoteCluster.GetId(), "")
		return &v1.ScanImageRequest{
			ImageName: imageStr,
			Force:     true,
			Cluster:   cluster,
		}
	}

	conn := centralgrpc.GRPCConnectionToCentral(t)

	// Verifies under normal conditions can successfully delegate a scan when
	// explicitly setting the destination cluster in the API request.
	ts.Run("delegate scan via cluster flag", func() {
		ts.resetConfig(ctx)

		ts.executeAndValidateScan(ctx, conn, scanImgReq(ts.ubi9Image.TagRef(), withClusterFlag))
	})

	// Verifies under normal conditions can successfully delegate a scan when
	// the scan destination is set via the delegated scanning config.
	ts.Run("delegate scan via config", func() {
		ts.setConfig(ctx, ts.ubi9Image)

		ts.executeAndValidateScan(ctx, conn, scanImgReq(ts.ubi9Image.TagRef(), !withClusterFlag))
	})

	// Image write permission is required to scan images, this test uses a token with only image
	// read permissions, and therefore should fail to scan any image.
	ts.Run("fail to scan image with only image read permission", func() {
		t := ts.T()

		ts.resetConfig(ctx)

		ps := &storage.PermissionSet{
			Name: deleScanPermissionSetName,
			ResourceToAccess: map[string]storage.Access{
				"Image": storage.Access_READ_ACCESS,
			},
		}

		role := &storage.Role{
			Name:          deleScanRoleName,
			AccessScopeId: allowAllAccessScope,
		}

		limitedConn := ts.getLimitedCentralConn(ctx, ps, role)
		service := v1.NewImageServiceClient(limitedConn)
		_, err := service.ScanImage(ctx, scanImgReq(ts.ubi9Image.TagRef(), withClusterFlag))
		require.ErrorContains(t, err, "not authorized")
	})

	// Test scanning an image using a token with minimal Image write permission.
	ts.Run("scan image with minimal permissions", func() {
		ts.resetConfig(ctx)

		ps := &storage.PermissionSet{
			Name: deleScanPermissionSetName,
			ResourceToAccess: map[string]storage.Access{
				"Image": storage.Access_READ_WRITE_ACCESS,
			},
		}

		role := &storage.Role{
			Name:          deleScanRoleName,
			AccessScopeId: allowAllAccessScope,
		}

		limitedConn := ts.getLimitedCentralConn(ctx, ps, role)
		ts.executeAndValidateScan(ctx, limitedConn, scanImgReq(ts.ubi9Image.TagRef(), withClusterFlag))
	})

	// Delegate a scan for an image watch.
	ts.Run("delegate scan image via image watch", func() {
		t := ts.T()

		// Set the delegated scanning config to delegate scans for the test image.
		ts.setConfig(ctx, ts.ubi9Image)

		service := v1.NewImageServiceClient(conn)

		// Since we cannot 'force' a scan via an image watch, we first delete
		// the image from Central to help ensure a fresh scan is executed.
		query := fmt.Sprintf("Image:%s", ts.ubi9Image.TagRef())
		delResp, err := service.DeleteImages(ctx, &v1.DeleteImagesRequest{Query: &v1.RawQuery{Query: query}, Confirm: true})
		require.NoError(t, err)
		logf(t, "Num images deleted from query %q: %d", query, delResp.NumDeleted)

		fromByte := ts.getSensorLastLogBytePos(ctx)

		// Setup the image watch.
		resp, err := service.WatchImage(ctx, &v1.WatchImageRequest{Name: ts.ubi9Image.TagRef()})
		require.NoError(t, err)
		require.Zero(t, resp.GetErrorType(), "expected no error")
		require.Equal(t, ts.ubi9Image.TagRef(), resp.GetNormalizedName())
		t.Cleanup(func() { _, _ = service.UnwatchImage(ctx, &v1.UnwatchImageRequest{Name: ts.ubi9Image.TagRef()}) })

		ts.waitUntilSensorLogsScan(ctx, ts.ubi9Image.TagRef(), fromByte)
	})

	// Scan an image from the OCP internal registry
	ts.Run("scan image from OCP internal registry", func() {
		ts.skipIfNotOpenShift()

		ts.resetConfig(ctx)

		ts.executeAndValidateScan(ctx, conn, scanImgReq(ts.ocpInternalImage.TagRef(), withClusterFlag))
	})

	// A user delegating a scan to a Secured Cluster must have access to the namespace
	// in order to pull the namespace specific secrets needed to scan an image from
	// the internal OCP image registry. This test ensures that scans fail when the user
	// does not have namespace access (per the access scope).
	ts.Run("fail to scan image from OCP internal registry if user has no namespace access", func() {
		t := ts.T()

		ts.skipIfNotOpenShift()

		ps := &storage.PermissionSet{
			Name: deleScanPermissionSetName,
			ResourceToAccess: map[string]storage.Access{
				"Image": storage.Access_READ_WRITE_ACCESS,
			},
		}

		role := &storage.Role{
			Name:          deleScanRoleName,
			AccessScopeId: denyAllAccessScope,
		}

		limitedConn := ts.getLimitedCentralConn(ctx, ps, role)
		service := v1.NewImageServiceClient(limitedConn)
		_, err := ts.scanWithRetries(ctx, service, scanImgReq(ts.ocpInternalImage.TagRef(), withClusterFlag))
		require.Error(t, err, "scan should fail when user has no namespace access")
	})

	// Ensure user with minimally scoped permission is able to delegate scans
	// for images from the OCP internal registry.
	ts.Run("scan image from OCP internal registry with minimally scoped access", func() {
		ts.skipIfNotOpenShift()

		ps := &storage.PermissionSet{
			Name: deleScanPermissionSetName,
			ResourceToAccess: map[string]storage.Access{
				"Image": storage.Access_READ_WRITE_ACCESS,
			},
		}

		role := &storage.Role{
			Name:          deleScanRoleName,
			AccessScopeId: allowAllAccessScope,
		}

		limitedConn := ts.getLimitedCentralConn(ctx, ps, role)
		ts.executeAndValidateScan(ctx, limitedConn, scanImgReq(ts.ocpInternalImage.TagRef(), withClusterFlag))
	})
}

// TestDeploymentScans tests delegating image scans via observed k8s deployments.
func (ts *DelegatedScanningSuite) TestDeploymentScans() {
	t := ts.T()
	ctx := ts.ctx

	conn := centralgrpc.GRPCConnectionToCentral(t)
	service := v1.NewDelegatedRegistryConfigServiceClient(conn)

	// deployAndVerify is a utility function used by the subsequent tests
	// that creates a deployment and validates a scan was successfully completed.
	deployAndVerify := func(deployName string, image deleScanTestImage) {
		t := ts.T()

		// Do an initial teardown in case a deployment is lingering from a previous test.
		teardownDeployment(t, deployName)

		// Since we cannot 'force' a scan when deploying an image, we first delete
		// the image to help ensure a fresh scan is executed. Otherwise Sensor
		// may pull the image from the scan cache.
		ts.deleteImageByID(image.ID())

		fromByte := ts.getSensorLastLogBytePos(ctx)

		// Create a deployment that will sleep forever.
		ts.deleScanUtils.DeploySleeperImage(t, ctx, ts.namespace, deployName, image.IDRef())

		// Pull the image scan from the API and validate it.
		imageService := v1.NewImageServiceClient(conn)
		img, err := ts.getImageWithRetries(ctx, imageService, &v1.GetImageRequest{Id: image.ID()})
		require.NoError(t, err)
		ts.validateImageScan(t, image.IDRef(), img)

		// Verify that a fresh scan was executed per Sensor logs.
		ts.waitUntilSensorLogsScan(ctx, image.IDRef(), fromByte)

		// Only perform teardown on success so that logs can be captured on failure.
		logf(t, "Tearing down deployment %q", deployName)
		teardownDeploymentWithoutCheck(t, deployName)
	}

	ts.Run("scan deployed image", func() {
		t := ts.T()

		// Update the delegated scanning config to delegate scans for all images.
		_, err := service.UpdateConfig(ctx, &v1.DelegatedRegistryConfig{
			EnabledFor: v1.DelegatedRegistryConfig_ALL,
		})
		require.NoError(t, err)

		deployAndVerify("dele-scan-norm", ts.ubi9ImageB)
	})

	// The test verifies that with delegated scanning disabled images from the OCP internal
	// registry are still scanned.
	ts.Run("scan deployed image from OCP internal registry", func() {
		ctx := ts.ctx

		ts.skipIfNotOpenShift()

		// Disable delegated scanning.
		ts.resetConfig(ctx)

		deployAndVerify("dele-scan-ocp-int", ts.ocpInternalImage)
	})

}

// TestMirrorScans will setup mirroring on a cluster and validate
// that delegated scanning is able to scan images from mirrors defined by the various
// mirroring CRs (ie: ImageContentSourcePolicy, ImageDigestMirrorSet, ImageTagMirrorSet)
func (ts *DelegatedScanningSuite) TestMirrorScans() {
	t := ts.T()
	ctx := ts.ctx

	ts.skipIfNotOpenShift()

	// Before setting up mirrors, attempt to disable the node draining behavior of the
	// OCP Machine Config Operator.
	nodesDrained := false
	err := ts.deleScanUtils.DisableMCONodeDrain(t, ctx)
	if err != nil {
		logf(t, "WARN: Attempts to disable machine config operator node draining behavior failed, this may lead to higher chance of flakes: %v", err)
		nodesDrained = true
	}

	// Create mirroring CRs and update OCP global pull secret, this will
	// trigger nodes to drain and may take between 5-10 mins to complete.
	icspSupported, idmsSupported, itmsSupported := ts.deleScanUtils.SetupMirrors(t, ctx, "quay.io/rhacs-eng", config.DockerConfigEntry{
		Username: ts.quayROUsername,
		Password: ts.quayROPassword,
		Email:    "dele-scan-test@example.com",
	})

	if !icspSupported && !idmsSupported && !itmsSupported {
		t.Skip("Mirroring CRs not supported in this cluster, skipping tests")
	}

	if nodesDrained {
		// Sensor connects to Central quicker on fresh start vs. waiting for automatic reconnect.
		// Since Sensor may have started first after the prior node drain, we restart Sensor
		// so that testing will be able to proceed quicker.
		logf(t, "Deleting Sensor to speed up ready state")
		sensorPod, err := ts.getSensorPodWithRetries(ctx, ts.namespace)
		require.NoError(t, err)
		err = ts.k8s.CoreV1().Pods(ts.namespace).Delete(ctx, sensorPod.GetName(), metaV1.DeleteOptions{})
		require.NoError(t, err)
	}

	// Wait for Central/Sensor to be healthy.
	ts.waitForHealthyCentralSensorConn()

	// The mirroring CRs map to quay.io, these tests will attempt to scan images from quay.io/rhacs-eng.
	// To ensure authenticate succeeds we create an image integration. Creating this integration
	// will not be necessary in the future if/when ROX-25709 is implemented, which would cause
	// Sensor to match secrets based on path in addition to registry host. The global pull secret
	// was changed to use the /rhacs-eng path which is currently truncated by Sensor, as a result
	// it is indeterminate if the quay.io or quay.io/rhacs-eng secret will be stored by Sensor.
	ts.createImageIntegration(&storage.DockerConfig{
		Endpoint: "quay.io",
		Username: ts.quayROUsername,
		Password: ts.quayROPassword,
	}, false, true)

	logf(t, "Enabling delegated scanning for the mirror hosts")
	err = ts.updateConfigWithRetries(ctx, &v1.DelegatedRegistryConfig{
		EnabledFor: v1.DelegatedRegistryConfig_SPECIFIC,
		Registries: []*v1.DelegatedRegistryConfig_DelegatedRegistry{
			// These paths are defined via the mirroring CRs in testdata/delegatedscanning/mirrors/*.
			{Path: "icsp.invalid"},
			{Path: "idms.invalid"},
			{Path: "itms.invalid"},
		},
	})
	require.NoError(t, err)

	icspImage := ts.nginxImage.WithReg("icsp.invalid")
	idmsImage := ts.httpdImage.WithReg("idms.invalid")
	itmsImage := ts.memcachedImage.WithReg("itms.invalid")

	adhocTCs := []struct {
		desc     string
		imageStr string
		skip     bool
	}{
		{"Scan ad-hoc image from mirror via ImageContentSourcePolicy", icspImage.IDRef(), !icspSupported},
		{"Scan ad-hoc image from mirror via ImageDigestMirrorSet", idmsImage.IDRef(), !idmsSupported},
		{"Scan ad-hoc image from mirror via ImageTagMirrorSet", itmsImage.TagRef(), !itmsSupported},
	}

	conn := centralgrpc.GRPCConnectionToCentral(t)
	for _, tc := range adhocTCs {
		ts.Run(tc.desc, func() {
			t := ts.T()

			if tc.skip {
				t.Skip("CR not supported, skipping test.")
			}

			req := &v1.ScanImageRequest{
				ImageName: tc.imageStr,
				Force:     true,
				Cluster:   ts.remoteCluster.GetId(),
			}

			ts.executeAndValidateScan(ctx, conn, req)
		})
	}

	deployTCs := []struct {
		desc       string
		deployName string
		imgID      string
		imageStr   string
		skip       bool
	}{
		{"Scan deploy image from mirror via ImageContentSourcePolicy", "dele-scan-icsp", icspImage.ID(), icspImage.IDRef(), !icspSupported},
		{"Scan deploy image from mirror via ImageDigestMirrorSet", "dele-scan-idms", idmsImage.ID(), idmsImage.IDRef(), !idmsSupported},
		{"Scan deploy image from mirror via ImageTagMirrorSet", "dele-scan-itms", itmsImage.ID(), itmsImage.TagRef(), !itmsSupported},
	}

	for _, tc := range deployTCs {
		ts.Run(tc.desc, func() {
			t := ts.T()

			if tc.skip {
				t.Skip("CR not supported, skipping test.")
			}

			// Do an initial teardown in case a deployment is lingering from a previous test.
			teardownDeployment(t, tc.deployName)

			// Because we cannot 'force' a scan for deployments, we explicitly delete the image
			// so that it is removed from Sensor scan cache.
			imageService := v1.NewImageServiceClient(conn)
			query := fmt.Sprintf("Image Sha:%s", tc.imgID)
			delResp, err := imageService.DeleteImages(ctx, &v1.DeleteImagesRequest{Query: &v1.RawQuery{Query: query}, Confirm: true})
			require.NoError(t, err)
			logf(t, "Num images deleted from query %q: %d", query, delResp.NumDeleted)

			fromByte := ts.getSensorLastLogBytePos(ctx)

			// Create deployment.
			logf(t, "Creating deployment %q with image: %q", tc.deployName, tc.imageStr)
			setupDeploymentNoWait(t, tc.imageStr, tc.deployName, 1)

			img, err := ts.getImageWithRetries(ctx, imageService, &v1.GetImageRequest{Id: tc.imgID})
			require.NoError(t, err)
			ts.validateImageScan(t, tc.imageStr, img)

			ts.waitUntilSensorLogsScan(ctx, tc.imageStr, fromByte)

			// Only perform teardown on success so that logs can be captured on failure.
			logf(t, "Tearing down deployment %q", tc.deployName)
			teardownDeployment(t, tc.deployName)
		})
	}
}

// validateImageScan will fail the test if the image's scan was not completed
// successfully, assumes that the image has at least one vulnerability.
func (ts *DelegatedScanningSuite) validateImageScan(t *testing.T, imgFullName string, img *storage.Image) {
	require.Equal(t, imgFullName, img.GetName().GetFullName())
	require.True(t, img.GetIsClusterLocal(), "image %q not flagged as cluster local which is expected for any delegated scans, most likely the scan was NOT delegated, check Central/Sensor logs to confirm", imgFullName)
	require.NotNil(t, img.GetScan(), "image scan for %q is nil, check logs for scan errors, image notes: %v", imgFullName, img.GetNotes())
	require.NotEmpty(t, img.GetScan().GetComponents(), "image scan for %q has no components, check central logs for scan errors, this can happen if indexing succeeds but matching fails, ROX-17472 will make this an error in the future", imgFullName)

	// Ensure at least one component has a vulnerability.
	for _, c := range img.GetScan().GetComponents() {
		if len(c.GetVulns()) > 0 {
			logf(t, "Found successful scan of %q", imgFullName)
			return
		}
	}

	require.Fail(t, "No vulnerabilities found.", "Expected at least one vulnerability in image %q, but found none.", imgFullName)
}

// getImageWithRetries will get an image from the StackRox API, retrying when not found.
func (ts *DelegatedScanningSuite) getImageWithRetries(ctx context.Context, service v1.ImageServiceClient, req *v1.GetImageRequest) (*storage.Image, error) {
	var err error
	var img *storage.Image

	retryFunc := func() error {
		img, err = service.GetImage(ctx, req)
		if err != nil {
			s, ok := status.FromError(err)
			if ok && s.Code() == codes.NotFound {
				return retry.MakeRetryable(err)
			}
		}

		return err
	}

	err = ts.withRetries(retryFunc, "Image not found")

	return img, err
}

// scanWithRetries will scan an image using the StackRox API, retrying when rate limited.
func (ts *DelegatedScanningSuite) scanWithRetries(ctx context.Context, service v1.ImageServiceClient, req *v1.ScanImageRequest) (*storage.Image, error) {
	var err error
	var img *storage.Image

	retryErrTokens := []string{
		scan.ErrTooManyParallelScans.Error(),
		"context deadline exceeded",
		"Client.Timeout exceeded while awaiting headers",

		// K8s services/pods may refuse connections shortly after restart
		//
		// ex:
		// - transport: Error while dialing: dial tcp <ip>:8443: connect: connection refused
		"connect: connection refused",

		// Registry issues, network glitches, resources contention, etc. may interrupt the download
		// of image layers.
		//
		// ex:
		// - could not advance in the tar archive: archive/tar: invalid tar header
		// - could not advance in the tar archive: unexpected EOF
		"could not advance in the tar archive",

		// Sensor may accept ad-hoc scan requests prior to the mirroring CRs being loaded, this may result
		// in attempts to reach out to the 'invalid' mirror host. We trigger a retry in this case to give
		// Sensor time to load the mirroring CRs.
		//
		// ex:
		// - unable to check TLS for registry "icsp.invalid": dial tcp: lookup icsp.invalid on <ip>:53: no such host
		"no such host",

		// Central's cluster API is used to report the health of secured clusters, this cluster status is on a delay
		// and may not represent actual state leading to flakes. When the actual connection to a cluster fails during
		// delegation, the scan attempt should be retried.
		//
		// ex:
		// - no connection to "a21b168a-280e-40d1-a175-e84d14ed8232"
		"no connection to",
	}

	retryFunc := func() error {
		img, err = service.ScanImage(ctx, req)
		if err == nil {
			return nil
		}

		for _, token := range retryErrTokens {
			if strings.Contains(err.Error(), token) {
				return retry.MakeRetryable(err)
			}
		}

		return err
	}

	err = ts.withRetries(retryFunc, "Scan failed")
	return img, err
}

// waitUntilLog is a custom wrapper for the delegated scanning tests around the common waitUntilLog method.
// This assumes Sensor is the log being read. A timeout is applied to the wait to be a good citizen
// for other e2e tests (by not consuming the full go test timeout).
func (ts *DelegatedScanningSuite) waitUntilLog(ctx context.Context, description string, logMatchers ...logMatcher) {
	ctx, cancel := context.WithTimeout(ctx, deleScanDefaultLogWaitTimeout)
	defer cancel()
	ts.KubernetesSuite.waitUntilLog(ctx, ts.namespace, sensorPodLabels, sensorContainer, description, logMatchers...)
}

// checkLogsMatch is a custom wrapper for the delegated scanning tests around the common checkLogsMatch method.
// This assumes Sensor is the log being read.
func (ts *DelegatedScanningSuite) checkLogsMatch(ctx context.Context, description string, logMatchers ...logMatcher) {
	ts.KubernetesSuite.checkLogsMatch(ctx, ts.namespace, sensorPodLabels, sensorContainer, description, logMatchers...)
}

// getSensorLastLogBytePos gets the last byte from the Sensor logs.  Used when search logs so that only 'new' lines
// are matched.
func (ts *DelegatedScanningSuite) getSensorLastLogBytePos(ctx context.Context) int64 {
	t := ts.T()
	sensorPod, err := ts.getSensorPodWithRetries(ctx, ts.namespace)
	require.NoError(t, err)

	fromByte, err := ts.getLastLogBytePos(ctx, ts.namespace, sensorPod.GetName(), sensorPod.Spec.Containers[0].Name)
	require.NoError(t, err)

	return fromByte
}

// resetConfig resets the delegated scanning config to default value.
func (ts *DelegatedScanningSuite) resetConfig(ctx context.Context) {
	t := ts.T()

	err := ts.updateConfigWithRetries(ctx, &v1.DelegatedRegistryConfig{})
	require.NoError(t, err)
}

// setConfig sets the delegated scanning config to delegate scans for
// the registry/remote of the provided image.
func (ts *DelegatedScanningSuite) setConfig(ctx context.Context, image deleScanTestImage) {
	t := ts.T()

	cfg := &v1.DelegatedRegistryConfig{
		EnabledFor:       v1.DelegatedRegistryConfig_SPECIFIC,
		DefaultClusterId: ts.remoteCluster.GetId(),
		Registries: []*v1.DelegatedRegistryConfig_DelegatedRegistry{
			{Path: image.Base()},
		},
	}

	err := ts.updateConfigWithRetries(ctx, cfg)
	require.NoError(t, err)
}

// updateConfigWithRetries will update the delegated registry config in Central
// retrying on connection refused error. This was added to reduce flakes in
// CI.
func (ts *DelegatedScanningSuite) updateConfigWithRetries(ctx context.Context, cfg *v1.DelegatedRegistryConfig) error {
	t := ts.T()

	conn := centralgrpc.GRPCConnectionToCentral(t)
	service := v1.NewDelegatedRegistryConfigServiceClient(conn)

	retryFunc := func() error {
		_, err := service.UpdateConfig(ctx, cfg)
		return ts.retryConnectionRefused(err)
	}

	return ts.withRetries(retryFunc, "Connection refused by Central attempting to update the delegated registry config")
}

// resetAccess will revoke API tokens, delete roles, and delete permissions sets
// that have been created by this suite.
func (ts *DelegatedScanningSuite) resetAccess(ctx context.Context) {
	t := ts.T()

	revokeAPIToken(t, ctx, deleScanAPITokenName)
	deleteRole(t, ctx, deleScanRoleName)
	deletePermissionSet(t, ctx, deleScanPermissionSetName)
}

// waitUntilSensorLogsScan will wait until Sensor logs show a successful scan has been completed for the
// provided image.
func (ts *DelegatedScanningSuite) waitUntilSensorLogsScan(ctx context.Context, imageStr string, fromByte int64) {
	t := ts.T()

	reStr := fmt.Sprintf(`Image "%s".* enriched with metadata using pull source`, imageStr)
	regexp.MustCompile(reStr)

	ts.waitUntilLog(ctx, "contain the image scan",
		containsLineMatchingAfter(regexp.MustCompile(reStr), fromByte),
	)

	logf(t, "Found Sensor log entries indiciating successful scan of %q after byte %d", imageStr, fromByte)
}

// getLimitedCentralConn will return a connection to central using a token defined by the provided
// permission set, role, and access scope. The role.permissionSetId provided will be overridden
// with the ID of the permission set provided by the StackRox API.
func (ts *DelegatedScanningSuite) getLimitedCentralConn(ctx context.Context, permissionSet *storage.PermissionSet, role *storage.Role) *grpc.ClientConn {
	t := ts.T()

	ts.resetAccess(ctx)

	// Create a new permission set.
	permissionSetID := mustCreatePermissionSet(t, ctx, permissionSet)

	// Create the role for the new permission set.
	role.PermissionSetId = permissionSetID
	mustCreateRole(t, ctx, role)

	// Create a token bound to the new role.
	_, token := mustCreateAPIToken(t, ctx, deleScanAPITokenName, []string{deleScanRoleName})

	// Connect to central using the new token.
	conn := centralgrpc.GRPCConnectionToCentral(t, func(opts *clientconn.Options) {
		opts.ConfigureTokenAuth(token)
	})

	return conn
}

// executeAndValidateScan will trigger a scan via the StackRox API, validate the scan was successful,
// and confirm the scan was executed in the Secured Cluster.
func (ts *DelegatedScanningSuite) executeAndValidateScan(ctx context.Context, conn *grpc.ClientConn, req *v1.ScanImageRequest) {
	t := ts.T()

	// Get the last byte in the Sensor log, so that we only search for the
	// logs AFTER that byte after we execute the scan.  This allows us to
	// re-use the same image in various tests and should also speed up
	// the search.
	fromByte := ts.getSensorLastLogBytePos(ctx)

	// Scan the image via the image service, retrying as needed.
	service := v1.NewImageServiceClient(conn)
	img, err := ts.scanWithRetries(ctx, service, req)
	require.NoError(t, err)

	// Validate that the image scan looks 'OK'
	imageStr := req.GetImageName()
	ts.validateImageScan(t, imageStr, img)

	// Search the Sensor logs for a record of the scan that was just completed
	ts.waitUntilSensorLogsScan(ctx, imageStr, fromByte)
}

// skipIfNotOpenShift will skip the current test if the cluster under test
// is NOT openshift.
func (ts *DelegatedScanningSuite) skipIfNotOpenShift() {
	t := ts.T()

	if !isOpenshift() {
		t.Skip("Skipping test - not an OCP cluster")
	}
}

// createImageIntegration will create an image integration (and also clean it up when test ends).
func (ts *DelegatedScanningSuite) createImageIntegration(dockerConfig *storage.DockerConfig, autogenerated bool, cleanup bool) *storage.ImageIntegration {
	t := ts.T()
	ctx := ts.ctx

	conn := centralgrpc.GRPCConnectionToCentral(t)
	iiService := v1.NewImageIntegrationServiceClient(conn)
	ii := &storage.ImageIntegration{
		Name: fmt.Sprintf("dele-scan-test-%s", uuid.NewV4().String()),
		Type: types.DockerType,
		IntegrationConfig: &storage.ImageIntegration_Docker{
			Docker: dockerConfig,
		},
		SkipTestIntegration: true,
		Categories: []storage.ImageIntegrationCategory{
			storage.ImageIntegrationCategory_REGISTRY,
		},
		Autogenerated: autogenerated,
	}

	var err error
	var rii *storage.ImageIntegration

	retryFunc := func() error {
		rii, err = iiService.PostImageIntegration(ctx, ii)
		return ts.retryConnectionRefused(err)
	}

	err = ts.withRetries(retryFunc, "Connection refused by Central attempting to create image integration")
	require.NoError(t, err)

	if cleanup {
		t.Cleanup(func() { _, _ = iiService.DeleteImageIntegration(ctx, &v1.ResourceByID{Id: rii.GetId()}) })
	}

	return rii
}

// deleteImageByID will delete an image via the StackRox API. Note that if a deployment exists
// that references the image, the image will be immediately re-scanned upon delete.  To avoid
// this ensure there are no deployments referencing the image under test.
func (ts *DelegatedScanningSuite) deleteImageByID(id string) {
	t := ts.T()
	ctx := ts.ctx

	conn := centralgrpc.GRPCConnectionToCentral(t)
	imageService := v1.NewImageServiceClient(conn)

	query := fmt.Sprintf("Image Sha:%s", id)

	delResp, err := imageService.DeleteImages(ctx, &v1.DeleteImagesRequest{Query: &v1.RawQuery{Query: query}, Confirm: true})
	require.NoError(t, err)

	logf(t, "Num images deleted from query %q: %d", query, delResp.NumDeleted)
}

// waitForHealthyCentralSensorConn will wait for the Sensor deployment to be ready
// and for Central to report a healthy connection to Sensor.
func (ts *DelegatedScanningSuite) waitForHealthyCentralSensorConn() {
	t := ts.T()
	ctx := ts.ctx

	// Wait for critical components to be healthy.
	logf(t, "Waiting for Sensor to be ready")
	ts.waitUntilK8sDeploymentReady(ctx, ts.namespace, sensorDeployment)

	logf(t, "Waiting for Central/Sensor connection to be ready")
	waitUntilCentralSensorConnectionIs(t, ctx, storage.ClusterHealthStatus_HEALTHY)
}

// getSensorPodWithRetries will retry calls to getSensorPod when more than one pod
// is detected.
func (ts *DelegatedScanningSuite) getSensorPodWithRetries(ctx context.Context, namespace string) (*coreV1.Pod, error) {
	var err error
	var pod *coreV1.Pod

	retryFunc := func() error {
		pod, err = ts.getSensorPod(ctx, namespace)

		if err != nil && strings.Contains(err.Error(), "more than one") {
			err = retry.MakeRetryable(err)
		}

		return err
	}

	err = ts.withRetries(retryFunc, "Found more then one sensor pod")
	return pod, err
}

// withRetries will execute retryFunc and retry execution when retryFunc marks the returned
// error as retriable, statusMsg will be printed between attempts.
func (ts *DelegatedScanningSuite) withRetries(retryFunc func() error, statusMsg string) error {
	t := ts.T()

	betweenAttemptsFunc := func(num int) {
		logf(t, "Trying again in %s, attempt %d/%d", deleScanDefaultRetryDelay, num, deleScanDefaultMaxRetries)
		time.Sleep(deleScanDefaultRetryDelay)
	}

	onFailedAttemptsFunc := func(err error) {
		// Log the error for each attempt to assist troubleshooting.
		logf(t, "%s: %v", statusMsg, err)
	}

	return retry.WithRetry(retryFunc,
		retry.BetweenAttempts(betweenAttemptsFunc),
		retry.Tries(deleScanDefaultMaxRetries),
		retry.WithExponentialBackoff(),
		retry.OnlyRetryableErrors(),
		retry.OnFailedAttempts(onFailedAttemptsFunc),
	)
}

// retryConnectionRefused will return err wrapped in a retryable if it is
// a connection refused error.
func (ts *DelegatedScanningSuite) retryConnectionRefused(err error) error {
	if err == nil {
		return err
	}

	if strings.Contains(err.Error(), "connection refused") {
		return retry.MakeRetryable(err)
	}

	return err
}
