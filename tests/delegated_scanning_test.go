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
	"github.com/stackrox/rox/pkg/namespaces"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common/scan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

// Delegated Scanning tests that verify the expected behavior for scans executed in
// the secured cluster(s).
//
// These tests should NOT be run in parallel with other tests.  Changes are made to
// the delegated scanning config and OCP mirroring CRs (ie: ImageContentSourcePolicy,
// ImageDigestMirrorSet, ImageTagMirrorSet) which may break other unrelated tests.
//
// These tests DO NOT validate that scan results contain specific packages or vulnerabilities,
// (we rely on the scanner specific tests for that). These tests instead focus on the foundational
// capabilities of delegated scanning to index images in Secured Clusters, and match
// any vulnerabilities via Central Services.
//
// All scans are executed with the `force` flag to ensure cache's are bypassed.
// TODO: perhaps remove this and add a test without the force flag to ensure that the cache
// is actually used to detect a regression in the force flag???
//
// These tests require the following env vars to be set in order to contact ACS/K8s:
// - ROX_USERNAME
// - ROX_PASSWORD
// - API_ENDPOINT
// - KUBECONFIG
// - QUAY_RHACS_ENG_RO_USERNAME
// - QUAY_RHACS_ENG_RO_PASSWORD

const (
	deleScanLogLevelEnvVar = "LOGLEVEL"

	// deleScanDesiredLogLevel is the desired value of Sensor's log level env var for executing these tests.
	deleScanDesiredLogLevel = "Debug"

	// deleScanAPITokenName is the name of the current stackrox API token that is created (and revoked)
	// by this suite.
	deleScanAPITokenName      = "dele-scan-api-token" //nolint:gosec // G101
	deleScanAccessScopeName   = "dele-scan-access-scope"
	deleScanRoleName          = "dele-scan-role"
	deleScanPermissionSetName = "dele-scan-permission-set"
)

var (
	// anonTestImageStr references an image that does not require auth to access and is small.
	// This image was chosen so that scans are fast (hence the small image size).
	anonTestImageStr = "registry.access.redhat.com/ubi9/ubi-minimal:9.4-1194"

	denyAllAccessScope  = accesscontrol.DefaultAccessScopeIDs[accesscontrol.DenyAllAccessScope]
	allowAllAccessScope = accesscontrol.DefaultAccessScopeIDs[accesscontrol.UnrestrictedAccessScope]
)

type DaveSuite struct {
	KubernetesSuite
}

func TestDave(t *testing.T) {
	suite.Run(t, new(DaveSuite))
}

func (ts *DaveSuite) TestDoIt() {
	t := ts.T()
	ctx := context.Background()

	thing := deleScanTestUtils{restCfg: getConfig(t)}
	user := mustGetEnv(t, "QUAY_RHACS_ENG_RO_USERNAME")
	pass := mustGetEnv(t, "QUAY_RHACS_ENG_RO_PASSWORD")
	thing.addCredToOCPGlobalPullSecret(t, ctx, "quay.io/rhacs-eng/", config.DockerConfigEntry{
		Username: user,
		Password: pass,
		Email:    "dele-scan-test@example.com",
	})

}

type DelegatedScanningSuite struct {
	KubernetesSuite
	ctx        context.Context
	cleanupCtx context.Context
	cancel     func()

	// origSensorLogLevel is the value of Sensor's log level env var prior to any changes / this suite executing.
	origSensorLogLevel string
	remoteCluster      *v1.DelegatedRegistryCluster
	namespace          string
	restCfg            *rest.Config

	quayROUsername string
	quayROPassword string

	failureHandled atomic.Bool
}

func TestDelegatedScanning(t *testing.T) {
	suite.Run(t, new(DelegatedScanningSuite))
}

