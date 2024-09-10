//go:build test_e2e

package tests

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
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
//   ROX_USERNAME, ROX_PASSWORD, API_ENDPOINT, KUBECONFIG

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

	// imageReadPermissionSet represents the minimal permission set needed for reading images (but
	// should NOT be able to scan/write images).
	imageReadPermissionSet = &storage.PermissionSet{
		Name: "dele-scan-image-read-permission-set",
		ResourceToAccess: map[string]storage.Access{
			"Image": storage.Access_READ_ACCESS,
		},
	}

	denyAllAccessScope  = accesscontrol.DefaultAccessScopeIDs[accesscontrol.DenyAllAccessScope]
	allowAllAccessScope = accesscontrol.DefaultAccessScopeIDs[accesscontrol.UnrestrictedAccessScope]

	// imageReadRole is a template role (meant to be cloned) representing the minimal
	// permissions needed for reading images.  Will bind the read only API token to this role.
	imageReadRole = &storage.Role{
		Name:            "dele-scan-image-read-role",
		PermissionSetId: accesscontrol.DefaultAccessScopeIDs[accesscontrol.Analyst],
		AccessScopeId:   allowAllAccessScope,
	}

	// namespaceAccessScope is a template access scope (meant to be cloned) and allows
	// access to specific namespaces.
	namespaceAccessScope = &storage.SimpleAccessScope{
		Name: "dele-scan-namespace-access-scope",
		Rules: &storage.SimpleAccessScope_Rules{
			IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
				{ClusterName: "", NamespaceName: "stackrox"},
			},
		},
	}
)

type DaveSuite struct {
	KubernetesSuite
}

func TestDave(t *testing.T) {
	suite.Run(t, new(DaveSuite))
}

func (ts *DaveSuite) TestDoIt() {
	t := ts.T()

	thing := deleScanTestUtils{
		restCfg: getConfig(t),
	}
	// imgStr := ocpImgBuilder.BuildOCPInternalImage(name, anonTestImageStr)
	thing.CreateMirrorCRs(t, context.Background())
}

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

	// imageReadToken minimally scoped API token that has the image read permission (NOT image write).
	imageReadToken string
	// imageWriteToken minimally scoped API token that has the image write permission
	imageWriteToken string

	// imageReadPermissionSetID holds the id of the created permission for cleanup purposes.
	imageReadPermissionSetID string

	// namespaceAccessScopeID holds the id of the created access scope for binding and cleanup purposes.
	namespaceAccessScopeID string

	quayROUsername string
	quayROPassword string
}

func TestDelegatedScanning(t *testing.T) {
	suite.Run(t, new(DelegatedScanningSuite))
}

func (ts *DelegatedScanningSuite) TestSetupTeardown() {
	ts.T().Logf("EXECUTING FAKE TESTS...")
}

