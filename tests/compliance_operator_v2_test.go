//go:build compliance

package tests

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis"
	complianceoperatorv1 "github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/service"
	v1 "github.com/stackrox/rox/generated/api/v1"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/protoconv/schedule"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingV1 "k8s.io/api/autoscaling/v1"
	extscheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	cached "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/kubernetes"
	cgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	coNamespaceV2     = "openshift-compliance"
	stackroxNamespace = "stackrox"
)

var (
	scanName        = "sync-test"
	initialProfiles = []string{"ocp4-cis"}
	updatedProfiles = []string{"ocp4-cis-1-4", "ocp4-cis-node-1-4"}
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
	scanConfig = v2.ComplianceScanConfiguration{
		ScanName: scanName,
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			Description:  scanName,
			OneTimeScan:  false,
			Profiles:     initialProfiles,
			ScanSchedule: initialSchedule,
		},
	}
)

func scaleToN(ctx context.Context, client kubernetes.Interface, deploymentName string, namespace string, replicas int32) (err error) {
	scaleRequest := &autoscalingV1.Scale{
		Spec: autoscalingV1.ScaleSpec{
			Replicas: replicas,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: namespace,
		},
	}

	_, err = client.AppsV1().Deployments(namespace).UpdateScale(ctx, deploymentName, scaleRequest, metav1.UpdateOptions{})
	if err != nil {
		return retry.MakeRetryable(err)
	}
	return nil
}

func createDynamicClient(t *testing.T) dynclient.Client {
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

func waitForComplianceSuiteToComplete(t *testing.T, suiteName string, interval, timeout time.Duration) {
	client := createDynamicClient(t)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	log.Info("Waiting for ComplianceSuite to reach DONE phase")
	for {
		select {
		case <-ticker.C:
			var suite complianceoperatorv1.ComplianceSuite
			err := client.Get(context.TODO(), types.NamespacedName{Name: suiteName, Namespace: "openshift-compliance"}, &suite)
			require.NoError(t, err, "failed to get ComplianceSuite %s", suiteName)

			if suite.Status.Phase == "DONE" {
				log.Infof("ComplianceSuite %s reached DONE phase", suiteName)
				return
			}
			log.Infof("ComplianceSuite %s is in %s phase", suiteName, suite.Status.Phase)
		case <-timer.C:
			t.Fatalf("Timed out waiting for ComplianceSuite to complete")
		}
	}
}

func cleanUpResources(ctx context.Context, t *testing.T, resourceName string, namespace string) {
	client := createDynamicClient(t)
	scanSetting := &complianceoperatorv1.ScanSetting{}
	scanSettingBinding := &complianceoperatorv1.ScanSettingBinding{}
	err := client.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: namespace}, scanSetting)
	if err == nil {
		_ = client.Delete(ctx, scanSetting)
	}
	err = client.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: namespace}, scanSettingBinding)
	if err == nil {
		_ = client.Delete(ctx, scanSettingBinding)
	}
}

func assertResourceDoesExist(ctx context.Context, t *testing.T, resourceName string, namespace string, obj dynclient.Object) dynclient.Object {
	client := createDynamicClient(t)
	require.Eventually(t, func() bool {
		return client.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: namespace}, obj) == nil
	}, 60*time.Second, 10*time.Millisecond)
	return obj
}

func assertResourceWasUpdated(ctx context.Context, t *testing.T, resourceName string, namespace string, obj dynclient.Object) dynclient.Object {
	client := createDynamicClient(t)
	oldResourceVersion := obj.GetResourceVersion()
	require.Eventually(t, func() bool {
		return client.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: namespace}, obj) == nil && obj.GetResourceVersion() != oldResourceVersion
	}, 60*time.Second, 10*time.Millisecond)
	return obj
}

func assertResourceDoesNotExist(ctx context.Context, t *testing.T, resourceName string, namespace string, obj dynclient.Object) {
	client := createDynamicClient(t)
	require.Eventually(t, func() bool {
		err := client.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: namespace}, obj)
		return errors2.IsNotFound(err)
	}, 60*time.Second, 10*time.Millisecond)
}

