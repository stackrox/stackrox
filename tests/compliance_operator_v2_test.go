//go:build compliance

package tests

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
)

// ACS API test suite for integration testing for the Compliance Operator.
func TestComplianceV2Integration(t *testing.T) {
	conn := centralgrpc.GRPCConnectionToCentral(t)
	client := v2.NewComplianceIntegrationServiceClient(conn)

	q := &v2.RawQuery{Query: ""}
	resp, err := client.ListComplianceIntegrations(context.TODO(), q)
	if err != nil {
		t.Fatal(err)
	}
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
	profile, err := client.GetComplianceProfile(context.TODO(), &v2.ResourceByID{Id: "ocp4-cis"})
	if err != nil {
		t.Fatal(err)
	}
	assert.Greater(t, len(profile.Rules), 0, "failed to verify ocp4-cis profile contains any rules")
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
