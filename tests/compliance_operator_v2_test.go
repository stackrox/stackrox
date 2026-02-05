//go:build compliance

package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis"
	complianceoperatorv1 "github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/service"
	v1 "github.com/stackrox/rox/generated/api/v1"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/protoconv/schedule"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	extscheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	cached "k8s.io/client-go/discovery/cached"
	cgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	coNamespaceV2   = "openshift-compliance"
	defaultTimeout  = 120 * time.Second
	defaultInterval = 5 * time.Second
)

var (
	initialProfiles = []string{"ocp4-cis"}
	updatedProfiles = []string{"ocp4-high", "ocp4-cis-node"}
	initialSchedule = &v2.Schedule{
		Hour:         12,
		Minute:       0,
		IntervalType: v2.Schedule_DAILY,
		Interval: &v2.Schedule_DaysOfWeek_{
			DaysOfWeek: &v2.Schedule_DaysOfWeek{
				Days: []int32{0, 1, 2, 3, 4, 5, 6},
			},
		},
	}
	updatedSchedule = &v2.Schedule{
		Hour:         1,
		Minute:       30,
		IntervalType: v2.Schedule_DAILY,
		Interval: &v2.Schedule_DaysOfWeek_{
			DaysOfWeek: &v2.Schedule_DaysOfWeek{
				Days: []int32{2, 3, 4},
			},
		},
	}
)

func createDynamicClient(t testutils.T) dynclient.Client {
	restCfg := getConfig(t)
	restCfg.WarningHandler = rest.NoWarnings{}
	k8sClient := createK8sClient(t)

	k8sScheme := runtime.NewScheme()

	err := cgoscheme.AddToScheme(k8sScheme)
	require.NoError(t, err, "error adding Kubernetes Scheme to client")

	err = extscheme.AddToScheme(k8sScheme)
	require.NoError(t, err, "error adding Kubernetes Scheme to client")

	cachedClientDiscovery := cached.NewMemCacheClient(k8sClient.Discovery())
	restMapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedClientDiscovery)
	restMapper.Reset()

	client, err := dynclient.New(
		restCfg,
		dynclient.Options{
			Scheme: k8sScheme,
			Mapper: restMapper,
		},
	)
	require.NoError(t, err, "failed to create dynamic client")

	// Add all the Compliance Operator schemes to the client so we can use
	// it for dealing with the Compliance Operator directly.
	err = apis.AddToScheme(client.Scheme())
	require.NoError(t, err, "failed to add Compliance Operator schemes to client")
	return client
}

func waitForComplianceSuiteToComplete(t *testing.T, client dynclient.Client, suiteName string, interval, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	t.Logf("Waiting for ComplianceSuite %s to reach DONE phase", suiteName)
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		callCtx, callCancel := context.WithTimeout(ctx, interval)
		defer callCancel()

		var suite complianceoperatorv1.ComplianceSuite
		err := client.Get(callCtx,
			types.NamespacedName{Name: suiteName, Namespace: "openshift-compliance"},
			&suite,
		)
		require.NoError(c, err)

		require.Equal(c, complianceoperatorv1.PhaseDone, suite.Status.Phase,
			"ComplianceSuite %s not DONE: is in %s phase", suiteName, suite.Status.Phase)
	}, timeout, interval)
	t.Logf("ComplianceSuite %s has reached DONE phase", suiteName)
}

func deleteResource[T any, PT interface {
	dynclient.Object
	*T
}](ctx context.Context, t *testing.T, client dynclient.Client, name, namespace string) {
	key := types.NamespacedName{Name: name, Namespace: namespace}

	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		var obj T
		ptr := PT(&obj)
		ptr.SetName(name)
		ptr.SetNamespace(namespace)
		err := client.Delete(ctx, ptr)
		if err != nil && !errors2.IsNotFound(err) {
			t.Logf("failed to delete %T %s/%s: %v", ptr, namespace, name, err)
		}
		err = client.Get(ctx, key, ptr)
		require.True(c, errors2.IsNotFound(err), "%T %s/%s still exists", ptr, namespace, name)
	}, defaultTimeout, defaultInterval)
}