func assertScanSetting(t *testing.T, scanConfig v2.ComplianceScanConfiguration, scanSetting *complianceoperatorv1.ScanSetting) {
	require.NotNil(t, scanSetting)
	cron, err := schedule.ConvertToCronTab(service.ConvertV2ScheduleToProto(scanConfig.GetScanConfig().GetScanSchedule()))
	require.NoError(t, err)
	assert.Equal(t, scanConfig.GetScanName(), scanSetting.GetName())
	assert.Equal(t, cron, scanSetting.ComplianceSuiteSettings.Schedule)
	assert.Contains(t, scanSetting.Labels, "app.kubernetes.io/name")
	assert.Equal(t, scanSetting.Labels["app.kubernetes.io/name"], "stackrox")
	assert.Contains(t, scanSetting.Annotations, "owner")
	assert.Equal(t, scanSetting.Annotations["owner"], "stackrox")
}

func assertScanSettingBinding(t *testing.T, scanConfig v2.ComplianceScanConfiguration, scanSettingBinding *complianceoperatorv1.ScanSettingBinding) {
	require.NotNil(t, scanSettingBinding)
	assert.Equal(t, scanConfig.GetScanName(), scanSettingBinding.GetName())
	for _, profile := range scanSettingBinding.Profiles {
		assert.Contains(t, scanConfig.GetScanConfig().GetProfiles(), profile.Name)
	}
	assert.Contains(t, scanSettingBinding.Labels, "app.kubernetes.io/name")
	assert.Equal(t, scanSettingBinding.Labels["app.kubernetes.io/name"], "stackrox")
	assert.Contains(t, scanSettingBinding.Annotations, "owner")
	assert.Equal(t, scanSettingBinding.Annotations["owner"], "stackrox")
}

func waitForDeploymentReady(ctx context.Context, t *testing.T, name string, namespace string, numReplicas int32) {
	client := createDynamicClient(t)
	require.Eventually(t, func() bool {
		deployment := &appsv1.Deployment{}
		return client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, deployment) == nil && deployment.Status.ReadyReplicas == numReplicas
	}, 60*time.Second, 10*time.Millisecond)
}