func (ts *DelegatedScanningSuite) SetupSuite() {
	// Ensure final failure is handled
	ts.T().Cleanup(ts.handleFailure)

	ts.KubernetesSuite.SetupSuite()
	ts.namespace = namespaces.StackRox

	t := ts.T()
	ts.ctx, ts.cleanupCtx, ts.cancel = testContexts(t, "TestDelegatedScanning", 30*time.Minute)
	ts.restCfg = getConfig(t)

	ctx := ts.ctx

	// Get a reference to the secured cluster to send delegated scans too, will use this reference throughout the tests.
	// We check this first because if a valid remote cluster is NOT available all tests in this suite will fail.
	t.Log("Getting remote stackrox cluster details")
	envVal, err := ts.getDeploymentEnvVal(ctx, ts.namespace, sensorDeployment, sensorContainer, env.LocalImageScanningEnabled.EnvVar())
	require.NoError(t, err)

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

	// Pre-clean any roles, permissions, etc. from previous runs, these statements are
	// in a specific order to ensure successful cleanup.
	t.Log("Deleteing resources from previous tests")
	revokeAPIToken(t, ctx, deleScanAPITokenName)
	deleteRole(t, ctx, deleScanRoleName)
	deletePermissionSet(t, ctx, deleScanPermissionSetName)
	deleteAccessScope(t, ctx, deleScanAccessScopeName)
}

func (ts *DelegatedScanningSuite) TearDownSuite() {
	t := ts.T()
	ctx := ts.cleanupCtx
	ns := ts.namespace

	revokeAPIToken(t, ctx, deleScanAPITokenName)
	deleteRole(t, ctx, deleScanRoleName)
	deletePermissionSet(t, ctx, deleScanPermissionSetName)
	deleteAccessScope(t, ctx, deleScanAccessScopeName)

	// Reset the log level back to its original value so that other tests are not impacted by the additional logging.
	if ts.origSensorLogLevel != deleScanDesiredLogLevel {
		if ts.origSensorLogLevel != "" {
			ts.mustSetDeploymentEnvVal(ctx, ns, sensorDeployment, sensorContainer, deleScanLogLevelEnvVar, ts.origSensorLogLevel)
			t.Logf("Log level reverted back to %q on Sensor", ts.origSensorLogLevel)
		} else {
			ts.mustDeleteDeploymentEnvVar(ctx, ns, sensorDeployment, deleScanLogLevelEnvVar)
			t.Logf("Log level env var removed from Sensor")
		}
	}

	ts.cancel()
}