func cleanUpResources(ctx context.Context, t *testing.T, client dynclient.Client, resourceName string, namespace string) {
	deleteResource[complianceoperatorv1.ScanSetting](ctx, t, client, resourceName, namespace)
	deleteResource[complianceoperatorv1.ScanSettingBinding](ctx, t, client, resourceName, namespace)
}

func assertResourceDoesNotExist[T any, PT interface {
	dynclient.Object
	*T
}](ctx context.Context, t testutils.T, client dynclient.Client, name, namespace string) {
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		var obj T
		err := client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, PT(&obj))
		require.True(c, errors2.IsNotFound(err), "%T %s/%s still exists", obj, namespace, name)
	}, defaultTimeout, defaultInterval)
}

func assertScanSetting(ctx context.Context, t testutils.T, client dynclient.Client, name, namespace string, scanConfig *v2.ComplianceScanConfiguration) {
	scanSetting := &complianceoperatorv1.ScanSetting{}
	err := client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, scanSetting)
	require.NoError(t, err, "ScanSetting %s/%s does not exist", namespace, name)

	cron, err := schedule.ConvertToCronTab(service.ConvertV2ScheduleToProto(scanConfig.GetScanConfig().GetScanSchedule()))
	require.NoError(t, err)
	assert.Equal(t, scanConfig.GetScanName(), scanSetting.GetName())
	assert.Equal(t, cron, scanSetting.ComplianceSuiteSettings.Schedule)
	require.Contains(t, scanSetting.GetLabels(), "app.kubernetes.io/name")
	assert.Equal(t, scanSetting.GetLabels()["app.kubernetes.io/name"], "stackrox")
	require.Contains(t, scanSetting.GetAnnotations(), "owner")
	assert.Equal(t, scanSetting.GetAnnotations()["owner"], "stackrox")
}

func assertScanSettingBinding(ctx context.Context, t testutils.T, client dynclient.Client, name, namespace string, scanConfig *v2.ComplianceScanConfiguration) {
	scanSettingBinding := &complianceoperatorv1.ScanSettingBinding{}
	err := client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, scanSettingBinding)
	require.NoError(t, err, "ScanSettingBinding %s/%s does not exist", namespace, name)

	assert.Equal(t, scanConfig.GetScanName(), scanSettingBinding.GetName())
	for _, profile := range scanSettingBinding.Profiles {
		assert.Contains(t, scanConfig.GetScanConfig().GetProfiles(), profile.Name)
	}
	require.Contains(t, scanSettingBinding.Labels, "app.kubernetes.io/name")
	assert.Equal(t, scanSettingBinding.Labels["app.kubernetes.io/name"], "stackrox")
	require.Contains(t, scanSettingBinding.Annotations, "owner")
	assert.Equal(t, scanSettingBinding.Annotations["owner"], "stackrox")
}

func waitForDeploymentReady(ctx context.Context, t *testing.T, client dynclient.Client, name string, namespace string, numReplicas int32) {
	require.Eventually(t, func() bool {
		deployment := &appsv1.Deployment{}
		return client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, deployment) == nil && deployment.Status.ReadyReplicas == numReplicas
	}, defaultTimeout, defaultInterval)
}