func TestComplianceV2CentralSendsScanConfiguration(t *testing.T) {
	ctx := context.Background()
	k8sClient := createK8sClient(t)

	conn := centralgrpc.GRPCConnectionToCentral(t)
	// Create the ScanConfiguration service
	scanConfigService := v2.NewComplianceScanConfigurationServiceClient(conn)

	// Get cluster ID
	serviceCluster := v1.NewClustersServiceClient(conn)
	clusters, err := serviceCluster.GetClusters(ctx, &v1.GetClustersRequest{})
	assert.NoError(t, err)
	require.Greater(t, len(clusters.GetClusters()), 0)
	clusterID := clusters.GetClusters()[0].GetId()

	// Set the cluster ID
	scanConfig.Clusters = []string{clusterID}

	// Scale down Sensor
	assert.NoError(t, scaleToN(ctx, k8sClient, "sensor", stackroxNamespace, 0))
	waitForDeploymentReady(ctx, t, "sensor", stackroxNamespace, 0)

	// Create ScanConfig in Central
	res, err := scanConfigService.CreateComplianceScanConfiguration(ctx, &scanConfig)
	assert.NoError(t, err)

	// Cleanup just in case the test fails
	t.Cleanup(func() {
		reqDelete := &v2.ResourceByID{
			Id: res.GetId(),
		}
		_, _ = scanConfigService.DeleteComplianceScanConfiguration(ctx, reqDelete)
		cleanUpResources(ctx, t, scanName, coNamespaceV2)
	})

	// Scale up Sensor
	assert.NoError(t, scaleToN(ctx, k8sClient, "sensor", stackroxNamespace, 1))
	waitForDeploymentReady(ctx, t, "sensor", stackroxNamespace, 1)

	// Assert the ScanSetting and the ScanSettingBinding are created
	scanSetting := &complianceoperatorv1.ScanSetting{}
	scanSettingBinding := &complianceoperatorv1.ScanSettingBinding{}
	assertResourceDoesExist(ctx, t, scanName, coNamespaceV2, scanSetting)
	assertResourceDoesExist(ctx, t, scanName, coNamespaceV2, scanSettingBinding)
	assertScanSetting(t, scanConfig, scanSetting)
	assertScanSettingBinding(t, scanConfig, scanSettingBinding)

	// Scale down Sensor
	assert.NoError(t, scaleToN(ctx, k8sClient, "sensor", stackroxNamespace, 0))
	waitForDeploymentReady(ctx, t, "sensor", stackroxNamespace, 0)

	// Update the ScanConfig in Central
	scanConfig.Id = res.GetId()
	scanConfig.ScanConfig.Profiles = updatedProfiles
	scanConfig.ScanConfig.ScanSchedule = updatedSchedule
	_, err = scanConfigService.UpdateComplianceScanConfiguration(ctx, &scanConfig)
	assert.NoError(t, err)

	// Scale up Sensor
	assert.NoError(t, scaleToN(ctx, k8sClient, "sensor", stackroxNamespace, 1))
	waitForDeploymentReady(ctx, t, "sensor", stackroxNamespace, 1)

	// Assert the ScanSetting and the ScanSettingBinding are updated
	assertResourceWasUpdated(ctx, t, scanName, coNamespaceV2, scanSetting)
	assertResourceWasUpdated(ctx, t, scanName, coNamespaceV2, scanSettingBinding)
	assertScanSetting(t, scanConfig, scanSetting)
	assertScanSettingBinding(t, scanConfig, scanSettingBinding)

	// Scale down Sensor
	assert.NoError(t, scaleToN(ctx, k8sClient, "sensor", stackroxNamespace, 0))
	waitForDeploymentReady(ctx, t, "sensor", stackroxNamespace, 0)

	// Delete the ScanConfig in Central
	reqDelete := &v2.ResourceByID{
		Id: res.GetId(),
	}
	_, err = scanConfigService.DeleteComplianceScanConfiguration(ctx, reqDelete)

	// Scale up Sensor
	assert.NoError(t, scaleToN(ctx, k8sClient, "sensor", stackroxNamespace, 1))
	waitForDeploymentReady(ctx, t, "sensor", stackroxNamespace, 1)

	// Assert the ScanSetting and the ScanSettingBinding are deleted
	assertResourceDoesNotExist(ctx, t, scanName, coNamespaceV2, scanSetting)
	assertResourceDoesNotExist(ctx, t, scanName, coNamespaceV2, scanSettingBinding)
}

// ACS API test suite for integration testing for the Compliance Operator.
func TestComplianceV2Integration(t *testing.T) {
	resp := getIntegrations(t)
	assert.Len(t, resp.Integrations, 1, "failed to assert there is only a single compliance integration")
	assert.Equal(t, resp.Integrations[0].ClusterName, "remote", "failed to find integration for cluster called \"remote\"")
	assert.Equal(t, resp.Integrations[0].Namespace, "openshift-compliance", "failed to find integration for \"openshift-compliance\" namespace")
}

func TestComplianceV2ProfileGet(t *testing.T) {
	conn := centralgrpc.GRPCConnectionToCentral(t)
	client := v2.NewComplianceProfileServiceClient(conn)

	// Get the clusters
	resp := getIntegrations(t)
	assert.Len(t, resp.Integrations, 1, "failed to assert there is only a single compliance integration")

	// Get the profiles for the cluster
	clusterID := resp.Integrations[0].ClusterId
	profileList, err := client.ListComplianceProfiles(context.TODO(), &v2.ProfilesForClusterRequest{ClusterId: clusterID})
	assert.Greater(t, len(profileList.Profiles), 0, "failed to assert the cluster has profiles")

	// Now take the ID from one of the cluster profiles to get the specific profile.
	profile, err := client.GetComplianceProfile(context.TODO(), &v2.ResourceByID{Id: profileList.Profiles[0].Id})
	if err != nil {
		t.Fatal(err)
	}
	assert.Greater(t, len(profile.Rules), 0, "failed to verify the selected profile contains any rules")
}

