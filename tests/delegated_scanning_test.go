//go:build test_e2e

package tests

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

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

// This file contains Delegated Scanning tests that verify the expected behavior
// for scans executed in the secured cluster(s).
//
// These tests should NOT be run in parallel with other tests because they
// change the delegated scanning config and OCP mirroring CRs
// (ie: ImageContentSourcePolicy, ImageDigestMirrorSet, ImageTagMirrorSet) which may
// break other unrelated tests.
//
// These tests require the following env vars to be set in order to contact ACS/K8s:
//   ROX_USERNAME, ROX_PASSWORD, API_ENDPOINT, KUBECONFIG

const (
	logLevelEnvVar = "LOGLEVEL"
)

var (
	sensorPodLabels = map[string]string{"app": "sensor"}
)

type DelegatedScanningSuite struct {
	KubernetesSuite
	ctx                context.Context
	cleanupCtx         context.Context
	cancel             func()
	origSensorLogLevel string // the original log level on sensor prior to any changes
	desiredLogLevel    string
	remoteClusterName  string
}

func (ts *DelegatedScanningSuite) SetupSuite() {
	ts.desiredLogLevel = "Debug"
	ts.KubernetesSuite.SetupSuite()
	ts.ctx, ts.cleanupCtx, ts.cancel = testContexts(ts.T(), "TestDelegatedScanning", 15*time.Minute)

	// Some tests rely on Sensor debug logs to accurately validate expected behaviors.
	ts.origSensorLogLevel = ts.getDeploymentEnvVal(ts.ctx, namespaces.StackRox, sensorDeployment, sensorContainer, logLevelEnvVar)
	if ts.origSensorLogLevel != ts.desiredLogLevel {
		ts.mustSetDeploymentEnvVal(ts.ctx, namespaces.StackRox, sensorDeployment, sensorContainer, logLevelEnvVar, ts.desiredLogLevel)
		ts.T().Logf("Log level changed from %q to %q on Sensor", ts.origSensorLogLevel, ts.desiredLogLevel)
	}

	// The changes above may have triggered a Sensor restart, wait for it to be healthy.
	ts.T().Log("Waiting for Sensor to be ready")
	ts.waitUntilK8sDeploymentReady(ts.ctx, namespaces.StackRox, sensorDeployment)

	ts.T().Log("Waiting for Central/Sensor connection to be ready")
	waitUntilCentralSensorConnectionIs(ts.T(), ts.ctx, storage.ClusterHealthStatus_HEALTHY)
}

func (ts *DelegatedScanningSuite) TearDownSuite() {
	// Reset the log level back to its original value so that other tests are not impacted by the additional logging.
	if ts.origSensorLogLevel != ts.desiredLogLevel {
		if ts.origSensorLogLevel != "" {
			ts.mustSetDeploymentEnvVal(ts.cleanupCtx, namespaces.StackRox, sensorDeployment, sensorContainer, logLevelEnvVar, ts.origSensorLogLevel)
		} else {
			ts.mustDeleteDeploymentEnvVar(ts.cleanupCtx, namespaces.StackRox, sensorDeployment, logLevelEnvVar)
		}
	}

	ts.cancel()
}

func TestDelegatedScanning(t *testing.T) {
	suite.Run(t, new(DelegatedScanningSuite))
}

// TestDelegatedRegistryConfig verifies that changes made to the delegated registry config
// stick and are propagated to the secured clusters.
func (ts *DelegatedScanningSuite) TestDelegatedRegistryConfig() {
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

		ts.waitUntilLog(ctx, namespaces.StackRox, sensorPodLabels, sensorContainer, "contain delegate registry config upsert",
			containsLineMatching(regexp.MustCompile(fmt.Sprintf("Upserted delegated registry config.*%s", path))),
		)
	})
}