func TestComplianceV2CentralSendsScanConfiguration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dynClient := createDynamicClient(t)

	conn := centralgrpc.GRPCConnectionToCentral(t)
	// Create the ScanConfiguration service
	scanConfigService := v2.NewComplianceScanConfigurationServiceClient(conn)

	// Get cluster ID
	serviceCluster := v1.NewClustersServiceClient(conn)
	clusters, err := serviceCluster.GetClusters(ctx, &v1.GetClustersRequest{})
	assert.NoError(t, err)
	require.Greater(t, len(clusters.GetClusters()), 0)
	clusterID := clusters.GetClusters()[0].GetId()

	// Create local scan config with UUID-based name for test isolation
	scanName := fmt.Sprintf("sync-test-%s", uuid.NewV4().String())
	scanConfig := v2.ComplianceScanConfiguration{
		ScanName: scanName,
		Clusters: []string{clusterID},
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			Description:  scanName,
			OneTimeScan:  false,
			Profiles:     initialProfiles,
			ScanSchedule: initialSchedule,
		},
	}

	// Create ScanConfig in Central
	res, err := scanConfigService.CreateComplianceScanConfiguration(ctx, &scanConfig)
	assert.NoError(t, err)

	// Cleanup just in case the test fails
	t.Cleanup(func() {
		reqDelete := &v2.ResourceByID{
			Id: res.GetId(),
		}
		_, _ = scanConfigService.DeleteComplianceScanConfiguration(ctx, reqDelete)
		cleanUpResources(ctx, t, dynClient, scanName, coNamespaceV2)
	})

	// Assert the ScanSetting and the ScanSettingBinding are created
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		assertScanSetting(ctx, wrapCollectT(t, c), dynClient, scanName, coNamespaceV2, &scanConfig)
		assertScanSettingBinding(ctx, wrapCollectT(t, c), dynClient, scanName, coNamespaceV2, &scanConfig)
	}, defaultTimeout, defaultInterval)

	// Update the ScanConfig in Central
	scanConfig.Id = res.GetId()
	scanConfig.ScanConfig.Profiles = updatedProfiles
	scanConfig.ScanConfig.ScanSchedule = updatedSchedule
	_, err = scanConfigService.UpdateComplianceScanConfiguration(ctx, &scanConfig)
	assert.NoError(t, err)

	// Assert the ScanSetting and the ScanSettingBinding are updated
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		assertScanSetting(ctx, wrapCollectT(t, c), dynClient, scanName, coNamespaceV2, &scanConfig)
		assertScanSettingBinding(ctx, wrapCollectT(t, c), dynClient, scanName, coNamespaceV2, &scanConfig)
	}, defaultTimeout, defaultInterval)

	// Delete the ScanConfig in Central
	reqDelete := &v2.ResourceByID{
		Id: res.GetId(),
	}
	_, err = scanConfigService.DeleteComplianceScanConfiguration(ctx, reqDelete)

	// Assert the ScanSetting and the ScanSettingBinding are deleted
	assertResourceDoesNotExist[complianceoperatorv1.ScanSetting](ctx, t, dynClient, scanName, coNamespaceV2)
	assertResourceDoesNotExist[complianceoperatorv1.ScanSettingBinding](ctx, t, dynClient, scanName, coNamespaceV2)
}

// ACS API test suite for integration testing for the Compliance Operator.
func TestComplianceV2Integration(t *testing.T) {
	t.Parallel()
	resp := getIntegrations(t)
	assert.Len(t, resp.GetIntegrations(), 1, "failed to assert there is only a single compliance integration")
	assert.Equal(t, resp.GetIntegrations()[0].GetClusterName(), "remote", "failed to find integration for cluster called \"remote\"")
	assert.Equal(t, resp.GetIntegrations()[0].GetNamespace(), "openshift-compliance", "failed to find integration for \"openshift-compliance\" namespace")
}