func TestComplianceV2ProfileGetSummaries(t *testing.T) {
	conn := centralgrpc.GRPCConnectionToCentral(t)
	client := v2.NewComplianceProfileServiceClient(conn)

	// Get the clusters
	resp := getIntegrations(t)
	assert.Len(t, resp.Integrations, 1, "failed to assert there is only a single compliance integration")

	// Get the profiles for the cluster
	clusterID := resp.Integrations[0].ClusterId
	profileSummaries, err := client.ListProfileSummaries(context.TODO(), &v2.ClustersProfileSummaryRequest{ClusterIds: []string{clusterID}})
	assert.NoError(t, err)
	assert.Greater(t, len(profileSummaries.Profiles), 0, "failed to assert the cluster has profiles")
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
	assert.Len(t, resp.Integrations, 1, "failed to assert there is only a single compliance integration")

	return resp
}

func TestComplianceV2CreateGetScanConfigurations(t *testing.T) {
	ctx := context.Background()
	conn := centralgrpc.GRPCConnectionToCentral(t)
	scanConfigService := v2.NewComplianceScanConfigurationServiceClient(conn)
	serviceCluster := v1.NewClustersServiceClient(conn)
	clusters, err := serviceCluster.GetClusters(ctx, &v1.GetClustersRequest{})
	assert.NoError(t, err)
	clusterID := clusters.GetClusters()[0].GetId()
	testName := fmt.Sprintf("test-%s", uuid.NewV4().String())
	req := &v2.ComplianceScanConfiguration{
		ScanName: testName,
		Id:       "",
		Clusters: []string{clusterID},
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan: false,
			Profiles:    []string{"rhcos4-e8"},
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

	query := &v2.RawQuery{Query: ""}
	scanConfigs, err := scanConfigService.ListComplianceScanConfigurations(ctx, query)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(scanConfigs.GetConfigurations()), 1)
	assert.GreaterOrEqual(t, scanConfigs.TotalCount, int32(1))

	configs := scanConfigs.GetConfigurations()
	scanconfigID := getscanConfigID(testName, configs)
	defer deleteScanConfig(ctx, scanconfigID, scanConfigService)

	serviceResult := v2.NewComplianceResultsServiceClient(conn)
	query = &v2.RawQuery{Query: ""}
	err = retry.WithRetry(func() error {
		results, err := serviceResult.GetComplianceScanResults(ctx, query)
		if err != nil {
			return err
		}

		resultsList := results.GetScanResults()
		for i := 0; i < len(resultsList); i++ {
			if resultsList[i].GetScanName() == testName {
				return nil
			}
		}
		return errors.New("scan result not found")
	}, retry.BetweenAttempts(func(previousAttemptNumber int) {
		time.Sleep(60 * time.Second)
	}), retry.Tries(10))
	assert.NoError(t, err)

	// Create a different scan configuration with the same profile
	duplicateTestName := fmt.Sprintf("test-%s", uuid.NewV4().String())
	duplicateProfileReq := &v2.ComplianceScanConfiguration{
		ScanName: duplicateTestName,
		Id:       "",
		Clusters: []string{clusterID},
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan: false,
			Profiles:    []string{"rhcos4-e8"},
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
	assert.Equal(t, len(scanConfigs.GetConfigurations()), 1)

	// Create a scan configuration with invalid profiles configuration
	// contains both rhcos4-high and ocp4-e8 profiles. This is going
	// to fail validation, so we don't need to worry about running a larger
	// profile (e.g., rhcos4-high), since it won't increase test times.
	invalidProfileTestName := fmt.Sprintf("test-%s", uuid.NewV4().String())
	invalidProfileReq := &v2.ComplianceScanConfiguration{
		ScanName: invalidProfileTestName,
		Id:       "",
		Clusters: []string{clusterID},
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan: false,
			Profiles:    []string{"rhcos4-high", "ocp4-cis-node"},
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

	// Verify that the invalid scan configuration was not created and the error message is correct
	_, err = scanConfigService.CreateComplianceScanConfiguration(ctx, invalidProfileReq)
	if err == nil {
		t.Fatal("expected error creating scan configuration with invalid profiles")
	}
	assert.Contains(t, err.Error(), "profiles must have the same product")

	query = &v2.RawQuery{Query: ""}
	scanConfigs, err = scanConfigService.ListComplianceScanConfigurations(ctx, query)
	assert.NoError(t, err)
	assert.Equal(t, len(scanConfigs.GetConfigurations()), 1)
}

func TestComplianceV2UpdateScanConfigurations(t *testing.T) {
	ctx := context.Background()
	conn := centralgrpc.GRPCConnectionToCentral(t)
	scanConfigService := v2.NewComplianceScanConfigurationServiceClient(conn)
	serviceCluster := v1.NewClustersServiceClient(conn)
	clusters, err := serviceCluster.GetClusters(ctx, &v1.GetClustersRequest{})
	assert.NoError(t, err)
	require.Greater(t, len(clusters.GetClusters()), 0)
	clusterID := clusters.GetClusters()[0].GetId()

	// Create a scan configuration
	scanName := fmt.Sprintf("test-%s", uuid.NewV4().String())
	req := &v2.ComplianceScanConfiguration{
		ScanName: scanName,
		Id:       "",
		Clusters: []string{clusterID},
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan: false,
			Profiles:    []string{"ocp4-cis"},
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
		cleanUpResources(ctx, t, req.GetScanName(), coNamespaceV2)
	})

	query := &v2.RawQuery{Query: ""}
	scanConfigs, err := scanConfigService.ListComplianceScanConfigurations(ctx, query)
	assert.NoError(t, err)
	require.GreaterOrEqual(t, len(scanConfigs.GetConfigurations()), 1)
	assert.GreaterOrEqual(t, scanConfigs.TotalCount, int32(1))

	// Assert the ScanSetting and the ScanSettingBinding are created
	scanSetting := &complianceoperatorv1.ScanSetting{}
	scanSettingBinding := &complianceoperatorv1.ScanSettingBinding{}
	assertResourceDoesExist(ctx, t, scanName, coNamespaceV2, scanSetting)
	assertResourceDoesExist(ctx, t, scanName, coNamespaceV2, scanSettingBinding)
	assertScanSetting(t, *req, scanSetting)
	assertScanSettingBinding(t, *req, scanSettingBinding)

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
	updateReq.ScanConfig.Profiles = []string{"ocp4-high", "ocp4-high-node"}
	_, err = scanConfigService.UpdateComplianceScanConfiguration(ctx, updateReq)
	assert.NoError(t, err)

	query = &v2.RawQuery{Query: ""}
	scanConfigs, err = scanConfigService.ListComplianceScanConfigurations(ctx, query)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(scanConfigs.GetConfigurations()), 1)
	assert.GreaterOrEqual(t, scanConfigs.TotalCount, int32(1))

	// Assert the ScanSetting and the ScanSettingBinding are updated
	assertResourceWasUpdated(ctx, t, scanName, coNamespaceV2, scanSetting)
	assertResourceWasUpdated(ctx, t, scanName, coNamespaceV2, scanSettingBinding)
	assertScanSetting(t, *updateReq, scanSetting)
	assertScanSettingBinding(t, *updateReq, scanSettingBinding)
}

