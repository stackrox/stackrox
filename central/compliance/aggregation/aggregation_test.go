package aggregation

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stackrox/rox/central/compliance/standards"
	"github.com/stackrox/rox/central/compliance/standards/metadata"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func qualifiedNamespace(clusterID, namespace string) string {
	return clusterID + namespace
}

func qualifiedNamespaceID(clusterID, namespace string) string {
	return qualifiedNamespace(clusterID, namespace) + "ID"
}

type mockStandardsRepo struct {
	standards.Repository
}

func (mockStandardsRepo) GetCategoryByControl(controlID string) *standards.Category {
	return &standards.Category{
		Category: metadata.Category{
			ID: "",
		},
		Standard: &standards.Standard{
			Standard: metadata.Standard{
				ID: "standard1",
			},
		},
	}
}

func mockRunResult(cluster, standard string) *storage.ComplianceRunResults {
	return &storage.ComplianceRunResults{
		Domain: &storage.ComplianceDomain{
			Cluster: &storage.Cluster{
				Id: cluster,
			},
			Deployments: map[string]*storage.Deployment{
				cluster + "deployment1": {
					Id:          cluster + "deployment1",
					Namespace:   qualifiedNamespace(cluster, "namespace1"),
					NamespaceId: qualifiedNamespaceID(cluster, "namespace1"),
					ClusterId:   cluster,
				},
				cluster + "deployment2": {
					Id:          cluster + "deployment2",
					Namespace:   qualifiedNamespace(cluster, "namespace2"),
					NamespaceId: qualifiedNamespaceID(cluster, "namespace2"),
					ClusterId:   cluster,
				},
				cluster + "deployment3": {
					Id:          cluster + "deployment3",
					Namespace:   qualifiedNamespace(cluster, "namespace3"),
					NamespaceId: qualifiedNamespaceID(cluster, "namespace3"),
					ClusterId:   cluster,
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
				"control7": {
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
					"control7": {
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
					"control7": {
						OverallState: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
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
					"control7": {
						OverallState: storage.ComplianceState_COMPLIANCE_STATE_SKIP,
					},
				},
			},
		},
	}
}

func mockFlatChecks(clusterID, standardID string) []flatCheck {
	category := fmt.Sprintf("%s:%s", standardID, "")
	return []flatCheck{
		newFlatCheck(clusterID, "", standardID, category, "control1", "", "", storage.ComplianceState_COMPLIANCE_STATE_FAILURE),
		newFlatCheck(clusterID, "", standardID, category, "control2", "", "", storage.ComplianceState_COMPLIANCE_STATE_SUCCESS),
		newFlatCheck(clusterID, "", standardID, category, "control7", "", "", storage.ComplianceState_COMPLIANCE_STATE_SUCCESS),
		newFlatCheck(clusterID, "", standardID, category, "control3", clusterID+"node1", "", storage.ComplianceState_COMPLIANCE_STATE_ERROR),
		newFlatCheck(clusterID, "", standardID, category, "control4", clusterID+"node1", "", storage.ComplianceState_COMPLIANCE_STATE_SUCCESS),
		newFlatCheck(clusterID, "", standardID, category, "control3", clusterID+"node2", "", storage.ComplianceState_COMPLIANCE_STATE_FAILURE),
		newFlatCheck(clusterID, "", standardID, category, "control4", clusterID+"node2", "", storage.ComplianceState_COMPLIANCE_STATE_SUCCESS),
		newFlatCheck(clusterID, qualifiedNamespaceID(clusterID, "namespace1"), standardID, category, "control5", "", clusterID+"deployment1", storage.ComplianceState_COMPLIANCE_STATE_FAILURE),
		newFlatCheck(clusterID, qualifiedNamespaceID(clusterID, "namespace1"), standardID, category, "control6", "", clusterID+"deployment1", storage.ComplianceState_COMPLIANCE_STATE_SUCCESS),
		newFlatCheck(clusterID, qualifiedNamespaceID(clusterID, "namespace1"), standardID, category, "control7", "", clusterID+"deployment1", storage.ComplianceState_COMPLIANCE_STATE_SUCCESS),
		newFlatCheck(clusterID, qualifiedNamespaceID(clusterID, "namespace2"), standardID, category, "control5", "", clusterID+"deployment2", storage.ComplianceState_COMPLIANCE_STATE_FAILURE),
		newFlatCheck(clusterID, qualifiedNamespaceID(clusterID, "namespace2"), standardID, category, "control6", "", clusterID+"deployment2", storage.ComplianceState_COMPLIANCE_STATE_SUCCESS),
		newFlatCheck(clusterID, qualifiedNamespaceID(clusterID, "namespace2"), standardID, category, "control7", "", clusterID+"deployment2", storage.ComplianceState_COMPLIANCE_STATE_FAILURE),
		newFlatCheck(clusterID, qualifiedNamespaceID(clusterID, "namespace3"), standardID, category, "control5", "", clusterID+"deployment3", storage.ComplianceState_COMPLIANCE_STATE_SKIP),
		newFlatCheck(clusterID, qualifiedNamespaceID(clusterID, "namespace3"), standardID, category, "control6", "", clusterID+"deployment3", storage.ComplianceState_COMPLIANCE_STATE_SKIP),
		newFlatCheck(clusterID, qualifiedNamespaceID(clusterID, "namespace3"), standardID, category, "control7", "", clusterID+"deployment3", storage.ComplianceState_COMPLIANCE_STATE_SKIP),
	}
}

func TestGetFlatChecksFromRunResult(t *testing.T) {
	ag := &aggregatorImpl{
		standards: mockStandardsRepo{},
	}
	assert.ElementsMatch(t, mockFlatChecks("cluster1", "standard1"), ag.getFlatChecksFromRunResult(mockRunResult("cluster1", "standard1"), [numScopes]set.StringSet{}))
}

func testName(groupBy []v1.ComplianceAggregation_Scope, unit v1.ComplianceAggregation_Scope) string {
	groupBys := make([]string, 0, len(groupBy))
	for _, g := range groupBy {
		groupBys = append(groupBys, g.String())
	}
	return fmt.Sprintf("GroupBy %s - Unit %s", strings.Join(groupBys, "-"), unit.String())
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
			failPerResult: 4,
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
			failPerResult: 4,
			passPerResult: 3,
			numResults:    2,
		},
		{
			groupBy:       []v1.ComplianceAggregation_Scope{v1.ComplianceAggregation_CLUSTER, v1.ComplianceAggregation_STANDARD},
			unit:          v1.ComplianceAggregation_CONTROL,
			failPerResult: 4,
			passPerResult: 3,
			numResults:    4,
		},
		{
			groupBy:       []v1.ComplianceAggregation_Scope{v1.ComplianceAggregation_CLUSTER},
			unit:          v1.ComplianceAggregation_CHECK,
			failPerResult: 12,
			passPerResult: 14,
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
			ag := &aggregatorImpl{
				standards: mockStandardsRepo{},
			}
			results, _ := ag.getAggregatedResults(c.groupBy, c.unit, runResults, [numScopes]set.StringSet{})
			require.Equal(t, c.numResults, len(results))
			for _, r := range results {
				assert.Equal(t, c.passPerResult, r.NumPassing)
				assert.Equal(t, c.failPerResult, r.NumFailing)
			}
		})
	}
}