func TestComplianceV2ProfileGet(t *testing.T) {
	t.Parallel()
	conn := centralgrpc.GRPCConnectionToCentral(t)
	client := v2.NewComplianceProfileServiceClient(conn)

	// Get the clusters
	resp := getIntegrations(t)
	assert.Len(t, resp.GetIntegrations(), 1, "failed to assert there is only a single compliance integration")

	// Get the profiles for the cluster
	clusterID := resp.GetIntegrations()[0].GetClusterId()
	profileList, err := client.ListComplianceProfiles(context.TODO(), &v2.ProfilesForClusterRequest{ClusterId: clusterID})
	assert.Greater(t, len(profileList.GetProfiles()), 0, "failed to assert the cluster has profiles")

	// Now take the ID from one of the cluster profiles to get the specific profile.
	profile, err := client.GetComplianceProfile(context.TODO(), &v2.ResourceByID{Id: profileList.GetProfiles()[0].GetId()})
	if err != nil {
		t.Fatal(err)
	}
	assert.Greater(t, len(profile.GetRules()), 0, "failed to verify the selected profile contains any rules")
}

func TestComplianceV2ProfileGetSummaries(t *testing.T) {
	t.Parallel()
	conn := centralgrpc.GRPCConnectionToCentral(t)
	client := v2.NewComplianceProfileServiceClient(conn)

	// Get the clusters
	resp := getIntegrations(t)
	assert.Len(t, resp.GetIntegrations(), 1, "failed to assert there is only a single compliance integration")

	// Get the profiles for the cluster
	clusterID := resp.GetIntegrations()[0].GetClusterId()
	profileSummaries, err := client.ListProfileSummaries(context.TODO(), &v2.ClustersProfileSummaryRequest{ClusterIds: []string{clusterID}})
	assert.NoError(t, err)
	assert.Greater(t, len(profileSummaries.GetProfiles()), 0, "failed to assert the cluster has profiles")
}

// Helper to get the integrations as the cluster id is needed in many API calls
func getIntegrations(t *testing.T) *v2.ListComplianceIntegrationsResponse {
	conn := centralgrpc.GRPCConnectionToCentral(t)
	client := v2.NewComplianceIntegrationServiceClient(conn)

	q := &v2.RawQuery{Query: ""}
	resp, err := client.ListComplianceIntegrations(context.TODO(), q)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, resp.GetIntegrations(), 1, "failed to assert there is only a single compliance integration")

	return resp
}

