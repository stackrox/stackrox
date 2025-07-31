package compliance

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"testing"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	v1 "github.com/stackrox/rox/generated/api/v1"
	v2 "github.com/stackrox/rox/generated/api/v2"
	"github.com/stackrox/rox/pkg/complianceoperator"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stackrox/rox/sensor/debugger/k8s"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	scanLabel = "compliance.openshift.io/scan-name"
)

var (
	profileNames = []string{"ocp4-cis", "ocp4-cis-1-5"}
	kubeConfigs  = map[string]string{
		"remote-1": os.Getenv("ROX_KUBECONFIG_1"),
		"remote-2": os.Getenv("ROX_KUBECONFIG_2"),
	}
)

func assertProfile(t *testing.T, profileName string, verbose bool) {
	t.Logf("Check results for profile %s", profileName)
	ctx := context.TODO()
	conn := centralgrpc.GRPCConnectionToCentral(t)
	clusterClient := v1.NewClustersServiceClient(conn)
	clusterRes, err := clusterClient.GetClusters(ctx, &v1.GetClustersRequest{})
	require.NoError(t, err)
	require.Len(t, clusterRes.GetClusters(), 2)
	scanClient := v2.NewComplianceResultsStatsServiceClient(conn)
	resultClient := v2.NewComplianceResultsServiceClient(conn)
	profileResults, err := scanClient.GetComplianceClusterStats(ctx, &v2.ComplianceProfileResultsRequest{
		ProfileName: profileName,
	})
	require.NoError(t, err)
	type check struct {
		Name   string
		Status string
	}
	acsResults := make(map[string][]check)
	for _, stats := range profileResults.GetScanStats() {
		res, err := resultClient.GetComplianceProfileClusterResults(ctx, &v2.ComplianceProfileClusterRequest{
			ProfileName: profileName,
			ClusterId:   stats.GetCluster().GetClusterId(),
		})
		require.NoError(t, err)
		t.Logf("acs results for cluster %s (total count %d)", stats.GetCluster().GetClusterName(), res.GetTotalCount())
		results := res.GetCheckResults()
		sort.Slice(results, func(i, j int) bool {
			return results[i].GetCheckName() < results[j].GetCheckName()
		})
		for _, r := range res.GetCheckResults() {
			// t.Logf("%s | %s", r.GetCheckName(), r.GetStatus().String())
			acsResults[stats.GetCluster().GetClusterName()] = append(
				acsResults[stats.GetCluster().GetClusterName()], check{
					Name:   r.GetCheckName(),
					Status: r.GetStatus().String(),
				})
		}
	}
	dirName := fmt.Sprintf("data/%s", profileName)
	items, err := os.ReadDir(dirName)
	require.NoError(t, err)
	reportResults := make(map[string][]check)
	for _, file := range items {
		if !strings.HasSuffix(file.Name(), ".csv") {
			continue
		}
		clusterName := strings.TrimPrefix(file.Name(), "cluster_")
		clusterName = clusterName[:len(clusterName)-(len(clusterName)-8)]
		csvFile, err := os.Open(path.Join(dirName, file.Name()))
		require.NoError(t, err)
		csvReader := csv.NewReader(csvFile)
		_, err = csvReader.Read()
		require.NoError(t, err)
		for row, err := csvReader.Read(); err != io.EOF; row, err = csvReader.Read() {
			// t.Logf("row %s | %s", row[1], row[5])
			reportResults[clusterName] = append(reportResults[clusterName], check{
				Name:   row[1],
				Status: row[5],
			})
		}
		sort.Slice(reportResults[clusterName], func(i, j int) bool {
			return reportResults[clusterName][i].Name < reportResults[clusterName][j].Name
		})
		t.Logf("report results for cluster %s file %s (total count: %d)", clusterName, file.Name(), len(reportResults[clusterName]))
	}
	k8sResults := make(map[string][]check)
	for cluster, kconfig := range kubeConfigs {
		t.Setenv("KUBECONFIG", kconfig)
		k8sClient, err := k8s.MakeOutOfClusterClient()
		require.NoError(t, err)
		checkClient := k8sClient.Dynamic().Resource(complianceoperator.ComplianceCheckResult.GroupVersionResource())
		listRes, err := checkClient.List(ctx, v12.ListOptions{})
		require.NoError(t, err)
		var checks []*v1alpha1.ComplianceCheckResult
		for _, it := range listRes.Items {
			c := &v1alpha1.ComplianceCheckResult{}
			require.NoError(t, runtime.DefaultUnstructuredConverter.FromUnstructured(it.Object, c))
			if profile, ok := c.GetLabels()[scanLabel]; ok && profile == profileName {
				checks = append(checks, c)
			}
		}
		sort.Slice(checks, func(i, j int) bool {
			return checks[i].GetName() < checks[j].GetName()
		})
		t.Logf("k8s results for cluster %s (total count: %d)", cluster, len(checks))
		for _, c := range checks {
			if profile, ok := c.GetLabels()[scanLabel]; ok && profile == profileName {
				// t.Logf("%s | %s | %s", c.GetName(), c.Status, profile)
				k8sResults[cluster] = append(k8sResults[cluster], check{
					Name:   c.GetName(),
					Status: string(c.Status),
				})
			}
		}
	}
	for cluster, checks := range k8sResults {
		acsChecks, ok := acsResults[cluster]
		require.True(t, ok)
		require.Equal(t, len(checks), len(acsChecks))
		reportChecks, ok := reportResults[cluster]
		require.True(t, ok)
		require.Equal(t, len(checks), len(reportChecks))
		for i := 0; i < len(checks); i++ {
			if verbose {
				t.Logf("%s - %s - %s", checks[i].Name, acsChecks[i].Name, reportChecks[i].Name)
				t.Logf("%s - %s - %s", checks[i].Status, acsChecks[i].Status, reportChecks[i].Status)
			}
			assert.Equal(t, checks[i].Name, acsChecks[i].Name)
			assert.Equal(t, checks[i].Status, acsChecks[i].Status)
			assert.Equal(t, checks[i].Name, reportChecks[i].Name)
			assert.Equal(t, checks[i].Status, reportChecks[i].Status)
		}
	}
}

func TestCoverage(t *testing.T) {
	for _, profileName := range profileNames {
		t.Run(fmt.Sprintf("Test profile %s", profileName), func(tt *testing.T) {
			assertProfile(tt, profileName, false)
		})
	}
}