// TestImageIntegrations tests various aspects of image integrations (such as syncing) related
// to delegated scanning.
func (ts *DelegatedScanningSuite) TestImageIntegrations() {
	t := ts.T()
	ctx := ts.ctx

	conn := centralgrpc.GRPCConnectionToCentral(t)
	service := v1.NewImageIntegrationServiceClient(conn)

	// This test verifies there are no autogenerated image integrations created for the
	// OCP internal registry.  This doesn't directly test Delegated Scanning, however
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

	// Created, updated, and deleted image integrations should be synced every secured cluster in
	// order for users to have control (if desired) over the credentials used for delegated scanning.
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

		// Create image integration
		rii, err := service.PostImageIntegration(ctx, ii)
		require.NoError(t, err)
		id := rii.GetId()

		ts.waitUntilLog(ctx, namespaces.StackRox, sensorPodLabels, sensorContainer, "contain the upserted integration",
			// Requires debug logging
			containsLineMatching(regexp.MustCompile(fmt.Sprintf("Upserted registry integration.*%s", id))),
		)

		// Update image integration
		rii.GetDocker().Insecure = !rii.GetDocker().Insecure
		_, err = service.UpdateImageIntegration(ctx, &v1.UpdateImageIntegrationRequest{Config: rii, UpdatePassword: false})
		require.NoError(t, err)

		ts.waitUntilLog(ctx, namespaces.StackRox, sensorPodLabels, sensorContainer, "contain the upserted integration",
			// Requires debug logging
			containsMultipleLinesMatching(regexp.MustCompile(fmt.Sprintf("Upserted registry integration.*%s", id)), 2),
		)

		// Delete the image integration
		_, err = service.DeleteImageIntegration(ctx, &v1.ResourceByID{Id: rii.GetId()})
		require.NoError(t, err)

		ts.waitUntilLog(ctx, namespaces.StackRox, sensorPodLabels, sensorContainer, "contain the deleted integration",
			// Requires debug logging
			containsLineMatching(regexp.MustCompile(fmt.Sprintf("Deleted registry integration.*%s", id))),
		)
	})

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

		// Create image integration
		rii, err := service.PostImageIntegration(ctx, ii)
		require.NoError(t, err)
		id := rii.GetId()

		ts.checkLogsMatch(ctx, namespaces.StackRox, sensorPodLabels, sensorContainer, "contain the upserted integration",
			// Requires debug logging
			containsNoLinesMatching(regexp.MustCompile(fmt.Sprintf("Upserted registry integration.*%s", id))),
		)
	})
}

func (ts *DelegatedScanningSuite) TestDelegatedScans() {
	t := ts.T()
	ctx := ts.ctx

	cluster := mustGetCluster(t, ctx)
	conn := centralgrpc.GRPCConnectionToCentral(t)

	imgStr := "registry.access.redhat.com/ubi9/ubi:9.4-1181"
	reStr := fmt.Sprintf(`Image "%s".* enriched with metadata using pull source`, imgStr)

	// Verify that nothing else has scanned this image
	// ts.checkLogsMatch(ctx, namespaces.StackRox, sensorPodLabels, sensorContainer, "do not contain the image scan",
	// 	containsNoLinesMatching(regexp.MustCompile(reStr)),
	// )

	service := v1.NewImageServiceClient(conn)

	var img *storage.Image
	var err error

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

	maxTries := 10
	betweenAttemptsFunc := func(num int) {
		t.Logf("Too many parallel scans, trying again, attempt %d/%d", num, maxTries)
	}

	retry.WithRetry(scanImageFunc,
		retry.BetweenAttempts(betweenAttemptsFunc),
		retry.Tries(maxTries),
		retry.WithExponentialBackoff(),
		retry.OnlyRetryableErrors(),
	)

	require.NoError(t, err)
	require.Equal(t, imgStr, img.GetName().GetFullName())
	require.NotEmpty(t, img.GetScan())

	ts.waitUntilLog(ctx, namespaces.StackRox, sensorPodLabels, sensorContainer, "contain the image scan",
		containsLineMatching(regexp.MustCompile(reStr)),
	)
}

// scanImage wraps retryable errors to facilitate retries.
func scanImage(ctx context.Context, service v1.ImageServiceClient, request *v1.ScanImageRequest) (*storage.Image, error) {
	img, err := service.ScanImage(ctx, request)
	if errors.Is(err, scan.ErrTooManyParallelScans) {
		err = retry.MakeRetryable(err)
	}

	return img, err
}

/*
common/delegatedregistry: 2024/08/29 22:10:47.499064 delegated_registry_handler.go:108: Info: Received scan request: "request_id:\"dcee0e6c-57a3-4ba9-b17c-bc0a8ad58f6f\" image_name:\"registry.access.redhat.com/ubi9/ubi:9.4-1181\" force:true "
common/delegatedregistry: 2024/08/29 22:10:52.499749 delegated_registry_handler.go:131: Error: Scan failed for req "dcee0e6c-57a3-4ba9-b17c-bc0a8ad58f6f" image "registry.access.redhat.com/ubi9/ubi:9.4-1181": too many parallel scans to local scanner
*/
