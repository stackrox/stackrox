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
	v1 "github.com/stackrox/rox/generated/api/v1"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	extscheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	cached "k8s.io/client-go/discovery/cached"
	cgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/restmapper"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func createDynamicClient(t *testing.T) dynclient.Client {
	restCfg := getConfig(t)
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
			Scheme:         k8sScheme,
			Mapper:         restMapper,
			WarningHandler: dynclient.WarningHandlerOptions{SuppressWarnings: true},
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

// ACS API test suite for integration testing for the Compliance Operator.
func TestComplianceV2Integration(t *testing.T) {
	resp := getIntegrations(t)
	assert.Len(t, resp.Integrations, 1, "failed to assert there is only a single compliance integration")
	assert.Equal(t, resp.Integrations[0].ClusterName, "remote", "failed to find integration for cluster called \"remote\"")
	assert.Equal(t, resp.Integrations[0].Namespace, "openshift-compliance", "failed to find integration for \"openshift-compliance\" namespace")
}

func TestComplianceV2ProfileCount(t *testing.T) {
	conn := centralgrpc.GRPCConnectionToCentral(t)
	client := v2.NewComplianceProfileServiceClient(conn)

	profileCount, err := client.GetComplianceProfileCount(context.TODO(), &v2.RawQuery{Query: ""})
	if err != nil {
		t.Fatal(err)
	}
	assert.Greater(t, profileCount.Count, int32(0), "unable to verify any compliance profiles were ingested")
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
	service := v2.NewComplianceScanConfigurationServiceClient(conn)
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
			Profiles:    []string{"rhcos4-moderate-rev-4"},
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

	resp, err := service.CreateComplianceScanConfiguration(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, req.GetScanName(), resp.GetScanName())

	query := &v2.RawQuery{Query: ""}
	scanConfigs, err := service.ListComplianceScanConfigurations(ctx, query)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(scanConfigs.GetConfigurations()), 1)

	configs := scanConfigs.GetConfigurations()
	scanconfigID := getscanConfigID(testName, configs)
	defer deleteScanConfig(ctx, scanconfigID, service)
	count, err := service.GetComplianceScanConfigurationsCount(ctx, query)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, count.GetCount(), int32(1))

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
}

func TestComplianceV2DeleteComplianceScanConfigurations(t *testing.T) {
	ctx := context.Background()
	conn := centralgrpc.GRPCConnectionToCentral(t)
	service := v2.NewComplianceScanConfigurationServiceClient(conn)
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
			Profiles:    []string{"rhcos4-moderate-rev-4"},
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

	resp, err := service.CreateComplianceScanConfiguration(ctx, req)
	assert.NoError(t, err)
	assert.Equal(t, req.GetScanName(), resp.GetScanName())

	query := &v2.RawQuery{Query: ""}
	scanConfigs, err := service.ListComplianceScanConfigurations(ctx, query)
	configs := scanConfigs.GetConfigurations()
	scanconfigID := getscanConfigID(testName, configs)
	reqDelete := &v2.ResourceByID{
		Id: scanconfigID,
	}
	_, err = service.DeleteComplianceScanConfiguration(ctx, reqDelete)
	assert.NoError(t, err)

	// Verify scan configuration no longer exists
	scanConfigs, err = service.ListComplianceScanConfigurations(ctx, query)
	configs = scanConfigs.GetConfigurations()
	scanconfigID = getscanConfigID(testName, configs)
	assert.Empty(t, scanconfigID)
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

	scanConfigName := "cis-scan-schedule"
	sc := v2.ComplianceScanConfiguration{
		ScanName: scanConfigName,
		ScanConfig: &v2.BaseComplianceScanConfigurationSettings{
			OneTimeScan: false,
			Profiles:    []string{"ocp4-cis", "ocp4-cis-node"},
			ScanSchedule: &v2.Schedule{
				IntervalType: 3,
				Hour:         0,
				Minute:       0,
			},
			Description: "Scan schedule for CIS profiles to run daily.",
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