func TestComplianceV2CreateGetScanConfigurations(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dynClient := createDynamicClient(t)
	conn := centralgrpc.GRPCConnectionToCentral(t)
	scanConfigService := v2.NewComplianceScanConfigurationServiceClient(conn)
	serviceCluster := v1.NewClustersServiceClient(conn)
	clusters, err := serviceCluster.GetClusters(ctx, &v1.GetClustersRequest{})
	assert.NoError(t, err)
	clusterID := clusters.GetClusters()[0].GetId()
	testName := fmt.Sprintf("create-get-%s", uuid.NewV4().String())
	req := &v2.ComplianceScanConfiguration{
		ScanName: testName,
		Id:       "",
		Clusters: []string{clusterID},
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan: false,
			Profiles:    []string{"rhcos4-moderate"},
			Description: "test config",
			ScanSchedule: &v2.Schedule{
				IntervalType: 1,
				Hour:         15,
				Minute:       0,
				Interval: &v2.Schedule_DaysOfWeek_{
					DaysOfWeek: &v2.Schedule_DaysOfWeek{
						Days: []int32{1, 2, 3, 4, 5, 6},
					},
				},
			},
		},
	}

	resp, err := scanConfigService.CreateComplianceScanConfiguration(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, req.GetScanName(), resp.GetScanName())
	t.Cleanup(func() {
		_ = deleteScanConfig(ctx, resp.GetId(), scanConfigService)
		cleanUpResources(ctx, t, dynClient, testName, coNamespaceV2)
	})

	query := &v2.RawQuery{Query: ""}
	scanConfigs, err := scanConfigService.ListComplianceScanConfigurations(ctx, query)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(scanConfigs.GetConfigurations()), 1)
	assert.GreaterOrEqual(t, scanConfigs.GetTotalCount(), int32(1))

	serviceResult := v2.NewComplianceResultsServiceClient(conn)
	query = &v2.RawQuery{Query: ""}
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		results, err := serviceResult.GetComplianceScanResults(ctx, query)
		require.NoError(c, err)

		resultsList := results.GetScanResults()
		var found bool
		for _, result := range resultsList {
			if result.GetScanName() == testName {
				found = true
				break
			}
		}
		require.True(c, found, "scan result not found for %s", testName)
	}, 10*time.Minute, 30*time.Second)

	// Create a different scan configuration with the same profile
	duplicateTestName := fmt.Sprintf("create-get-dup-%s", uuid.NewV4().String())
	duplicateProfileReq := &v2.ComplianceScanConfiguration{
		ScanName: duplicateTestName,
		Id:       "",
		Clusters: []string{clusterID},
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan: false,
			Profiles:    []string{"rhcos4-moderate"},
			Description: "test config with duplicate profile",
			ScanSchedule: &v2.Schedule{
				IntervalType: 1,
				Hour:         15,
				Minute:       0,
				Interval: &v2.Schedule_DaysOfWeek_{
					DaysOfWeek: &v2.Schedule_DaysOfWeek{
						Days: []int32{1, 2, 3, 4, 5, 6},
					},
				},
			},
		},
	}

	// Verify that the duplicate profile was not created and the error message is correct
	_, err = scanConfigService.CreateComplianceScanConfiguration(ctx, duplicateProfileReq)
	assert.Contains(t, err.Error(), "already uses profile")

	query = &v2.RawQuery{Query: ""}
	scanConfigs, err = scanConfigService.ListComplianceScanConfigurations(ctx, query)
	assert.NoError(t, err)
	// Verify the original config exists but duplicate was not created
	assert.NotEmpty(t, getscanConfigID(testName, scanConfigs.GetConfigurations()), "expected original scan config %s to exist", testName)
	assert.Empty(t, getscanConfigID(duplicateTestName, scanConfigs.GetConfigurations()), "expected duplicate scan config %s to not exist", duplicateTestName)

	// Create a scan configuration with profiles with different products (rhcos, cis-node).
	// This should be valid in version >= 4.9
	differentProductProfileTestName := fmt.Sprintf("create-get-diffprod-%s", uuid.NewV4().String())
	differentProductProfileReq := &v2.ComplianceScanConfiguration{
		ScanName: differentProductProfileTestName,
		Id:       "",
		Clusters: []string{clusterID},
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan: false,
			Profiles:    []string{"rhcos4-stig", "ocp4-cis-node"},
			Description: "test config with invalid profiles",
			ScanSchedule: &v2.Schedule{
				IntervalType: 1,
				Hour:         15,
				Minute:       0,
				Interval: &v2.Schedule_DaysOfWeek_{
					DaysOfWeek: &v2.Schedule_DaysOfWeek{
						Days: []int32{1, 2, 3, 4, 5, 6},
					},
				},
			},
		},
	}

	res, err := scanConfigService.CreateComplianceScanConfiguration(ctx, differentProductProfileReq)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = deleteScanConfig(ctx, res.GetId(), scanConfigService)
		cleanUpResources(ctx, t, dynClient, differentProductProfileTestName, coNamespaceV2)
	})

	query = &v2.RawQuery{Query: ""}
	scanConfigs, err = scanConfigService.ListComplianceScanConfigurations(ctx, query)
	assert.NoError(t, err)
	// Verify both scan configs exist
	assert.NotEmpty(t, getscanConfigID(testName, scanConfigs.GetConfigurations()), "expected original scan config %s to exist", testName)
	assert.NotEmpty(t, getscanConfigID(differentProductProfileTestName, scanConfigs.GetConfigurations()), "expected different product scan config %s to exist", differentProductProfileTestName)
}

