//go:build test_e2e

package tests

import (
	"context"
	"fmt"
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
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/ternary"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common/scan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
// - API_ENDPOINT               - location of the stackrox API.
// - ROX_USERNAME               - for admin access to the stackrox API.
// - ROX_PASSWORD               - for admin access to the stackrox API.
// - KUBECONFIG                 - for inspecting logs, creating deploys, etc.
// - QUAY_RHACS_ENG_RO_USERNAME - for reading quay.io/rhacs-eng images.
// - QUAY_RHACS_ENG_RO_PASSWORD - for reading quay.io/rhacs-eng images.
// - ORCHESTRATOR_FLAVOR        - to determine if running on OCP or not.

const (
	// deleScanLogLevelEnvVar is the name of Sensor's env var that controls the logging level.
	deleScanLogLevelEnvVar = "LOGLEVEL"

	// deleScanDesiredLogLevel is the desired value of Sensor's log level env var for executing these tests.
	deleScanDesiredLogLevel = "Debug"

	// deleScanDefaultLogWaitTimeout is the amount of time that the various delegated scanning tests will wait for
	// a log entry to be found.
	deleScanDefaultLogWaitTimeout = 30 * time.Second

	// deleScanDefaultMaxRetries the number of times to retry pulling images from the stackrox API or executing
	// scans that are rate limited.
	deleScanDefaultMaxRetries = 30

	// deleScanDefaultRetryDelay the amount of time to wait between retries interacting with the stackrox API.
	deleScanDefaultRetryDelay = 5 * time.Second
)

// This block contains names for resources created during testing, consistent names
// make cleanup easier.
const (
	deleScanAPITokenName      = "dele-scan-api-token" //nolint:gosec // G101
	deleScanAccessScopeName   = "dele-scan-access-scope"
	deleScanRoleName          = "dele-scan-role"
	deleScanPermissionSetName = "dele-scan-permission-set"
)

var (
	// deleScanTestImageStr references an image that is small (scans fast), does not require auth, and
	// has at least one vulnerability.  This image is used by majority of the scanning tests.
	deleScanTestImageStr = "registry.access.redhat.com/ubi9/ubi-minimal:9.4-1194"

	denyAllAccessScope  = accesscontrol.DefaultAccessScopeIDs[accesscontrol.DenyAllAccessScope]
	allowAllAccessScope = accesscontrol.DefaultAccessScopeIDs[accesscontrol.UnrestrictedAccessScope]
)

type DelegatedScanningSuite struct {
	KubernetesSuite
	ctx        context.Context
	cleanupCtx context.Context
	cancel     func()

	// origSensorLogLevel is the value of Sensor's log level env var prior to any changes from this suite.
	origSensorLogLevel string

	// remoteCluster represents the Secured Cluster that scans will be delegated to.
	remoteCluster *v1.DelegatedRegistryCluster

	// namespace holds the current k8s namespace that StackRox is installed in and where deployments
	// for testing will be created.
	namespace string

	// restCfg is used by various client-go methods and is setup to point to the k8s cluster under test.
	restCfg *rest.Config

	// These hold the credentials for accessing images from quay.io/rhacs-eng
	quayROUsername string
	quayROPassword string

	// failureHandled is used to ensure the cleanup activities are not executed multiple times by the suite.
	failureHandled atomic.Bool

	// ocpInternalImageStr is the name of the image that was created in the OCP internal registry during setup.
	ocpInternalImageStr string

	// deleScanUtils is a reference to a utility library used to setup variuos aspects of these tests, such as
	// creating images in the OCP image registry or creating mirroring CRs.
	deleScanUtils *deleScanTestUtils
}

// TestDelegatedScanning the entrypoint for all delegated scanning tests.
func TestDelegatedScanning(t *testing.T) {
	suite.Run(t, new(DelegatedScanningSuite))
}

func (ts *DelegatedScanningSuite) SetupSuite() {
	// Ensure failures during setup are handled.
	ts.T().Cleanup(ts.handleFailure)

	ts.KubernetesSuite.SetupSuite()
	ts.namespace = namespaces.StackRox

	t := ts.T()
	ts.ctx, ts.cleanupCtx, ts.cancel = testContexts(t, "TestDelegatedScanning", 30*time.Minute)
	ts.restCfg = getConfig(t)

	ctx := ts.ctx

	// Get a reference to the Secured Cluster to send delegated scans too, will use this reference throughout the tests.
	// If a valid remote cluster is NOT available all tests in this suite will fail.
	t.Log("Getting remote stackrox cluster details")
	envVal, _ := ts.getDeploymentEnvVal(ctx, ts.namespace, sensorDeployment, sensorContainer, env.LocalImageScanningEnabled.EnvVar())

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

	// Ensure rhacs-eng repo user/pass is avail.
	ts.quayROUsername = mustGetEnv(t, "QUAY_RHACS_ENG_RO_USERNAME")
	ts.quayROPassword = mustGetEnv(t, "QUAY_RHACS_ENG_RO_PASSWORD")

	// Enable Sensor debug logs, some tests need this to accurately validate expected behaviors.
	ts.origSensorLogLevel, _ = ts.getDeploymentEnvVal(ctx, ts.namespace, sensorDeployment, sensorContainer, deleScanLogLevelEnvVar)
	if ts.origSensorLogLevel != deleScanDesiredLogLevel {
		ts.mustSetDeploymentEnvVal(ctx, ts.namespace, sensorDeployment, sensorContainer, deleScanLogLevelEnvVar, deleScanDesiredLogLevel)
		t.Logf("Log level env var changed from %q to %q on Sensor", ts.origSensorLogLevel, deleScanDesiredLogLevel)
	}

	// Wait for critical components to be healthy.
	t.Log("Waiting for Sensor to be ready")
	ts.waitUntilK8sDeploymentReady(ctx, ts.namespace, sensorDeployment)
	t.Log("Waiting for Central/Sensor connection to be ready")
	waitUntilCentralSensorConnectionIs(t, ctx, storage.ClusterHealthStatus_HEALTHY)

	// Pre-clean any roles, permissions, etc. from previous runs.
	t.Log("Deleteing resources from previous tests")
	ts.resetAccess(ctx)

	t.Log("Resetting delegated scanning config")
	ts.resetConfig(ctx)

	ts.deleScanUtils = NewDeleScanTestUtils(t, ts.restCfg, ts.listK8sAPIResources())
	if isOpenshift() {
		t.Log("Building and pushing an image to the OCP internal registry")
		ts.ocpInternalImageStr = ts.deleScanUtils.BuildOCPInternalImage(t, ts.ctx, ts.namespace, "dele-scan-test-01", deleScanTestImageStr)
	}
}

func (ts *DelegatedScanningSuite) TearDownSuite() {
	t := ts.T()
	ctx := ts.cleanupCtx

	// Handle data colletcion on test failures prior to resetting the log level.
	// Changing the log level may result in pod restarts and logs lost.
	ts.handleFailure()

	// Reset the log level back to its original value so that other e2e tests are
	// not impacted by the additional logging.
	if ts.origSensorLogLevel != deleScanDesiredLogLevel {
		if ts.origSensorLogLevel != "" {
			ts.mustSetDeploymentEnvVal(ctx, ts.namespace, sensorDeployment, sensorContainer, deleScanLogLevelEnvVar, ts.origSensorLogLevel)
			t.Logf("Log level reverted back to %q on Sensor", ts.origSensorLogLevel)
		} else {
			ts.mustDeleteDeploymentEnvVar(ctx, ts.namespace, sensorDeployment, deleScanLogLevelEnvVar)
			t.Logf("Log level env var removed from Sensor")
		}
	}

	ts.cancel()
}

// handleFailure is a catch all for handling test suite failures, invoked via t.Cleanup in SetupSuite AND
// as part of TearDownSuite. We cannot handle failures solely in TearDownSuite because a failure in SetupSuite
// prevents TearDownSuite from executing. Subsequent invocations of this method after the first will be no-ops.
//
// TODO: move log captures to after individual test, since each test may
// make changes that result in logs lost (such as pod restarts).
func (ts *DelegatedScanningSuite) handleFailure() {
	if ts.failureHandled.Swap(true) {
		ts.T().Log("Failure already handled")
		return
	}

	t := ts.T()
	if t.Failed() {
		ts.logf("Test failed. Collecting logs before final cleanup.")
		collectLogs(t, ts.namespace, "delegated-scanning-failure")
	}
}

// TestConfig verifies that changes made to the delegated registry config
// stick and are propagated to the Secured Clusters.
func (ts *DelegatedScanningSuite) TestConfig() {
	t := ts.T()
	ctx := ts.ctx

	conn := centralgrpc.GRPCConnectionToCentral(t)
	service := v1.NewDelegatedRegistryConfigServiceClient(conn)

	// Ensure that at least one valid Secured Cluster exists, which is required for
	// delegating scans, otherwise fail.
	getClustersResp, err := service.GetClusters(ctx, nil)
	require.NoError(t, err)
	require.Greater(t, len(getClustersResp.Clusters), 0)

	cluster := getClustersResp.Clusters[0]
	require.NotEmpty(t, cluster.GetId())
	require.NotEmpty(t, cluster.GetName())

	// Verify the API returns the expected values when the config
	// is set to it's default values.
	ts.Run("config has expected default values", func() {
		t := ts.T()
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
	// (as opposed to in-mem). This test verifies that the config
	// is persisted to the DB and read back accurately.
	ts.Run("update config and get same values back", func() {
		t := ts.T()

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
	// that the change is propagated to the Secured Cluster.
	ts.Run("config propagated to secured cluster", func() {
		t := ts.T()
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

		fromByte := ts.getSensorLastLogBytePos(ctx)

		_, err := service.UpdateConfig(ctx, cfg)
		require.NoError(t, err)

		ts.waitUntilLog(ctx, "contain delegated registry config upsert",
			containsLineMatchingAfter(regexp.MustCompile(fmt.Sprintf("Upserted delegated registry config.*%s", path)), fromByte),
		)
	})
}

// TestImageIntegrations tests various aspects of image integrations
// (such as syncing) related to delegated scanning.
func (ts *DelegatedScanningSuite) TestImageIntegrations() {
	t := ts.T()
	ctx := ts.ctx

	conn := centralgrpc.GRPCConnectionToCentral(t)
	service := v1.NewImageIntegrationServiceClient(conn)

	// This test verifies there are no autogenerated image integrations created for the
	// OCP internal registry. This doesn't directly test delegated scanning, however
	// the presence of these integrations is a symptom that indicates the detection of
	// OCP internal registry secrets is broken and will result in delegated scanning failures
	// (ex: regression ROX-25526).
	//
	// Scan failures are not used to detect this regression because our test environment
	// consists of a single cluster/namespace Stackrox installation (as opposed to multi-cluster).
	// In a single cluster installation scanning images from the OCP internal registry may,
	// by luck, succeed if an autogenerated integration with appropriate access
	// exists at the time of scan.
	ts.Run("no autogenerated integrations are created for the OCP internal registry", func() {
		t := ts.T()
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

	// Created, updated, and deleted image integrations should be sent to every Secured Cluster, this
	// gives users control over the credentials used for delegated scanning.
	ts.Run("image integrations are kept synced with the secured clusters", func() {
		t := ts.T()
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

		fromByte := ts.getSensorLastLogBytePos(ctx)

		// Create image integration.
		rii, err := service.PostImageIntegration(ctx, ii)
		require.NoError(t, err)
		id := rii.GetId()

		ts.waitUntilLog(ctx, "contain the upserted integration",
			// Requires debug logging.
			containsLineMatchingAfter(regexp.MustCompile(fmt.Sprintf("Upserted registry integration.*%s", id)), fromByte),
		)

		// Update image integration.
		rii.GetDocker().Insecure = !rii.GetDocker().Insecure
		_, err = service.UpdateImageIntegration(ctx, &v1.UpdateImageIntegrationRequest{Config: rii, UpdatePassword: false})
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
		t := ts.T()

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

		fromByte := ts.getSensorLastLogBytePos(ctx)

		// Create image integration.
		rii, err := service.PostImageIntegration(ctx, ii)
		require.NoError(t, err)
		id := rii.GetId()
		t.Cleanup(func() { service.DeleteImageIntegration(ctx, &v1.ResourceByID{Id: id}) })

		ts.checkLogsMatch(ctx, "contain the upserted integration",
			// Requires debug logging.
			containsNoLinesMatchingAfter(regexp.MustCompile(fmt.Sprintf("Upserted registry integration.*%s", id)), fromByte),
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

	scanImgReq := func(imgStr string, withClusterFlag bool) *v1.ScanImageRequest {
		cluster := ternary.String(withClusterFlag, ts.remoteCluster.GetId(), "")
		return &v1.ScanImageRequest{
			ImageName: imgStr,
			Force:     true,
			Cluster:   cluster,
		}
	}

	conn := centralgrpc.GRPCConnectionToCentral(t)

	// Verifies under normal conditions can successfully delegate a scan when
	// explicitly setting the destination cluster in the API request.
	ts.Run("delegate scan via cluster flag", func() {
		ts.resetConfig(ctx)

		ts.executeAndValidateScan(ctx, conn, scanImgReq(deleScanTestImageStr, withClusterFlag))
	})

	// Verifies under normal conditions can successfully delegate a scan when
	// the scan destination is set via the delegated scanning config.
	ts.Run("delegate scan via config", func() {
		ts.setConfig(ctx, deleScanTestImageStr)

		ts.executeAndValidateScan(ctx, conn, scanImgReq(deleScanTestImageStr, !withClusterFlag))
	})

	// Image write permission is required to scan images, this test uses a token with only image
	// read permissions, and therefore should fail to scan any image. This test is meant to catch
	// a past regression where a role with only image read permissions was mistakenly able to scan images.
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

		// Execute the scan.
		limitedConn := ts.getLimitedCentralConn(ctx, ps, role)
		service := v1.NewImageServiceClient(limitedConn)
		_, err := service.ScanImage(ctx, scanImgReq(deleScanTestImageStr, withClusterFlag))
		require.ErrorContains(t, err, "not authorized")
	})

	// Test scanning an image using a token with minimal permissions (ie: Image write).
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
		ts.executeAndValidateScan(ctx, limitedConn, scanImgReq(deleScanTestImageStr, withClusterFlag))
	})

	// Create an image watch that results in Secured Cluster initiating the scan.
	ts.Run("delegate scan image via image watch", func() {
		t := ts.T()

		// Set the delegated scanning config to delegate scans for the test image.
		ts.setConfig(ctx, deleScanTestImageStr)

		service := v1.NewImageServiceClient(conn)

		// Since we cannot 'force' a scan via an image watch, we first delete
		// the image from Central to help ensure a fresh scan is executed.
		query := fmt.Sprintf("Image:%s", deleScanTestImageStr)
		delResp, err := service.DeleteImages(ctx, &v1.DeleteImagesRequest{Query: &v1.RawQuery{Query: query}, Confirm: true})
		require.NoError(t, err)
		t.Logf("Num images deleted from query %q: %d", query, delResp.NumDeleted)

		fromByte := ts.getSensorLastLogBytePos(ctx)

		// Setup the image watch
		resp, err := service.WatchImage(ctx, &v1.WatchImageRequest{Name: deleScanTestImageStr})
		require.NoError(t, err)
		require.Zero(t, resp.GetErrorType(), "expected no error")
		require.Equal(t, deleScanTestImageStr, resp.GetNormalizedName())
		t.Cleanup(func() { _, _ = service.UnwatchImage(ctx, &v1.UnwatchImageRequest{Name: deleScanTestImageStr}) })

		ts.waitUntilSensorLogsScan(ctx, deleScanTestImageStr, fromByte)
	})

	// Scan an image from the OCP internal registry
	ts.Run("scan image from OCP internal registry", func() {
		t := ts.T()

		if !isOpenshift() {
			t.Skip("Skipping test - not an OCP cluster")
		}

		ts.resetConfig(ctx)

		ts.executeAndValidateScan(ctx, conn, scanImgReq(ts.ocpInternalImageStr, withClusterFlag))
	})

	// A user delegating a scan to a Secured Cluster must have access to the namespace
	// in order to pull the namespace specific secrets needed to scan an image from
	// the internal OCP image registry. This test ensures that scans fail when the user
	// does not have namespace access (defined via the access scope).
	ts.Run("fail to scan image from OCP internal registry if user has no namespace access", func() {
		t := ts.T()

		if !isOpenshift() {
			t.Skip("Skipping test - not an OCP cluster")
		}

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
		_, err := ts.scanWithRetries(t, ctx, service, scanImgReq(ts.ocpInternalImageStr, withClusterFlag))
		require.Error(t, err, "scan should fail when user has no namespace access")
	})

	// Ensure user with minimally scoped permission are able to delegate scans
	// for images from the OCP internal registry.
	ts.Run("scan image from OCP internal registry with minimally scoped access", func() {
		t := ts.T()

		if !isOpenshift() {
			t.Skip("Skipping test - not an OCP cluster")
		}

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
		ts.executeAndValidateScan(ctx, limitedConn, scanImgReq(ts.ocpInternalImageStr, withClusterFlag))
	})
}

// TestDeploymentScans ...
func (ts *DelegatedScanningSuite) TestDeploymentScans() {
	ts.T().SkipNow()

	ts.Run("scan deployment for OCP internal registry", func() {
	})
	ts.Run("scan deployment all registries", func() {
	})
	ts.Run("scan deployment specific registries", func() {
	})
}

// TestMirrorScans will setup mirroring on a cluster and validate
// that delegated scanning is able to scan images from mirrors defined by the various
// mirroring CRs (ie: ImageContentSourcePolicy, ImageDigestMirrorSet, ImageTagMirrorSet)
func (ts *DelegatedScanningSuite) TestMirrorScans() {
	t := ts.T()
	if !isOpenshift() {
		t.Skip("Skipping test - not an OCP cluster")
	}
	ctx := ts.ctx

	// Create mirroring CRs and update OCP global pull secret, this will
	// trigger nodes to drain and may take between 5-10 mins to complete.
	icspAvail, idmsAvail, itmsAvail := ts.deleScanUtils.SetupMirrors(t, ctx, "quay.io/rhacs-eng", config.DockerConfigEntry{
		Username: ts.quayROUsername,
		Password: ts.quayROPassword,
		Email:    "dele-scan-test@example.com",
	})

	if !icspAvail && !idmsAvail && !itmsAvail {
		t.Skip("Mirroring CRs not available in this cluster, skipping tests")
	}

	// Sensor connects to Central quicker on fresh start vs. waiting for automatic reconnect.
	// Since Sensor may have started first after the node drain triggered above,
	// we restart Sensor so that testing will be able to proceed quicker.
	t.Log("Deleteing Sensor to speed up ready state")
	sensorPod, err := ts.getSensorPod(ctx, ts.namespace)
	require.NoError(t, err)
	err = ts.k8s.CoreV1().Pods(ts.namespace).Delete(ctx, sensorPod.GetName(), metaV1.DeleteOptions{})
	require.NoError(t, err)

	// Wait for Central/Sensor to be healthy.
	t.Log("Waiting for Sensor to be ready")
	ts.waitUntilK8sDeploymentReady(ctx, ts.namespace, sensorDeployment)
	t.Log("Waiting for Central/Sensor connection to be ready")
	waitUntilCentralSensorConnectionIs(t, ctx, storage.ClusterHealthStatus_HEALTHY)

	// The mirroring CRs map to quay.io, these tests will attempt to scan images from quay.io/rhacs-eng.
	// To ensure authenticate succeeds we create an image integration. Creating this integration
	// will not be necessary in the future if/when ROX-25709 is implemented, which would cause
	// Sensor to match secrets based on path in addition to registry host. The global pull secret
	// changes above use the /rhacs-eng path which is currently truncated by Sensor, as a result
	// it is indeterminate if the quay.io or quay.io/rhacs-eng secret will be stored by Sensor.
	conn := centralgrpc.GRPCConnectionToCentral(t)
	iiService := v1.NewImageIntegrationServiceClient(conn)
	ii := &storage.ImageIntegration{
		Name: fmt.Sprintf("dele-scan-test-%s", uuid.NewV4().String()),
		Type: types.DockerType,
		IntegrationConfig: &storage.ImageIntegration_Docker{
			Docker: &storage.DockerConfig{
				Endpoint: "quay.io",
				Username: ts.quayROUsername,
				Password: ts.quayROPassword,
			},
		},
		SkipTestIntegration: true,
		Categories: []storage.ImageIntegrationCategory{
			storage.ImageIntegrationCategory_REGISTRY,
		},
	}
	rii, err := iiService.PostImageIntegration(ctx, ii)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = iiService.DeleteImageIntegration(ctx, &v1.ResourceByID{Id: rii.GetId()}) })

	t.Logf("Enabling delegated scanning for the mirror hosts")
	deleService := v1.NewDelegatedRegistryConfigServiceClient(conn)
	_, err = deleService.UpdateConfig(ctx, &v1.DelegatedRegistryConfig{
		EnabledFor: v1.DelegatedRegistryConfig_SPECIFIC,
		Registries: []*v1.DelegatedRegistryConfig_DelegatedRegistry{
			// These paths are defined via the mirroring CRs in testdata/delegatedscanning/mirrors/*.
			{Path: "icsp.invalid"},
			{Path: "idms.invalid"},
			{Path: "itms.invalid"},
		},
	})
	require.NoError(t, err)

	adhocTCs := []struct {
		desc   string
		imgStr string
		skip   bool
	}{
		{
			"Scan ad-hoc image from mirror defined by ImageContentSourcePolicy",
			// Mirrors to: quay.io/rhacs-eng/qa:dele-scan-nginx
			"icsp.invalid/rhacs-eng/qa@sha256:68b418b74715000e41a894428bd787442945592486a08d4cbea89a9b4fa03302",
			!icspAvail,
		},
		{
			"Scan ad-hoc image from mirror defined by ImageDigestMirrorSet",
			// Mirrors to: quay.io/rhacs-eng/qa:dele-scan-httpd
			"idms.invalid/rhacs-eng/qa@sha256:489576ec07d6d8d64690bedb4cf1eeb366a8f03f8530367c3eee0c71579b5f5e",
			!idmsAvail,
		},
		{
			"Scan ad-hoc image from mirror defined by ImageTagMirrorSet",
			"itms.invalid/rhacs-eng/qa:dele-scan-memcached",
			!itmsAvail,
		},
	}
	for _, tc := range adhocTCs {
		ts.Run(tc.desc, func() {
			t := ts.T()

			if tc.skip {
				t.Skip("CR not avail, skipping test.")
			}

			req := &v1.ScanImageRequest{
				ImageName: tc.imgStr,
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
		imgStr     string
		skip       bool
	}{
		{
			"Scan deployment image from mirror defined by ImageContentSourcePolicy",
			"dele-scan-icsp",
			// Mirrors to: quay.io/rhacs-eng/qa:dele-scan-nginx
			"sha256:68b418b74715000e41a894428bd787442945592486a08d4cbea89a9b4fa03302",
			"icsp.invalid/rhacs-eng/qa@sha256:68b418b74715000e41a894428bd787442945592486a08d4cbea89a9b4fa03302",
			!icspAvail,
		},
		{
			"Scan deployment image from mirror defined by ImageDigestMirrorSet",
			"dele-scan-idms",
			// Mirrors to: quay.io/rhacs-eng/qa:dele-scan-httpd
			"sha256:489576ec07d6d8d64690bedb4cf1eeb366a8f03f8530367c3eee0c71579b5f5e",
			"idms.invalid/rhacs-eng/qa@sha256:489576ec07d6d8d64690bedb4cf1eeb366a8f03f8530367c3eee0c71579b5f5e",
			!idmsAvail,
		},
		{
			"Scan deployment image from mirror defined by ImageTagMirrorSet",
			"dele-scan-itms",
			"sha256:1cf25340014838bef90aa9d19eaef725a0b4986af3c8e8a6be3203c2cef8cb61",
			"itms.invalid/rhacs-eng/qa:dele-scan-memcached",
			!itmsAvail,
		},
	}

	for _, tc := range deployTCs {
		ts.Run(tc.desc, func() {
			t := ts.T()

			if tc.skip {
				t.Skip("CR not avail, skipping test.")
			}

			fromByte := ts.getSensorLastLogBytePos(ctx)

			// Do an initial teardown in case a deployment is lingering from a previous test.
			teardownDeployment(t, tc.deployName)

			// Because we cannot 'force' a scan for deployments, we explicitly delete the image
			// so that it is removed from Sensor scan cache.
			imageService := v1.NewImageServiceClient(conn)
			query := fmt.Sprintf("Image Sha:%s", tc.imgID)
			delResp, err := imageService.DeleteImages(ctx, &v1.DeleteImagesRequest{Query: &v1.RawQuery{Query: query}, Confirm: true})
			require.NoError(t, err)
			t.Logf("Num images deleted from query %q: %d", query, delResp.NumDeleted)

			// Create deployment.
			t.Logf("Creating deployment %q with image: %q", tc.deployName, tc.imgStr)
			setupDeploymentNoWait(t, tc.imgStr, tc.deployName, 1)

			img, err := ts.getImageWithRetires(t, ctx, imageService, &v1.GetImageRequest{Id: tc.imgID})
			require.NoError(t, err)
			ts.validateImageScan(t, tc.imgStr, img)

			ts.waitUntilSensorLogsScan(ctx, tc.imgStr, fromByte)

			// Only perform teardown on success so that logs can be captured on failure.
			t.Logf("Tearing down deployment %q", tc.deployName)
			teardownDeployment(t, tc.deployName)
		})
	}
}

// validateImageScan will fail the test if the image's scan was not completed
// successfully, assumes that the image haa at least one vulnerability.
func (ts *DelegatedScanningSuite) validateImageScan(t *testing.T, imgFullName string, img *storage.Image) {
	require.Equal(t, imgFullName, img.GetName().GetFullName())
	require.True(t, img.GetIsClusterLocal(), "image %q not flagged as cluster local which is expected for any delegated scans", imgFullName)
	require.NotNil(t, img.GetScan(), "image scan for %q is nil, check logs for errors, image notes: %v", imgFullName, img.GetNotes())
	require.NotEmpty(t, img.GetScan().GetComponents(), "image scan for %q has no components, check central logs for errors, this can happen if indexing succeeds but matching fails, ROX-17472 will make this an error in the future", imgFullName)

	// Ensure at least one component has a vulnerability.
	for _, c := range img.GetScan().GetComponents() {
		if len(c.GetVulns()) > 0 {
			return
		}
	}

	require.Fail(t, "No vulnerabilities found.", "Expected at least one vulnerability in image %q, but found none.", imgFullName)
}

// getImageWithRetires will retry attempts to get an image from the stackrox API, and retry if not found.
func (ts *DelegatedScanningSuite) getImageWithRetires(t *testing.T, ctx context.Context, service v1.ImageServiceClient, req *v1.GetImageRequest) (*storage.Image, error) {
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

	betweenAttemptsFunc := func(num int) {
		t.Logf("Image not found, trying again %s, attempt %d/%d", deleScanDefaultRetryDelay, num, deleScanDefaultMaxRetries)
		time.Sleep(deleScanDefaultRetryDelay)
	}

	err = retry.WithRetry(retryFunc,
		retry.BetweenAttempts(betweenAttemptsFunc),
		retry.Tries(deleScanDefaultMaxRetries),
		retry.OnlyRetryableErrors(),
	)

	return img, err
}

// scanWithRetries will retry attempts scan an image using the stackrox API, and retry when rate limited.
func (ts *DelegatedScanningSuite) scanWithRetries(t *testing.T, ctx context.Context, service v1.ImageServiceClient, req *v1.ScanImageRequest) (*storage.Image, error) {
	var err error
	var img *storage.Image

	retryFunc := func() error {
		img, err = service.ScanImage(ctx, req)

		if err != nil && strings.Contains(err.Error(), scan.ErrTooManyParallelScans.Error()) {
			err = retry.MakeRetryable(err)
		}

		return err
	}

	betweenAttemptsFunc := func(num int) {
		t.Logf("Too many parallel scans, trying again in %s, attempt %d/%d", deleScanDefaultRetryDelay, num, deleScanDefaultMaxRetries)
		time.Sleep(deleScanDefaultRetryDelay)
	}

	err = retry.WithRetry(retryFunc,
		retry.BetweenAttempts(betweenAttemptsFunc),
		retry.Tries(deleScanDefaultMaxRetries),
		retry.WithExponentialBackoff(),
		retry.OnlyRetryableErrors(),
	)

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

// getSensorLastLogBytePos gets the last byte from the Sensor logs to assist in only searching logs for only 'new'
// matching lines.
func (ts *DelegatedScanningSuite) getSensorLastLogBytePos(ctx context.Context) int64 {
	t := ts.T()
	sensorPod, err := ts.getSensorPod(ctx, ts.namespace)
	require.NoError(t, err)

	fromByte, err := ts.getLastLogBytePos(ctx, ts.namespace, sensorPod.GetName(), sensorPod.Spec.Containers[0].Name)
	require.NoError(t, err)

	return fromByte
}

// resetConfig resets the delegated scanning config to default value.
func (ts *DelegatedScanningSuite) resetConfig(ctx context.Context) {
	t := ts.T()
	conn := centralgrpc.GRPCConnectionToCentral(t)

	service := v1.NewDelegatedRegistryConfigServiceClient(conn)
	_, err := service.UpdateConfig(ctx, &v1.DelegatedRegistryConfig{})
	require.NoError(t, err)
}

// setConfig sets the delegated scanning config to delegate scans to
// the registry/remote of the provided image.
func (ts *DelegatedScanningSuite) setConfig(ctx context.Context, imageStr string) {
	t := ts.T()

	cImage, err := utils.GenerateImageFromString(imageStr)
	require.NoError(t, err)

	path := fmt.Sprintf("%s/%s", cImage.GetName().GetRegistry(), cImage.GetName().GetRemote())

	conn := centralgrpc.GRPCConnectionToCentral(t)
	service := v1.NewDelegatedRegistryConfigServiceClient(conn)

	cfg := &v1.DelegatedRegistryConfig{
		EnabledFor:       v1.DelegatedRegistryConfig_SPECIFIC,
		DefaultClusterId: ts.remoteCluster.GetId(),
		Registries: []*v1.DelegatedRegistryConfig_DelegatedRegistry{
			{Path: path},
		},
	}
	_, err = service.UpdateConfig(ctx, cfg)
	require.NoError(t, err)
}

// resetAccess will revoke any API token and delete roles, permissions sets
// and access scopes that were created as part of this test suite.
func (ts *DelegatedScanningSuite) resetAccess(ctx context.Context) {
	t := ts.T()

	revokeAPIToken(t, ctx, deleScanAPITokenName)
	deleteRole(t, ctx, deleScanRoleName)
	deletePermissionSet(t, ctx, deleScanPermissionSetName)
	deleteAccessScope(t, ctx, deleScanAccessScopeName)
}

// waitUntilSensorLogsScan will wait until Sensor logs show a successful scan has been completed.
func (ts *DelegatedScanningSuite) waitUntilSensorLogsScan(ctx context.Context, imageStr string, fromByte int64) {
	reStr := fmt.Sprintf(`Image "%s".* enriched with metadata using pull source`, imageStr)
	regexp.MustCompile(reStr)

	ts.waitUntilLog(ctx, "contain the image scan",
		containsLineMatchingAfter(regexp.MustCompile(reStr), fromByte),
	)
}

// getLimitedCentralConn will return a connection to central using a token defined by the provided
// permission set, role, and access scope. The role.permissionSetId provided will be overridden
// on execution.
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

func (ts *DelegatedScanningSuite) executeAndValidateScan(ctx context.Context, conn *grpc.ClientConn, req *v1.ScanImageRequest) {
	t := ts.T()

	// Get the last byte in the Sensor log, so that we only search for the
	// logs AFTER that byte after we execute the scan.  This allows us to
	// re-use the same image in various tests and should also speed up
	// the search.
	fromByte := ts.getSensorLastLogBytePos(ctx)

	// Scan the image via the image service, retrying as needed.
	service := v1.NewImageServiceClient(conn)
	img, err := ts.scanWithRetries(t, ctx, service, req)
	require.NoError(t, err)

	// Validate that the image scan looks 'OK'
	imgStr := req.GetImageName()
	ts.validateImageScan(t, imgStr, img)

	// Search the Sensor logs for a record of the scan that was just completed
	ts.waitUntilSensorLogsScan(ctx, imgStr, fromByte)
}
