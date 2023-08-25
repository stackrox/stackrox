package aggregation

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stackrox/rox/central/compliance/standards"
	"github.com/stackrox/rox/central/compliance/standards/metadata"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func qualifiedNamespace(clusterID, namespace string) string {
	return clusterID + namespace
}

func qualifiedNamespaceID(clusterID, namespace string) string {
	return qualifiedNamespace(clusterID, namespace) + "ID"
}

// we register only standard1, standard2 will not be found in controlsByID map.
const registeredStandardID = "standard1"

func mockStandardsRepo(t require.TestingT) standards.Repository {
	controls := make([]metadata.Control, 0, 8)
	for i := 0; i < 9; i++ {
		controls = append(controls, metadata.Control{
			ID: fmt.Sprintf("control%d", i),
		})
	}

	repo, err := standards.NewRegistry(nil, metadata.Standard{
		ID:          registeredStandardID,
		Name:        "",
		Description: "",
		Dynamic:     false,
		Categories: []metadata.Category{
			{
				ID:          "",
				Name:        "",
				Description: "",
				Controls:    controls,
			},
		},
	})

	require.NoError(t, err)
	return repo
}

func mockRunResult(cluster, standard string) *storage.ComplianceRunResults {
	return &storage.ComplianceRunResults{
		Domain: &storage.ComplianceDomain{
			Cluster: &storage.ComplianceDomain_Cluster{
				Id: cluster,
			},
			Deployments: map[string]*storage.ComplianceDomain_Deployment{
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
			Nodes: map[string]*storage.ComplianceDomain_Node{
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
				standard + ":control1": {
					OverallState: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
				},
				standard + ":control2": {
					OverallState: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
				},
				standard + ":control7": {
					OverallState: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
				},
			},
		},
		NodeResults: map[string]*storage.ComplianceRunResults_EntityResults{
			cluster + "node1": {
				ControlResults: map[string]*storage.ComplianceResultValue{
					standard + ":control3": {
						OverallState: storage.ComplianceState_COMPLIANCE_STATE_ERROR,
					},
					standard + ":control4": {
						OverallState: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
					},
				},
			},
			cluster + "node2": {
				ControlResults: map[string]*storage.ComplianceResultValue{
					standard + ":control3": {
						OverallState: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
					},
					standard + ":control4": {
						OverallState: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
					},
				},
			},
		},
		DeploymentResults: map[string]*storage.ComplianceRunResults_EntityResults{
			cluster + "deployment1": {
				ControlResults: map[string]*storage.ComplianceResultValue{
					standard + ":control5": {
						OverallState: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
					},
					standard + ":control6": {
						OverallState: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
					},
					standard + ":control7": {
						OverallState: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
					},
				},
			},
			cluster + "deployment2": {
				ControlResults: map[string]*storage.ComplianceResultValue{
					standard + ":control5": {
						OverallState: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
					},
					standard + ":control6": {
						OverallState: storage.ComplianceState_COMPLIANCE_STATE_SUCCESS,
					},
					standard + ":control7": {
						OverallState: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
					},
					standard + ":control8": {
						OverallState: storage.ComplianceState_COMPLIANCE_STATE_NOTE,
					},
				},
			},
			cluster + "deployment3": {
				ControlResults: map[string]*storage.ComplianceResultValue{
					standard + ":control5": {
						OverallState: storage.ComplianceState_COMPLIANCE_STATE_SKIP,
					},
					standard + ":control6": {
						OverallState: storage.ComplianceState_COMPLIANCE_STATE_SKIP,
					},
					standard + ":control7": {
						OverallState: storage.ComplianceState_COMPLIANCE_STATE_SKIP,
					},
				},
			},
		},
	}
}

func testName(groupBy []storage.ComplianceAggregation_Scope, unit storage.ComplianceAggregation_Scope) string {
	groupBys := make([]string, 0, len(groupBy))
	for _, g := range groupBy {
		groupBys = append(groupBys, g.String())
	}
	return fmt.Sprintf("GroupBy %s - Unit %s", strings.Join(groupBys, "-"), unit.String())
}

func TestMaxScopeMatches(t *testing.T) {
	assert.Equal(t, len(storage.ComplianceAggregation_Scope_name)-1, int(maxScope))
}

func TestInvalidParameters(t *testing.T) {
	a := &aggregatorImpl{}
	_, _, _, err := a.Aggregate(context.TODO(), "", []storage.ComplianceAggregation_Scope{storage.ComplianceAggregation_NAMESPACE}, storage.ComplianceAggregation_UNKNOWN)
	assert.Error(t, err)

	_, _, _, err = a.Aggregate(context.TODO(), "", nil, storage.ComplianceAggregation_UNKNOWN)
	assert.Error(t, err)

	_, _, _, err = a.Aggregate(context.TODO(), "", []storage.ComplianceAggregation_Scope{storage.ComplianceAggregation_UNKNOWN}, storage.ComplianceAggregation_CHECK)
	assert.Error(t, err)
}

func TestGetAggregatedResults(t *testing.T) {
	var cases = []struct {
		groupBy       []storage.ComplianceAggregation_Scope
		unit          storage.ComplianceAggregation_Scope
		passPerResult int32
		failPerResult int32
		skipPerResult int32
		numResults    int
		mask          *mask
	}{
		{
			unit:          storage.ComplianceAggregation_CLUSTER,
			failPerResult: 2,
			numResults:    1,
		},
		{
			unit:          storage.ComplianceAggregation_NAMESPACE,
			failPerResult: 4,
			skipPerResult: 2,
			numResults:    1,
		},
		{
			unit:          storage.ComplianceAggregation_NODE,
			failPerResult: 4,
			numResults:    1,
		},
		{
			unit:          storage.ComplianceAggregation_DEPLOYMENT,
			failPerResult: 4,
			skipPerResult: 2,
			numResults:    1,
		},
		{
			unit:          storage.ComplianceAggregation_STANDARD,
			failPerResult: 2,
			numResults:    1,
		},
		{
			unit:          storage.ComplianceAggregation_CONTROL,
			failPerResult: 8,
			passPerResult: 6,
			skipPerResult: 2,
			numResults:    1,
		},
		{
			groupBy:       []storage.ComplianceAggregation_Scope{storage.ComplianceAggregation_CLUSTER},
			unit:          storage.ComplianceAggregation_CLUSTER,
			failPerResult: 1,
			numResults:    2,
		},
		{
			groupBy:       []storage.ComplianceAggregation_Scope{storage.ComplianceAggregation_CLUSTER},
			unit:          storage.ComplianceAggregation_NAMESPACE,
			failPerResult: 2,
			skipPerResult: 1,
			numResults:    2,
		},
		{
			groupBy:       []storage.ComplianceAggregation_Scope{storage.ComplianceAggregation_CLUSTER},
			unit:          storage.ComplianceAggregation_NODE,
			failPerResult: 2,
			numResults:    2,
		},
		{
			groupBy:       []storage.ComplianceAggregation_Scope{storage.ComplianceAggregation_CLUSTER},
			unit:          storage.ComplianceAggregation_DEPLOYMENT,
			failPerResult: 2,
			skipPerResult: 1,
			numResults:    2,
		},
		{
			groupBy:       []storage.ComplianceAggregation_Scope{storage.ComplianceAggregation_CLUSTER},
			unit:          storage.ComplianceAggregation_STANDARD,
			failPerResult: 2,
			numResults:    2,
		},
		{
			groupBy:       []storage.ComplianceAggregation_Scope{storage.ComplianceAggregation_CLUSTER},
			unit:          storage.ComplianceAggregation_CONTROL,
			failPerResult: 8,
			passPerResult: 6,
			skipPerResult: 2,
			numResults:    2,
		},
		{
			groupBy:       []storage.ComplianceAggregation_Scope{storage.ComplianceAggregation_CLUSTER, storage.ComplianceAggregation_STANDARD},
			unit:          storage.ComplianceAggregation_CONTROL,
			failPerResult: 4,
			passPerResult: 3,
			skipPerResult: 1,
			numResults:    4,
		},
		{
			groupBy:       []storage.ComplianceAggregation_Scope{storage.ComplianceAggregation_CLUSTER},
			unit:          storage.ComplianceAggregation_CHECK,
			failPerResult: 12,
			passPerResult: 14,
			skipPerResult: 8,
			numResults:    2,
		},
		{
			groupBy:       []storage.ComplianceAggregation_Scope{storage.ComplianceAggregation_CLUSTER},
			unit:          storage.ComplianceAggregation_CHECK,
			failPerResult: 2,
			passPerResult: 4,
			numResults:    1,
			mask: &mask{
				storage.ComplianceAggregation_NAMESPACE - minScope: set.NewStringSet(qualifiedNamespaceID("cluster1", "namespace1")),
			},
		},
		{
			groupBy:       []storage.ComplianceAggregation_Scope{storage.ComplianceAggregation_DEPLOYMENT},
			unit:          storage.ComplianceAggregation_CHECK,
			failPerResult: 2,
			passPerResult: 4,
			numResults:    1,
			mask: &mask{
				storage.ComplianceAggregation_DEPLOYMENT - minScope: set.NewStringSet("cluster1deployment1"),
			},
		},
	}
	runResults := []*storage.ComplianceRunResults{
		mockRunResult("cluster1", registeredStandardID),
		mockRunResult("cluster1", "standard2"),
		mockRunResult("cluster2", registeredStandardID),
		mockRunResult("cluster2", "standard2"),
	}

	for _, c := range cases {
		t.Run(testName(c.groupBy, c.unit), func(t *testing.T) {
			ag := &aggregatorImpl{
				standards: mockStandardsRepo(t),
			}
			results, _ := ag.getAggregatedResults(c.groupBy, c.unit, runResults, c.mask)
			require.Equal(t, c.numResults, len(results))
			for _, r := range results {
				assert.Equal(t, c.passPerResult, r.NumPassing)
				assert.Equal(t, c.failPerResult, r.NumFailing)
				assert.Equal(t, c.skipPerResult, r.NumSkipped)
			}
		})
	}
}

func TestDomainAttribution(t *testing.T) {
	ag := &aggregatorImpl{
		standards: mockStandardsRepo(t),
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
				Nodes: map[string]*storage.ComplianceDomain_Node{
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
				Nodes: map[string]*storage.ComplianceDomain_Node{
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
		[]storage.ComplianceAggregation_Scope{storage.ComplianceAggregation_CONTROL, storage.ComplianceAggregation_NODE},
		storage.ComplianceAggregation_CHECK,
		complianceRunResults,
		&mask{},
	)
	for i, r := range results {
		nodeID := r.AggregationKeys[1].GetId()
		mappedDomain := domainMap[results[i]].GetNodes()
		_, ok := mappedDomain[nodeID]
		assert.True(t, ok)
	}
}

/*

Test Cases illustrated by test below

Search Cluster A

      Cluster    Namespace    Deployment    Node
        A            ""           ""         "" - EXPECT
        A            B            C          "" - EXPECT
        A            ""           ""         D  - EXPECT

Mask   [A]        [B, B1]       [C, C1]    [D, D1]


Search Namespace B

      Cluster    Namespace    Deployment    Node
        A            ""           ""         "" - DONT EXPECT
        A            B            C          "" - EXPECT
        A            ""           ""         D  - DONT EXPECT

Mask  <nil>        [B, B1]       [C, C1]    <nil>

*/

func TestIsValidCheck(t *testing.T) {
	type check struct {
		fc     flatCheck
		result bool
	}

	var cases = []struct {
		mask   map[storage.ComplianceAggregation_Scope]set.StringSet
		checks []check
	}{
		{
			mask: map[storage.ComplianceAggregation_Scope]set.StringSet{
				storage.ComplianceAggregation_CLUSTER:    set.NewStringSet("A"),
				storage.ComplianceAggregation_NAMESPACE:  set.NewStringSet("B"),
				storage.ComplianceAggregation_DEPLOYMENT: set.NewStringSet("C"),
				storage.ComplianceAggregation_NODE:       set.NewStringSet("D"),
			},
			checks: []check{
				{
					fc: flatCheck{
						values: &flatCheckValues{
							storage.ComplianceAggregation_CLUSTER - minScope: "A",
						},
					},
					result: true,
				},
				{
					fc: flatCheck{
						values: &flatCheckValues{
							storage.ComplianceAggregation_CLUSTER - minScope:    "A",
							storage.ComplianceAggregation_NAMESPACE - minScope:  "B",
							storage.ComplianceAggregation_DEPLOYMENT - minScope: "C",
						},
					},
					result: true,
				},
				{
					fc: flatCheck{
						values: &flatCheckValues{
							storage.ComplianceAggregation_CLUSTER - minScope: "A",
							storage.ComplianceAggregation_NODE - minScope:    "D",
						},
					},
					result: true,
				},
			},
		},
		{
			mask: map[storage.ComplianceAggregation_Scope]set.StringSet{
				storage.ComplianceAggregation_NAMESPACE:  set.NewStringSet("B"),
				storage.ComplianceAggregation_DEPLOYMENT: set.NewStringSet("C"),
			},
			checks: []check{
				{
					fc: flatCheck{
						values: &flatCheckValues{
							storage.ComplianceAggregation_CLUSTER - minScope: "A",
						},
					},
					result: false,
				},
				{
					fc: flatCheck{
						values: &flatCheckValues{
							storage.ComplianceAggregation_CLUSTER - minScope:    "A",
							storage.ComplianceAggregation_NAMESPACE - minScope:  "B",
							storage.ComplianceAggregation_DEPLOYMENT - minScope: "C",
						},
					},
					result: true,
				},
				{
					fc: flatCheck{
						values: &flatCheckValues{
							storage.ComplianceAggregation_CLUSTER - minScope: "A",
							storage.ComplianceAggregation_NODE - minScope:    "D",
						},
					},
					result: false,
				},
			},
		},
	}
	for _, testCase := range cases {
		// testCase mask to actual mask
		testMask := &mask{}
		for k, v := range testCase.mask {
			testMask.set(k, v.Clone())
		}
		for _, c := range testCase.checks {
			t.Run("aggregation", func(t *testing.T) {
				assert.Equal(t, c.result, isValidCheck(testMask, c.fc))
			})
		}
	}
}

func mockBenchmarkRunResult() *storage.ComplianceRunResults {
	deploymentResults := make(map[string]*storage.ComplianceRunResults_EntityResults)
	deployments := make(map[string]*storage.ComplianceDomain_Deployment)
	for i := 0; i < 10000; i++ {
		results := &storage.ComplianceRunResults_EntityResults{
			ControlResults: make(map[string]*storage.ComplianceResultValue),
		}
		for i := 0; i < 50; i++ {
			results.ControlResults[fmt.Sprintf("%s:control%d", registeredStandardID, i)] = &storage.ComplianceResultValue{
				OverallState: storage.ComplianceState_COMPLIANCE_STATE_FAILURE,
			}
		}
		fixture := fixtures.GetDeployment()
		fixture.Id = uuid.NewV4().String()

		deploymentResults[fixture.GetId()] = results
		deployments[fixture.GetId()] = &storage.ComplianceDomain_Deployment{
			Id:          fixture.GetId(),
			NamespaceId: fixture.GetNamespaceId(),
			Name:        fixture.GetName(),
			Type:        fixture.GetType(),
			Namespace:   fixture.GetNamespace(),
			ClusterName: fixture.GetClusterName(),
		}
	}

	return &storage.ComplianceRunResults{
		Domain: &storage.ComplianceDomain{
			Cluster: &storage.ComplianceDomain_Cluster{
				Id: "cluster",
			},
			Deployments: deployments,
		},
		RunMetadata: &storage.ComplianceRunMetadata{
			StandardId: registeredStandardID,
		},
		DeploymentResults: deploymentResults,
	}
}

func BenchmarkAggregatedResults(b *testing.B) {
	result := mockBenchmarkRunResult()

	b.ResetTimer()
	a := &aggregatorImpl{
		standards: mockStandardsRepo(b),
	}
	for i := 0; i < b.N; i++ {
		a.getAggregatedResults(nil, storage.ComplianceAggregation_CHECK, []*storage.ComplianceRunResults{result}, &mask{})
	}
}