func TestComplianceV2UpdateScanConfigurations(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dynClient := createDynamicClient(t)
	conn := centralgrpc.GRPCConnectionToCentral(t)
	scanConfigService := v2.NewComplianceScanConfigurationServiceClient(conn)
	serviceCluster := v1.NewClustersServiceClient(conn)
	clusters, err := serviceCluster.GetClusters(ctx, &v1.GetClustersRequest{})
	assert.NoError(t, err)
	require.Greater(t, len(clusters.GetClusters()), 0)
	clusterID := clusters.GetClusters()[0].GetId()

	// Create a scan configuration
	scanName := fmt.Sprintf("update-%s", uuid.NewV4().String())
	req := &v2.ComplianceScanConfiguration{
		ScanName: scanName,
		Id:       "",
		Clusters: []string{clusterID},
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan: false,
			Profiles:    []string{"ocp4-moderate"},
			Description: "test config",
			ScanSchedule: &v2.Schedule{
				IntervalType: 1,
				Hour:         15,
				Minute:       0,
				Interval: &v2.Schedule_DaysOfWeek_{
					DaysOfWeek: &v2.Schedule_DaysOfWeek{
						Days: []int32{1, 2, 3, 4, 5, 6},
					},
				},
			},
		},
	}

	resp, err := scanConfigService.CreateComplianceScanConfiguration(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, req.GetScanName(), resp.GetScanName())
	t.Cleanup(func() {
		_ = deleteScanConfig(ctx, resp.GetId(), scanConfigService)
		cleanUpResources(ctx, t, dynClient, req.GetScanName(), coNamespaceV2)
	})

	query := &v2.RawQuery{Query: ""}
	scanConfigs, err := scanConfigService.ListComplianceScanConfigurations(ctx, query)
	assert.NoError(t, err)
	require.GreaterOrEqual(t, len(scanConfigs.GetConfigurations()), 1)
	assert.GreaterOrEqual(t, scanConfigs.GetTotalCount(), int32(1))

	// Assert the ScanSetting and the ScanSettingBinding are created
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		assertScanSetting(ctx, wrapCollectT(t, c), dynClient, scanName, coNamespaceV2, req)
		assertScanSettingBinding(ctx, wrapCollectT(t, c), dynClient, scanName, coNamespaceV2, req)
	}, defaultTimeout, defaultInterval)

	// Update the scan configuration
	updateReq := req.CloneVT()
	updateReq.Id = resp.GetId()
	updateReq.ScanConfig.ScanSchedule = &v2.Schedule{
		IntervalType: 1,
		Hour:         12,
		Minute:       30,
		Interval: &v2.Schedule_DaysOfWeek_{
			DaysOfWeek: &v2.Schedule_DaysOfWeek{
				Days: []int32{2, 4, 6},
			},
		},
	}
	updateReq.ScanConfig.Profiles = []string{"ocp4-moderate-node"}
	_, err = scanConfigService.UpdateComplianceScanConfiguration(ctx, updateReq)
	assert.NoError(t, err)

	query = &v2.RawQuery{Query: ""}
	scanConfigs, err = scanConfigService.ListComplianceScanConfigurations(ctx, query)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(scanConfigs.GetConfigurations()), 1)
	assert.GreaterOrEqual(t, scanConfigs.GetTotalCount(), int32(1))

	// Assert the ScanSetting and the ScanSettingBinding are updated
	assert.EventuallyWithT(t, func(c *assert.CollectT) {
		assertScanSetting(ctx, wrapCollectT(t, c), dynClient, scanName, coNamespaceV2, updateReq)
		assertScanSettingBinding(ctx, wrapCollectT(t, c), dynClient, scanName, coNamespaceV2, updateReq)
	}, defaultTimeout, defaultInterval)
}