func TestComplianceV2DeleteComplianceScanConfigurations(t *testing.T) {
	ctx := context.Background()
	conn := centralgrpc.GRPCConnectionToCentral(t)
	scanConfigService := v2.NewComplianceScanConfigurationServiceClient(conn)
	// Retrieve the results from the scan configuration once the scan is complete
	serviceCluster := v1.NewClustersServiceClient(conn)
	clusters, err := serviceCluster.GetClusters(ctx, &v1.GetClustersRequest{})
	assert.NoError(t, err)

	clusterID := clusters.GetClusters()[0].GetId()
	testName := fmt.Sprintf("test-%s", uuid.NewV4().String())
	req := &v2.ComplianceScanConfiguration{
		ScanName: testName,
		Id:       "",
		Clusters: []string{clusterID},
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan: false,
			Profiles:    []string{"rhcos4-e8"},
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
	ctx := context.Background()
	conn := centralgrpc.GRPCConnectionToCentral(t)
	scanConfigService := v2.NewComplianceScanConfigurationServiceClient(conn)
	serviceCluster := v1.NewClustersServiceClient(conn)
	clusters, err := serviceCluster.GetClusters(ctx, &v1.GetClustersRequest{})
	assert.NoError(t, err)
	clusterID := clusters.GetClusters()[0].GetId()
	testName := fmt.Sprintf("test-%s", uuid.NewV4().String())
	req := &v2.ComplianceScanConfiguration{
		ScanName: testName,
		Id:       "",
		Clusters: []string{clusterID},
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan: false,
			Profiles:    []string{"rhcos4-e8"},
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

	query := &v2.RawQuery{Query: ""}
	scanConfigs, err := scanConfigService.ListComplianceScanConfigurations(ctx, query)
	configs := scanConfigs.GetConfigurations()
	scanconfigID := getscanConfigID(testName, configs)
	defer deleteScanConfig(ctx, scanconfigID, scanConfigService)

	// Ensure the ScanSetting and ScanSettingBinding have ACS metadata
	client := createDynamicClient(t)
	var scanSetting complianceoperatorv1.ScanSetting
	err = client.Get(context.TODO(), types.NamespacedName{Name: testName, Namespace: "openshift-compliance"}, &scanSetting)
	require.NoError(t, err, "failed to get ScanSetting %s", testName)

	assert.Contains(t, scanSetting.Labels, "app.kubernetes.io/name")
	assert.Equal(t, scanSetting.Labels["app.kubernetes.io/name"], "stackrox")
	assert.Contains(t, scanSetting.Annotations, "owner")
	assert.Equal(t, scanSetting.Annotations["owner"], "stackrox")

	var scanSettingBinding complianceoperatorv1.ScanSetting
	err = client.Get(context.TODO(), types.NamespacedName{Name: testName, Namespace: "openshift-compliance"}, &scanSettingBinding)
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
	conn := centralgrpc.GRPCConnectionToCentral(t)
	client := v2.NewComplianceScanConfigurationServiceClient(conn)
	integrationClient := v2.NewComplianceIntegrationServiceClient(conn)
	resp, err := integrationClient.ListComplianceIntegrations(context.TODO(), &v2.RawQuery{Query: ""})
	if err != nil {
		t.Fatal(err)
	}
	clusterId := resp.Integrations[0].ClusterId

	scanConfigName := "e8-scan-schedule"
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
			Description: "Scan schedule for the Austrailian Essential Eight profile to run daily.",
		},
		Clusters: []string{clusterId},
	}
	scanConfig, err := client.CreateComplianceScanConfiguration(context.TODO(), &sc)
	if err != nil {
		t.Fatal(err)
	}

	defer client.DeleteComplianceScanConfiguration(context.TODO(), &v2.ResourceByID{Id: scanConfig.GetId()})

	waitForComplianceSuiteToComplete(t, scanConfig.ScanName, 2*time.Second, 5*time.Minute)

	// Invoke a rescan
	_, err = client.RunComplianceScanConfiguration(context.TODO(), &v2.ResourceByID{Id: scanConfig.GetId()})
	require.NoError(t, err, "failed to rerun scan schedule %s", scanConfigName)

	// Assert the scan is rerunning on the cluster using the Compliance Operator CRDs
	waitForComplianceSuiteToComplete(t, scanConfig.ScanName, 2*time.Second, 5*time.Minute)
}
