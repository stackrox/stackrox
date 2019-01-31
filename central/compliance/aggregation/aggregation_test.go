package aggregation

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockRunResult(cluster, standard string) *storage.ComplianceRunResults {
	return &storage.ComplianceRunResults{
		Domain: &storage.ComplianceDomain{
			Cluster: &storage.Cluster{
				Id: cluster,
			},
			Deployments: map[string]*storage.Deployment{
				cluster + "deployment1": {
					Id:        cluster + "deployment1",
					Namespace: cluster + "namespace1",
				},
				cluster + "deployment2": {
					Id:        cluster + "deployment2",
					Namespace: cluster + "namespace2",
				},
				cluster + "deployment3": {
					Id:        cluster + "deployment3",
					Namespace: cluster + "namespace3",
				},
			},
			Nodes: map[string]*storage.Node{
				cluster + "node1": {
					Id: cluster + "node1",
				},
				cluster + "node1": {
					Id: cluster + "node2",
				},
			},
		},
		RunMetadata: &storage.ComplianceRunMetadata{
			StandardId: standard,
		},
		ClusterResults: &storage.ComplianceRunResults_EntityResults{
			ControlResults: map[string]*storage.ComplianceResultValue{
				"control1": {
					OverallState: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
				},
				"control2": {
					OverallState: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
				},
			},
		},
		NodeResults: map[string]*storage.ComplianceRunResults_EntityResults{
			cluster + "node1": {
				ControlResults: map[string]*storage.ComplianceResultValue{
					"control3": {
						OverallState: storage.ComplianceState_COMPLIANCE_STATE_ERROR,
					},
					"control4": {
						OverallState: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
					},
				},
			},
			cluster + "node2": {
				ControlResults: map[string]*storage.ComplianceResultValue{
					"control3": {
						OverallState: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
					},
					"control4": {
						OverallState: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
					},
				},
			},
		},
		DeploymentResults: map[string]*storage.ComplianceRunResults_EntityResults{
			cluster + "deployment1": {
				ControlResults: map[string]*storage.ComplianceResultValue{
					"control5": {
						OverallState: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
					},
					"control6": {
						OverallState: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
					},
				},
			},
			cluster + "deployment2": {
				ControlResults: map[string]*storage.ComplianceResultValue{
					"control5": {
						OverallState: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
					},
					"control6": {
						OverallState: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
					},
				},
			},
			cluster + "deployment3": {
				ControlResults: map[string]*storage.ComplianceResultValue{
					"control5": {
						OverallState: storage.ComplianceState_COMPLIANCE_STATE_SKIP,
					},
					"control6": {
						OverallState: storage.ComplianceState_COMPLIANCE_STATE_SKIP,
					},
				},
			},
		},
	}
}

func mockFlatChecks(clusterID, standardID string) []flatCheck {
	return []flatCheck{
		newFlatCheck(clusterID, "", standardID, "", "control1", "", "", storage.ComplianceState_COMPLIANCE_STATE_FAILURE),
		newFlatCheck(clusterID, "", standardID, "", "control2", "", "", storage.ComplianceState_COMPLIANCE_STATE_SUCCESS),
		newFlatCheck(clusterID, "", standardID, "", "control3", clusterID+"node1", "", storage.ComplianceState_COMPLIANCE_STATE_ERROR),
		newFlatCheck(clusterID, "", standardID, "", "control4", clusterID+"node1", "", storage.ComplianceState_COMPLIANCE_STATE_SUCCESS),
		newFlatCheck(clusterID, "", standardID, "", "control3", clusterID+"node2", "", storage.ComplianceState_COMPLIANCE_STATE_FAILURE),
		newFlatCheck(clusterID, "", standardID, "", "control4", clusterID+"node2", "", storage.ComplianceState_COMPLIANCE_STATE_SUCCESS),
		newFlatCheck(clusterID, clusterID+"namespace1", standardID, "", "control5", "", clusterID+"deployment1", storage.ComplianceState_COMPLIANCE_STATE_FAILURE),
		newFlatCheck(clusterID, clusterID+"namespace1", standardID, "", "control6", "", clusterID+"deployment1", storage.ComplianceState_COMPLIANCE_STATE_SUCCESS),
		newFlatCheck(clusterID, clusterID+"namespace2", standardID, "", "control5", "", clusterID+"deployment2", storage.ComplianceState_COMPLIANCE_STATE_FAILURE),
		newFlatCheck(clusterID, clusterID+"namespace2", standardID, "", "control6", "", clusterID+"deployment2", storage.ComplianceState_COMPLIANCE_STATE_SUCCESS),
		newFlatCheck(clusterID, clusterID+"namespace3", standardID, "", "control5", "", clusterID+"deployment3", storage.ComplianceState_COMPLIANCE_STATE_SKIP),
		newFlatCheck(clusterID, clusterID+"namespace3", standardID, "", "control6", "", clusterID+"deployment3", storage.ComplianceState_COMPLIANCE_STATE_SKIP),
	}
}