func TestDomainAttribution(t *testing.T) {
	ag := &aggregatorImpl{
		standards: mockStandardsRepo{},
	}
	complianceRunResults := []*storage.ComplianceRunResults{
		{
			NodeResults: map[string]*storage.ComplianceRunResults_EntityResults{
				"cluster1-node1": {
					ControlResults: map[string]*storage.ComplianceResultValue{
						"check1": {},
						"check2": {},
					},
				},
				"cluster1-node2": {
					ControlResults: map[string]*storage.ComplianceResultValue{
						"check1": {},
						"check2": {},
					},
				},
			},
			Domain: &storage.ComplianceDomain{
				Nodes: map[string]*storage.Node{
					"cluster1-node1": {
						Id:   "cluster1-node1",
						Name: "cluster1-node1",
					},
					"cluster1-node2": {
						Id:   "cluster1-node2",
						Name: "cluster1-node2",
					},
				},
			},
		},
		{
			NodeResults: map[string]*storage.ComplianceRunResults_EntityResults{
				"cluster2-node1": {
					ControlResults: map[string]*storage.ComplianceResultValue{
						"check1": {},
						"check2": {},
					},
				},
				"cluster2-node2": {
					ControlResults: map[string]*storage.ComplianceResultValue{
						"check1": {},
						"check2": {},
					},
				},
			},
			Domain: &storage.ComplianceDomain{
				Nodes: map[string]*storage.Node{
					"cluster2-node1": {
						Id:   "cluster2-node1",
						Name: "cluster2-node1",
					},
					"cluster2-node2": {
						Id:   "cluster2-node2",
						Name: "cluster2-node2",
					},
				},
			},
		},
	}

	results, domainMap := ag.getAggregatedResults(
		[]v1.ComplianceAggregation_Scope{v1.ComplianceAggregation_CONTROL, v1.ComplianceAggregation_NODE},
		v1.ComplianceAggregation_CHECK,
		complianceRunResults,
		[numScopes]set.StringSet{},
	)

	for i, r := range results {
		nodeID := r.AggregationKeys[1].GetId()
		mappedDomain := domainMap[results[i]].GetNodes()
		_, ok := mappedDomain[nodeID]
		assert.True(t, ok)
	}
}