func (ts *DelegatedScanningSuite) SetupSuite() {
	ts.KubernetesSuite.SetupSuite()
	ts.namespace = namespaces.StackRox

	t := ts.T()
	ts.ctx, ts.cleanupCtx, ts.cancel = testContexts(t, "TestDelegatedScanning", 30*time.Minute)
	ts.restCfg = getConfig(t)

	ctx := ts.ctx

	// Ensure rhacs-eng repo user/pass is avail.
	ts.quayROUsername = mustGetEnv(t, "QUAY_RHACS_ENG_RO_USERNAME")
	ts.quayROPassword = mustGetEnv(t, "QUAY_RHACS_ENG_RO_PASSWORD")

	// Enable Sensor debug logs which some tests rely on Sensor to accurately validate expected behaviors.
	ts.origSensorLogLevel, _ = ts.getDeploymentEnvVal(ctx, ts.namespace, sensorDeployment, sensorContainer, deleScanLogLevelEnvVar)
	if ts.origSensorLogLevel != deleScanDesiredLogLevel {
		ts.mustSetDeploymentEnvVal(ctx, ts.namespace, sensorDeployment, sensorContainer, deleScanLogLevelEnvVar, deleScanDesiredLogLevel)
		t.Logf("Log level env var changed from %q to %q on Sensor", ts.origSensorLogLevel, deleScanDesiredLogLevel)
	}

	// TOOD: Uncomment when done with local testing
	/*
		// The changes above may have triggered a Sensor restart, wait for it to be healthy.
		t.Log("Waiting for Sensor to be ready")
		ts.waitUntilK8sDeploymentReady(ctx, ts.namespace, sensorDeployment)
		t.Log("Waiting for Central/Sensor connection to be ready")
		waitUntilCentralSensorConnectionIs(t, ctx, storage.ClusterHealthStatus_HEALTHY)
	*/
	// Get the remote cluster to send delegated scans too, will use this to obtain the cluster name, Id, etc.
	t.Log("Getting remote stackrox cluster details")
	ts.remoteCluster = mustGetCluster(t, ctx)

	// Pre-clean any roles, permissions, etc. from previous runs, these statements are
	// in a specific order to ensure successful cleanup.
	t.Log("Deleteing resources from previous tests")
	revokeAPIToken(t, ctx, deleScanAPITokenName)
	deleteRole(t, ctx, deleScanRoleName)
	deletePermissionSet(t, ctx, deleScanPermissionSetName)
	deleteAccessScope(t, ctx, deleScanAccessScopeName)

	/*
		// Create an access scope limited to the stackrox namespace
		scope := namespaceAccessScope.CloneVT()
		scope.GetRules().GetIncludedNamespaces()[0].ClusterName = ts.remoteCluster.GetName()
		ts.namespaceAccessScopeID = mustCreateAccessScope(t, ctx, scope)

		// Create a read only permission set.
		ts.imageReadPermissionSetID = mustCreatePermissionSet(t, ctx, imageReadPermissionSet)

		// Create a read only role.
		role := imageReadRole.CloneVT()
		role.PermissionSetId = ts.imageReadPermissionSetID
		role.AccessScopeId = ts.namespaceAccessScopeID
		mustCreateRole(t, ctx, role)

		// Create a read only token (needed by some tests)
		_, ts.imageReadToken = mustCreateAPIToken(t, ctx, deleScanAPITokenName, []string{imageReadRole.GetName()})
	*/
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

	// revokeAPIToken(t, ctx, imageReadTokenName)
	// deleteRole(t, ctx, imageReadRole.GetName())
	// deletePermissionSet(t, ctx, ts.imageReadPermissionSetID)

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

	// TOOD: Uncomment the below section when done with local testing
	/*
		// Ensure central/sensor are in a good state before moving to next test's to avoid flakes.
		t.Log("Waiting for Sensor to be ready (On Tear Down)")
		ts.waitUntilK8sDeploymentReady(ctx, ns, sensorDeployment)
		t.Log("Waiting for Central/Sensor connection to be ready (On Tear Down)")
		waitUntilCentralSensorConnectionIs(t, ctx, storage.ClusterHealthStatus_HEALTHY)
	*/
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
		fromLine, err := ts.getNumLogLines(ctx, ts.namespace, sensorPod.GetName(), sensorPod.Spec.Containers[0].Name)
		require.NoError(t, err)
		ts.logf("Only matching sensor logs after line %d from pod %s", fromLine, sensorPod.GetName())

		img, err := ts.scanWithRetries(t, ctx, service, scanImgReq(anonTestImageStr))
		require.NoError(t, err)
		ts.validateImageScan(t, anonTestImageStr, img)

		reStr := fmt.Sprintf(`Image "%s".* enriched with metadata using pull source`, anonTestImageStr)
		ts.waitUntilLog(ctx, ts.namespace, sensorPodLabels, sensorContainer, "contain the image scan",
			containsLineMatchingAfter(regexp.MustCompile(reStr), fromLine),
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

		fromLine, err := ts.getNumLogLines(ctx, ts.namespace, sensorPod.GetName(), sensorPod.Spec.Containers[0].Name)
		require.NoError(t, err)
		ts.logf("Only matching sensor logs after line %d from pod %s", fromLine, sensorPod.GetName())

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
			containsLineMatchingAfter(regexp.MustCompile(reStr), fromLine),
		)
	})

	// Create role for just image scanning
	// Create user that uses that role
	// Get token for user that uses that role
	t.Run("scan image from OCP internal registry with minimally scoped user", func(t *testing.T) {
		t.SkipNow()

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
	t := ts.T()
	ctx := ts.ctx

	deleScanUtils := deleScanTestUtils{restCfg: getConfig(t)}
	deleScanUtils.CreateMirrorCRs(t, ctx)

	// The changes above may have triggered Central and/or Sensor restarts, wait for both to be healthy.
	t.Log("Waiting for Sensor to be ready")
	ts.waitUntilK8sDeploymentReady(ctx, ts.namespace, sensorDeployment)
	t.Log("Waiting for Central/Sensor connection to be ready")
	waitUntilCentralSensorConnectionIs(t, ctx, storage.ClusterHealthStatus_HEALTHY)

	sensorPod, err := ts.getSensorPod(ctx, ts.namespace)
	require.NoError(t, err)

	// The mirroring CRs map to quay.io, these tests will attempt to scan images from quay.io/rhacs-eng,
	// in ensure authenticate succeeds we create an image integration.
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
	t.Cleanup(func() { iiService.DeleteImageIntegration(ctx, &v1.ResourceByID{Id: rii.GetId()}) })

	deleService := v1.NewDelegatedRegistryConfigServiceClient(conn)

	// Get the current config so that we can undo any changes when
	// tests are finished.
	origCfg, err := deleService.GetConfig(ctx, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		// Be a good citizen and return the config back to its original value.
		t.Logf("Resetting DELE config")
		_, _ = deleService.UpdateConfig(ctx, origCfg)
	})

	t.Logf("Enable delegated scanning for the mirror sources")
	_, err = deleService.UpdateConfig(ctx, &v1.DelegatedRegistryConfig{
		EnabledFor: v1.DelegatedRegistryConfig_SPECIFIC,
		Registries: []*v1.DelegatedRegistryConfig_DelegatedRegistry{
			{Path: "icsp.invalid"},
			{Path: "idms.invalid"},
			{Path: "itms.invalid"},
		},
	})
	require.NoError(t, err)

	// TODO: add read only creds to the OCP global secret so that deployments are actually created for all scans.

	adhocTCs := []struct {
		desc   string
		imgStr string
	}{
		{
			"Scan ad-hoc image from mirror defined by ImageContentSourcePolicy",
			// Mirrors to: quay.io/rhacs-eng/qa:dele-scan-nginx
			"icsp.invalid/rhacs-eng/qa@sha256:596c783ac62b9a43c60edb876fe807376cd5022a4e25e89b9a9ae06c374299d4",
		},
		{
			"Scan ad-hoc image from mirror defined by ImageDigestMirrorSet",
			// Mirrors to: quay.io/rhacs-eng/qa:dele-scan-httpd
			"idms.invalid/rhacs-eng/qa@sha256:9a1f89a470bc5ec4c5dc9e1aa13bd20898d59b71890cf27098507fc39750d988",
		},
		{
			"Scan ad-hoc image from mirror defined by ImageTagMirrorSet",
			"itms.invalid/rhacs-eng/qa:dele-scan-memcached",
		},
	}
	for _, tc := range adhocTCs {
		t.Run(tc.desc, func(t *testing.T) {
			t.SkipNow()
			fromLine, err := ts.getNumLogLines(ctx, ts.namespace, sensorPod.GetName(), sensorPod.Spec.Containers[0].Name)
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
			t.Logf("%s: Metadata: %v, Scan: %v", img.GetName().GetFullName(), img.GetMetadata().GetDataSource(), img.GetScan().GetDataSource())

			// Verify the request made it to the cluster.
			reStr := fmt.Sprintf(`Image "%s".* enriched with metadata using pull source`, tc.imgStr)
			ts.waitUntilLog(ctx, ts.namespace, sensorPodLabels, sensorContainer, "contain the image scan",
				containsLineMatchingAfter(regexp.MustCompile(reStr), fromLine),
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
			"sha256:596c783ac62b9a43c60edb876fe807376cd5022a4e25e89b9a9ae06c374299d4",
			// Mirrors to: quay.io/rhacs-eng/qa:dele-scan-nginx
			"icsp.invalid/rhacs-eng/qa@sha256:596c783ac62b9a43c60edb876fe807376cd5022a4e25e89b9a9ae06c374299d4",
		},
		{
			"Scan deployment image from mirror defined by ImageDigestMirrorSet",
			"dele-scan-idms",
			"sha256:9a1f89a470bc5ec4c5dc9e1aa13bd20898d59b71890cf27098507fc39750d988",
			// Mirrors to: quay.io/rhacs-eng/qa:dele-scan-httpd
			"idms.invalid/rhacs-eng/qa@sha256:9a1f89a470bc5ec4c5dc9e1aa13bd20898d59b71890cf27098507fc39750d988",
		},
		{
			"Scan deployment image from mirror defined by ImageTagMirrorSet",
			"dele-scan-itms",
			"sha256:98f8ec75657d21b924fe4f69b6b9bff2f6550ea48838af479d8894a852000e40",
			// To validate the scan for a tagged image was successful, we need a deployment's pods in a
			// Running state in order to extract the image ID (from the .status.containerStatuses.imageID field). This
			// image ID is required to save scan results to Central DB and subsequently for examining scan results.
			//
			// Mirroed images that require auth enter imagePullBackoff unless creds are added to the OCP global pull secret
			// OR we use an image that does not require auth. Here we are using an image that does not require auth.
			"itms.invalid/library/nginx:1.27.0",
		},
	}

	for _, tc := range deployTCs {
		t.Run(tc.desc, func(t *testing.T) {
			fromLine, err := ts.getNumLogLines(ctx, ts.namespace, sensorPod.GetName(), sensorPod.Spec.Containers[0].Name)
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

			t.Logf("Setting up deployment %q with image: %q", tc.deployName, tc.imgStr)
			setupDeploymentNoWait(t, tc.imgStr, tc.deployName, 1)
			t.Cleanup(func() {
				t.Logf("Tearing down deployment %q", tc.deployName)
				teardownDeployment(t, tc.deployName)
			})

			img, err := ts.getImageWithRetires(t, ctx, imageService, &v1.GetImageRequest{Id: tc.imgID})
			require.NoError(t, err)
			ts.validateImageScan(t, tc.imgStr, img)

			ts.waitUntilLog(ctx, ts.namespace, sensorPodLabels, sensorContainer, "contain the image scan",
				matchesAny(
					containsLineMatchingAfter(regexp.MustCompile(fmt.Sprintf(`Image "%s".* enriched with metadata using pull source`, tc.imgStr)), fromLine),
				),
			)
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