func TestComplianceV2DeleteComplianceScanConfigurations(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dynClient := createDynamicClient(t)
	conn := centralgrpc.GRPCConnectionToCentral(t)
	scanConfigService := v2.NewComplianceScanConfigurationServiceClient(conn)
	// Retrieve the results from the scan configuration once the scan is complete
	serviceCluster := v1.NewClustersServiceClient(conn)
	clusters, err := serviceCluster.GetClusters(ctx, &v1.GetClustersRequest{})
	assert.NoError(t, err)

	clusterID := clusters.GetClusters()[0].GetId()
	testName := fmt.Sprintf("delete-%s", uuid.NewV4().String())
	req := &v2.ComplianceScanConfiguration{
		ScanName: testName,
		Id:       "",
		Clusters: []string{clusterID},
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan: false,
			Profiles:    []string{"rhcos4-high"},
			Description: "test config",
			ScanSchedule: &v2.Schedule{
				IntervalType: 1,
				Hour:         15,
				Minute:       0,
				Interval: &v2.Schedule_DaysOfWeek_{
					DaysOfWeek: &v2.Schedule_DaysOfWeek{
						Days: []int32{1, 2, 3, 4, 5, 6},
					},
				},
			},
		},
	}

	resp, err := scanConfigService.CreateComplianceScanConfiguration(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, req.GetScanName(), resp.GetScanName())
	t.Cleanup(func() {
		_ = deleteScanConfig(ctx, resp.GetId(), scanConfigService)
		cleanUpResources(ctx, t, dynClient, testName, coNamespaceV2)
	})

	query := &v2.RawQuery{Query: ""}
	scanConfigs, err := scanConfigService.ListComplianceScanConfigurations(ctx, query)
	configs := scanConfigs.GetConfigurations()
	scanconfigID := getscanConfigID(testName, configs)
	reqDelete := &v2.ResourceByID{
		Id: scanconfigID,
	}
	_, err = scanConfigService.DeleteComplianceScanConfiguration(ctx, reqDelete)
	assert.NoError(t, err)

	// Verify scan configuration no longer exists
	scanConfigs, err = scanConfigService.ListComplianceScanConfigurations(ctx, query)
	configs = scanConfigs.GetConfigurations()
	scanconfigID = getscanConfigID(testName, configs)
	assert.Empty(t, scanconfigID)
}

func TestComplianceV2ComplianceObjectMetadata(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	dynClient := createDynamicClient(t)
	conn := centralgrpc.GRPCConnectionToCentral(t)
	scanConfigService := v2.NewComplianceScanConfigurationServiceClient(conn)
	serviceCluster := v1.NewClustersServiceClient(conn)
	clusters, err := serviceCluster.GetClusters(ctx, &v1.GetClustersRequest{})
	assert.NoError(t, err)
	clusterID := clusters.GetClusters()[0].GetId()
	testName := fmt.Sprintf("metadata-%s", uuid.NewV4().String())
	req := &v2.ComplianceScanConfiguration{
		ScanName: testName,
		Id:       "",
		Clusters: []string{clusterID},
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan: false,
			Profiles:    []string{"rhcos4-nerc-cip"},
			Description: "test config",
			ScanSchedule: &v2.Schedule{
				IntervalType: 1,
				Hour:         15,
				Minute:       0,
				Interval: &v2.Schedule_DaysOfWeek_{
					DaysOfWeek: &v2.Schedule_DaysOfWeek{
						Days: []int32{1, 2, 3, 4, 5, 6},
					},
				},
			},
		},
	}

	resp, err := scanConfigService.CreateComplianceScanConfiguration(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, req.GetScanName(), resp.GetScanName())
	t.Cleanup(func() {
		_ = deleteScanConfig(ctx, resp.GetId(), scanConfigService)
		cleanUpResources(ctx, t, dynClient, testName, coNamespaceV2)
	})

	query := &v2.RawQuery{Query: ""}
	scanConfigs, err := scanConfigService.ListComplianceScanConfigurations(ctx, query)
	configs := scanConfigs.GetConfigurations()
	_ = getscanConfigID(testName, configs) // verify config exists

	// Ensure the ScanSetting and ScanSettingBinding have ACS metadata
	var scanSetting complianceoperatorv1.ScanSetting
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		callCtx, callCancel := context.WithTimeout(ctx, 10*time.Second)
		defer callCancel()

		err := dynClient.Get(callCtx,
			types.NamespacedName{Name: testName, Namespace: "openshift-compliance"},
			&scanSetting,
		)
		require.NoError(c, err)
	}, defaultTimeout, defaultInterval)

	assert.Contains(t, scanSetting.Labels, "app.kubernetes.io/name")
	assert.Equal(t, scanSetting.Labels["app.kubernetes.io/name"], "stackrox")
	assert.Contains(t, scanSetting.Annotations, "owner")
	assert.Equal(t, scanSetting.Annotations["owner"], "stackrox")

	var scanSettingBinding complianceoperatorv1.ScanSetting
	err = dynClient.Get(context.TODO(), types.NamespacedName{Name: testName, Namespace: "openshift-compliance"}, &scanSettingBinding)
	require.NoError(t, err, "failed to get ScanSettingBinding %s", testName)
	assert.Contains(t, scanSettingBinding.Labels, "app.kubernetes.io/name")
	assert.Equal(t, scanSettingBinding.Labels["app.kubernetes.io/name"], "stackrox")
	assert.Contains(t, scanSettingBinding.Annotations, "owner")
	assert.Equal(t, scanSettingBinding.Annotations["owner"], "stackrox")
}