// handleFailure is a catch all for handling test suite failures, invoked via t.Cleanup in SetupSuite AND
// as part of TearDownSuite. We cannot do this cleanup solely in TearDownSuite because a failure in SetupSuite
// prevents TearDownSuite from executing. Subsequent invocations after the first will be no-ops.
//
// Initial use case for this is to ensure that logs are captured on any suite failures.
//
// TODO: is it better to handle log captures after each individual test, since each test may
// trigger pod restarts / make changes that may wipe logs.
func (ts *DelegatedScanningSuite) handleFailure() {
	if ts.failureHandled.Swap(true) {
		ts.T().Log("Failure already handled")
		return
	}

	t := ts.T()
	if t.Failed() {
		ts.logf("Test failed. Collecting k8s artifacts before cleanup.")
		collectLogs(t, ts.namespace, "delegated-scanning-failure")
	}
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
		// Be a good citizen and return the config back to its original value.
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
		_, err = service.DeleteImageIntegration(ctx, &v1.ResourceByID{Id: id})
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

	scanImgReq := func(imgStr string) *v1.ScanImageRequest {
		return &v1.ScanImageRequest{
			ImageName: imgStr,
			Force:     true,
			Cluster:   ts.remoteCluster.GetId(),
		}
	}

	// The image write permission is required to scan images, this test uses a token with only image
	// read permissions, and therefore should fail to scan any image. This test is meant to catch
	// a past regression where a role with only image read permissions was mistakenly able to scan images.
	t.Run("fail to scan image with only image read permission", func(t *testing.T) {
		// Create a new read only permission set.
		permissionSetID := mustCreatePermissionSet(t, ctx, &storage.PermissionSet{
			Name: deleScanPermissionSetName,
			ResourceToAccess: map[string]storage.Access{
				"Image": storage.Access_READ_ACCESS,
			},
		})
		t.Cleanup(func() { deletePermissionSet(t, ctx, permissionSetID) })

		// Create the role for the new permission set.
		role := &storage.Role{
			Name:            deleScanRoleName,
			PermissionSetId: permissionSetID,
			AccessScopeId:   allowAllAccessScope,
		}
		mustCreateRole(t, ctx, role)
		t.Cleanup(func() { deleteRole(t, ctx, role.GetName()) })

		// Create a token bound to the new role.
		tokenID, token := mustCreateAPIToken(t, ctx, deleScanAPITokenName, []string{deleScanRoleName})
		t.Cleanup(func() { revokeAPIToken(t, ctx, tokenID) })

		// Connect to central using the new token.
		conn := centralgrpc.GRPCConnectionToCentral(t, func(opts *clientconn.Options) {
			opts.ConfigureTokenAuth(token)
		})
		service := v1.NewImageServiceClient(conn)

		// Execute the scan.
		_, err := service.ScanImage(ctx, &v1.ScanImageRequest{
			ImageName: anonTestImageStr,
			Force:     true,
			Cluster:   ts.remoteCluster.GetId(),
		})
		require.ErrorContains(t, err, "not authorized")
	})

	t.Run("baseline delegated scan without auth", func(t *testing.T) {
		fromByte, err := ts.getLastLogBytePos(ctx, ts.namespace, sensorPod.GetName(), sensorPod.Spec.Containers[0].Name)
		require.NoError(t, err)

		img, err := ts.scanWithRetries(t, ctx, service, scanImgReq(anonTestImageStr))
		require.NoError(t, err)
		ts.validateImageScan(t, anonTestImageStr, img)

		reStr := fmt.Sprintf(`Image "%s".* enriched with metadata using pull source`, anonTestImageStr)
		ts.waitUntilLog(ctx, ts.namespace, sensorPodLabels, sensorContainer, "contain the image scan",
			containsLineMatchingAfterByte(regexp.MustCompile(reStr), fromByte),
		)
	})

	t.Run("scan image with image write permission", func(t *testing.T) {
	})

	t.Run("image watch", func(t *testing.T) {
		// set config and do image watch
		// fromLine, err := ts.getNumLogLines(ctx, ts.namespace, sensorPod.GetName(), sensorPod.Spec.Containers[0].Name)
		// require.NoError(t, err)
		// ts.logf("Only matching sensor logs after line %d from pod %s", fromLine, sensorPod.GetName())

		// // This image was chosen because it requires no auth and is small (no other reason).
		// service := v1.NewImageServiceClient(conn)
		// _, err = service.WatchImage(ctx, &v1.WatchImageRequest{Name: anonTestImageStr})
		// require.NoError(t, err)

		// _, err = service.UnwatchImage(ctx, &v1.UnwatchImageRequest{Name: anonTestImageStr})
		// assert.NoError(t, err)
	})

	t.Run("scan image from OCP internal registry", func(t *testing.T) {
		if !isOpenshift() {
			t.Skip("Skipping test - not an OCP cluster")
		}

		fromByte, err := ts.getLastLogBytePos(ctx, ts.namespace, sensorPod.GetName(), sensorPod.Spec.Containers[0].Name)
		require.NoError(t, err)

		// Push an to the OCP internal registry.
		ocpImgBuilder := deleScanTestUtils{restCfg: ts.restCfg}
		name := "test-01"
		imgStr := ocpImgBuilder.BuildOCPInternalImage(t, ts.ctx, ts.namespace, name, anonTestImageStr)

		// Scan the image.
		img, err := service.ScanImage(ctx, scanImgReq(imgStr))
		require.NoError(t, err)
		ts.validateImageScan(t, imgStr, img)

		// Verify the request made it to the cluster.
		reStr := fmt.Sprintf(`Image "%s".* enriched with metadata using pull source`, imgStr)
		ts.waitUntilLog(ctx, ts.namespace, sensorPodLabels, sensorContainer, "contain the image scan",
			containsLineMatchingAfterByte(regexp.MustCompile(reStr), fromByte),
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
	// 	service := v1.NewImageServiceClient(conn)
	// 	_, err = service.WatchImage(ctx, &v1.WatchImageRequest{Name: anonTestImageStr})
	// 	require.NoError(t, err)

	// 	_, err = service.UnwatchImage(ctx, &v1.UnwatchImageRequest{Name: anonTestImageStr})
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
	if !isOpenshift() {
		ts.T().Skip("Skipping test - not an OCP cluster")
	}
	t := ts.T()
	ctx := ts.ctx

	// Create mirroring CRs and update OCP global pull secret, this will
	// trigger nodes to drain and may take between 5-10 mins to complete.
	deleScanUtils := &deleScanTestUtils{restCfg: getConfig(t)}
	deleScanUtils.SetupMirrors(t, ctx, "quay.io/rhacs-eng/", config.DockerConfigEntry{
		Username: ts.quayROUsername,
		Password: ts.quayROPassword,
		Email:    "dele-scan-test@example.com",
	})

	// Sensor is avail quicker on fresh start when compared to time spent
	// waiting for it to reconnect to Central automatically.
	// Since Sensor may have been started first after the node drain triggered by mirror setup,
	// by deleting Sensor testing will be able to proceed quicker.
	t.Log("Deleteing Sensor to speed up ready state")
	sensorPod, err := ts.getSensorPod(ctx, ts.namespace)
	require.NoError(t, err)
	err = ts.k8s.CoreV1().Pods(ts.namespace).Delete(ctx, sensorPod.GetName(), metaV1.DeleteOptions{})
	require.NoError(t, err)

	// Wait for Central/Sensor to be healthy after potential restarts.
	t.Log("Waiting for Sensor to be ready")
	ts.waitUntilK8sDeploymentReady(ctx, ts.namespace, sensorDeployment)
	t.Log("Waiting for Central/Sensor connection to be ready")
	waitUntilCentralSensorConnectionIs(t, ctx, storage.ClusterHealthStatus_HEALTHY)

	// Get a reference to the sensorPod so that can track where the log ends to
	// ensure log matching only considers the data 'after' the test was ran.
	sensorPod, err = ts.getSensorPod(ctx, ts.namespace)
	require.NoError(t, err)

	// The mirroring CRs map to quay.io, these tests will attempt to scan images from quay.io/rhacs-eng,
	// to ensure authenticate succeeds we create an image integration.
	conn := centralgrpc.GRPCConnectionToCentral(t)

	deleService := v1.NewDelegatedRegistryConfigServiceClient(conn)

	// Get the current delegated scanning config so that we can revert
	// it when tests are finished.
	origCfg, err := deleService.GetConfig(ctx, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = deleService.UpdateConfig(ctx, origCfg) })

	t.Logf("Enabling delegated scanning for the mirror hosts")
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
	}{
		{
			"Scan ad-hoc image from mirror defined by ImageContentSourcePolicy",
			// Mirrors to: quay.io/rhacs-eng/qa:dele-scan-nginx
			"icsp.invalid/rhacs-eng/qa@sha256:68b418b74715000e41a894428bd787442945592486a08d4cbea89a9b4fa03302",
		},
		{
			"Scan ad-hoc image from mirror defined by ImageDigestMirrorSet",
			// Mirrors to: quay.io/rhacs-eng/qa:dele-scan-httpd
			"idms.invalid/rhacs-eng/qa@sha256:fe0ed227b13f672a268505770854730bb6b4ab40c507ed86a0b12e69471e47d9",
		},
		{
			"Scan ad-hoc image from mirror defined by ImageTagMirrorSet",
			"itms.invalid/rhacs-eng/qa:dele-scan-memcached",
		},
	}
	for _, tc := range adhocTCs {
		t.Run(tc.desc, func(t *testing.T) {
			fromByte, err := ts.getLastLogBytePos(ctx, ts.namespace, sensorPod.GetName(), sensorPod.Spec.Containers[0].Name)
			require.NoError(t, err)

			req := &v1.ScanImageRequest{
				ImageName: tc.imgStr,
				Force:     true,
				Cluster:   ts.remoteCluster.GetId(),
			}

			service := v1.NewImageServiceClient(conn)
			img, err := ts.scanWithRetries(t, ctx, service, req)
			require.NoError(t, err)
			ts.validateImageScan(t, tc.imgStr, img)

			// Verify the request made it to the cluster.
			reStr := fmt.Sprintf(`Image "%s".* enriched with metadata using pull source`, tc.imgStr)
			ts.waitUntilLog(ctx, ts.namespace, sensorPodLabels, sensorContainer, "contain the image scan",
				containsLineMatchingAfterByte(regexp.MustCompile(reStr), fromByte),
			)
		})
	}

	deployTCs := []struct {
		desc       string
		deployName string
		imgID      string
		imgStr     string
	}{
		{
			"Scan deployment image from mirror defined by ImageContentSourcePolicy",
			"dele-scan-icsp",
			// Mirrors to: quay.io/rhacs-eng/qa:dele-scan-nginx
			"sha256:68b418b74715000e41a894428bd787442945592486a08d4cbea89a9b4fa03302",
			"icsp.invalid/rhacs-eng/qa@sha256:68b418b74715000e41a894428bd787442945592486a08d4cbea89a9b4fa03302",
		},
		{
			"Scan deployment image from mirror defined by ImageDigestMirrorSet",
			"dele-scan-idms",
			// Mirrors to: quay.io/rhacs-eng/qa:dele-scan-httpd
			"sha256:fe0ed227b13f672a268505770854730bb6b4ab40c507ed86a0b12e69471e47d9",
			"idms.invalid/rhacs-eng/qa@sha256:fe0ed227b13f672a268505770854730bb6b4ab40c507ed86a0b12e69471e47d9",
		},
		{
			"Scan deployment image from mirror defined by ImageTagMirrorSet",
			"dele-scan-itms",
			"sha256:1cf25340014838bef90aa9d19eaef725a0b4986af3c8e8a6be3203c2cef8cb61",
			"itms.invalid/rhacs-eng/qa:dele-scan-memcached",
		},
	}

	for _, tc := range deployTCs {
		t.Run(tc.desc, func(t *testing.T) {
			fromByte, err := ts.getLastLogBytePos(ctx, ts.namespace, sensorPod.GetName(), sensorPod.Spec.Containers[0].Name)
			require.NoError(t, err)

			// Do an initial teardown in case a deployment is lingering from a previous test.
			teardownDeployment(t, tc.deployName)

			// Because we cannot 'force' a scan for deployments, we explicitly delete the image
			// so that it is removed from sensors scan cache.
			imageService := v1.NewImageServiceClient(conn)
			query := fmt.Sprintf("Image Sha:%s", tc.imgID)
			delResp, err := imageService.DeleteImages(ctx, &v1.DeleteImagesRequest{Query: &v1.RawQuery{Query: query}, Confirm: true})
			require.NoError(t, err)
			t.Logf("Num images deleted from query %q: %d", query, delResp.NumDeleted)

			t.Logf("Creating deployment %q with image: %q", tc.deployName, tc.imgStr)
			setupDeploymentNoWait(t, tc.imgStr, tc.deployName, 1)

			img, err := ts.getImageWithRetires(t, ctx, imageService, &v1.GetImageRequest{Id: tc.imgID})
			require.NoError(t, err)
			ts.validateImageScan(t, tc.imgStr, img)

			ts.waitUntilLog(ctx, ts.namespace, sensorPodLabels, sensorContainer, "contain the image scan",
				matchesAny(
					containsLineMatchingAfterByte(regexp.MustCompile(fmt.Sprintf(`Image "%s".* enriched with metadata using pull source`, tc.imgStr)), fromByte),
					// containsLineMatchingAfterByte(regexp.MustCompile(fmt.Sprintf(`Image "%s".* enriched with metadata using pull source`, tc.imgStr)), fromByte),
					// TODO: In the event that the time this out quicker and check for image being pulled from cache
				),
			)

			// Only perform teardown on success so that logs can be captured on failure.
			t.Logf("Tearing down deployment %q", tc.deployName)
			teardownDeployment(t, tc.deployName)
		})
	}
}

// validateImageScan will fail the test if the image's scan was not completed
// successfully, assumes that the image will have at least one vulnerability.
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

// getImageWithRetires will retry attempts to get an image.
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

	maxTries := 30
	betweenAttemptsFunc := func(num int) {
		t.Logf("Image not found, trying again, attempt %d/%d", num, maxTries)
		time.Sleep(5 * time.Second)
	}

	err = retry.WithRetry(retryFunc,
		retry.BetweenAttempts(betweenAttemptsFunc),
		retry.Tries(maxTries),
		retry.OnlyRetryableErrors(),
	)

	return img, err
}

// scanWithRetries will retry attempts scan an image when rate limited.
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

	maxTries := 30
	betweenAttemptsFunc := func(num int) {
		t.Logf("Too many parallel scans, trying again, attempt %d/%d", num, maxTries)
	}

	err = retry.WithRetry(retryFunc,
		retry.BetweenAttempts(betweenAttemptsFunc),
		retry.Tries(maxTries),
		retry.WithExponentialBackoff(),
		retry.OnlyRetryableErrors(),
	)

	return img, err
}