func TestGetFlatChecksFromRunResult(t *testing.T) {
	assert.ElementsMatch(t, mockFlatChecks("cluster1", "standard1"), getFlatChecksFromRunResult(mockRunResult("cluster1", "standard1")))
}

func testName(groupBy []v1.ComplianceAggregation_Scope, unit v1.ComplianceAggregation_Scope) string {
	groupBys := make([]string, 0, len(groupBy))
	for _, g := range groupBy {
		groupBys = append(groupBys, g.String())
	}
	return fmt.Sprintf("GroupBy: %s - Unit: %s", strings.Join(groupBys, "-"), unit.String())
}

func TestGetAggregatedResults(t *testing.T) {
	var cases = []struct {
		groupBy       []v1.ComplianceAggregation_Scope
		unit          v1.ComplianceAggregation_Scope
		passPerResult int32
		failPerResult int32
		numResults    int
	}{
		{
			unit:          v1.ComplianceAggregation_CLUSTER,
			failPerResult: 2,
			numResults:    1,
		},
		{
			unit:          v1.ComplianceAggregation_NAMESPACE,
			failPerResult: 4,
			numResults:    1,
		},
		{
			unit:          v1.ComplianceAggregation_NODE,
			failPerResult: 4,
			numResults:    1,
		},
		{
			unit:          v1.ComplianceAggregation_DEPLOYMENT,
			failPerResult: 4,
			numResults:    1,
		},
		{
			unit:          v1.ComplianceAggregation_STANDARD,
			failPerResult: 2,
			numResults:    1,
		},
		{
			unit:          v1.ComplianceAggregation_CONTROL,
			failPerResult: 3,
			passPerResult: 3,
			numResults:    1,
		},
		{
			groupBy:       []v1.ComplianceAggregation_Scope{v1.ComplianceAggregation_CLUSTER},
			unit:          v1.ComplianceAggregation_CLUSTER,
			failPerResult: 1,
			numResults:    2,
		},
		{
			groupBy:       []v1.ComplianceAggregation_Scope{v1.ComplianceAggregation_CLUSTER},
			unit:          v1.ComplianceAggregation_NAMESPACE,
			failPerResult: 2,
			numResults:    2,
		},
		{
			groupBy:       []v1.ComplianceAggregation_Scope{v1.ComplianceAggregation_CLUSTER},
			unit:          v1.ComplianceAggregation_NODE,
			failPerResult: 2,
			numResults:    2,
		},
		{
			groupBy:       []v1.ComplianceAggregation_Scope{v1.ComplianceAggregation_CLUSTER},
			unit:          v1.ComplianceAggregation_DEPLOYMENT,
			failPerResult: 2,
			numResults:    2,
		},
		{
			groupBy:       []v1.ComplianceAggregation_Scope{v1.ComplianceAggregation_CLUSTER},
			unit:          v1.ComplianceAggregation_STANDARD,
			failPerResult: 2,
			numResults:    2,
		},
		{
			groupBy:       []v1.ComplianceAggregation_Scope{v1.ComplianceAggregation_CLUSTER},
			unit:          v1.ComplianceAggregation_CONTROL,
			failPerResult: 3,
			passPerResult: 3,
			numResults:    2,
		},
		{
			groupBy:       []v1.ComplianceAggregation_Scope{v1.ComplianceAggregation_CLUSTER, v1.ComplianceAggregation_STANDARD},
			unit:          v1.ComplianceAggregation_CONTROL,
			failPerResult: 3,
			passPerResult: 3,
			numResults:    4,
		},
		{
			groupBy:       []v1.ComplianceAggregation_Scope{v1.ComplianceAggregation_CLUSTER},
			unit:          v1.ComplianceAggregation_CHECK,
			failPerResult: 10,
			passPerResult: 10,
			numResults:    2,
		},
	}
	runResults := []*storage.ComplianceRunResults{
		mockRunResult("cluster1", "standard1"),
		mockRunResult("cluster1", "standard2"),
		mockRunResult("cluster2", "standard1"),
		mockRunResult("cluster2", "standard2"),
	}

	for _, c := range cases {
		t.Run(testName(c.groupBy, c.unit), func(t *testing.T) {
			results, _ := GetAggregatedResults(c.groupBy, c.unit, runResults)
			require.Equal(t, c.numResults, len(results))
			for _, r := range results {
				assert.Equal(t, c.passPerResult, r.NumPassing)
				assert.Equal(t, c.failPerResult, r.NumFailing)
			}
		})
	}
}