func deleteScanConfig(ctx context.Context, scanID string, service v2.ComplianceScanConfigurationServiceClient) error {
	req := &v2.ResourceByID{
		Id: scanID,
	}
	_, err := service.DeleteComplianceScanConfiguration(ctx, req)
	return err
}

func getscanConfigID(configName string, scanConfigs []*v2.ComplianceScanConfigurationStatus) string {
	configID := ""
	for i := 0; i < len(scanConfigs); i++ {
		if scanConfigs[i].GetScanName() == configName {
			configID = scanConfigs[i].GetId()
		}
	}
	return configID
}

func TestComplianceV2ScheduleRescan(t *testing.T) {
	t.Parallel()
	dynClient := createDynamicClient(t)
	conn := centralgrpc.GRPCConnectionToCentral(t)
	client := v2.NewComplianceScanConfigurationServiceClient(conn)
	integrationClient := v2.NewComplianceIntegrationServiceClient(conn)
	resp, err := integrationClient.ListComplianceIntegrations(context.TODO(), &v2.RawQuery{Query: ""})
	if err != nil {
		t.Fatal(err)
	}
	clusterId := resp.GetIntegrations()[0].GetClusterId()

	scanConfigName := fmt.Sprintf("schedule-rescan-%s", uuid.NewV4().String())
	sc := v2.ComplianceScanConfiguration{
		ScanName: scanConfigName,
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan: false,
			Profiles:    []string{"ocp4-e8"},
			ScanSchedule: &v2.Schedule{
				IntervalType: 3,
				Hour:         0,
				Minute:       0,
			},
			Description: "Scan schedule for the Australian Essential Eight profile to run daily.",
		},
		Clusters: []string{clusterId},
	}
	scanConfig, err := client.CreateComplianceScanConfiguration(context.TODO(), &sc)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = client.DeleteComplianceScanConfiguration(context.TODO(), &v2.ResourceByID{Id: scanConfig.GetId()})
		cleanUpResources(context.Background(), t, dynClient, scanConfigName, coNamespaceV2)
	})

	waitForComplianceSuiteToComplete(t, dynClient, scanConfig.GetScanName(), 30*time.Second, 5*time.Minute)

	// Invoke a rescan
	_, err = client.RunComplianceScanConfiguration(context.TODO(), &v2.ResourceByID{Id: scanConfig.GetId()})
	require.NoError(t, err, "failed to rerun scan schedule %s", scanConfigName)

	// Assert the scan is rerunning on the cluster using the Compliance Operator CRDs
	waitForComplianceSuiteToComplete(t, dynClient, scanConfig.GetScanName(), 30*time.Second, 5*time.Minute)
}
