package effectiveaccessscope

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	labelUtils "github.com/stackrox/rox/pkg/labels"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

////////////////////////////////////////////////////////////////////////////////
// Cluster and namespace configuration                                        //
//                                                                            //
// Object definitions in test_datasets.go file                                //
//                                                                            //
// Earth   { }                                                                //
//   Skunk Works   { focus: transportation, region: NA, clearance: yes }      //
//   Fraunhofer    { focus: applied_research, region: EU, clearance: no, founded: 1949 }
//   CERN          { focus: physics, region: EU }                             //
//   JPL           { focus: applied_research, region: NA }                    //
//                                                                            //
// Arrakis { focus: melange }                                                 //
//   Atreides      { focus: melange, homeworld: Caladan }                     //
//   Harkonnen     { focus: melange }                                         //
//   Spacing Guild { focus: transportation, region: dune_universe, depends-on: melange }
//   Bene Gesserit { region: dune_universe, alias: witches }                  //
//   Fremen        { }                                                        //
//                                                                            //
// Giedi=Prime { focus: melange }                                             //
//   Harkonnen     { focus: melange }
//                                                                            //

var clusters = []*storage.Cluster{
	clusterEarth,
	clusterArrakis,
	clusterGiediPrime,
}

var namespaces = []*storage.NamespaceMetadata{
	nsErrored,
	// Earth
	nsSkunkWorks,
	nsFraunhofer,
	nsCERN,
	nsJPL,
	// Arrakis
	nsAtreides,
	nsHarkonnen,
	nsSpacingGuild,
	nsBeneGesserit,
	nsFremen,
	// Giedi Prime
	nsHarkonnenAtHome,
}

////////////////////////////////////////////////////////////////////////////////
// Tests                                                                      //
//                                                                            //
// The tests closely resemble configuration scenarios and sample access       //
// scopes discussed in the design doc, see                                    //
//     https://docs.google.com/document/d/1GiPSPpRLm0M8NG9T7axxTc0grrNKriju8QxtbIJtl3s/edit#
//                                                                            //

const (
	accessScopeID   = "io.stackrox.authz.accessscope.test"
	accessScopeName = "test simple access scope"
)

const (
	opIN        = storage.SetBasedLabelSelector_IN
	opNOTIN     = storage.SetBasedLabelSelector_NOT_IN
	opEXISTS    = storage.SetBasedLabelSelector_EXISTS
	opNOTEXISTS = storage.SetBasedLabelSelector_NOT_EXISTS
)

func TestComputeEffectiveAccessScope(t *testing.T) {
	type testCase struct {
		desc      string
		scopeDesc string
		scopeStr  string
		scopeJSON string
		scope     *storage.SimpleAccessScope
		expected  *ScopeTree
		hasError  bool
		detail    v1.ComputeEffectiveAccessScopeRequest_Detail
	}

	testCaseMap := map[string]testCase{
		"no access scope includes nothing": {
			desc:      "no access scope includes nothing",
			scopeDesc: `nil => { }`,
			scopeStr:  "",
			scopeJSON: `{}`,
			scope:     nil,
			expected: &ScopeTree{
				State:           Excluded,
				clusterIDToName: clusterIDs,
				Clusters: map[string]*clustersScopeSubTree{
					"Earth": {
						State: Excluded,
						Namespaces: namespacesTree(
							excluded(nsSkunkWorks),
							excluded(nsFraunhofer),
							excluded(nsCERN),
							excluded(nsJPL),
						),
						Attributes: earthAttributes,
					},
					"Arrakis": {
						State: Excluded,
						Namespaces: namespacesTree(
							excluded(nsAtreides),
							excluded(nsHarkonnen),
							excluded(nsSpacingGuild),
							excluded(nsBeneGesserit),
							excluded(nsFremen),
						),
						Attributes: arrakisAttributes,
					},
					"Giedi=Prime": {
						State: Excluded,
						Namespaces: namespacesTree(
							excluded(nsHarkonnenAtHome),
						),
						Attributes: giediPrimeAttributes,
					},
					"Not Found": notFoundCluster,
				},
			},
			detail:   v1.ComputeEffectiveAccessScopeRequest_HIGH,
			hasError: false,
		},
		"empty access scope includes nothing": {
			desc:      "empty access scope includes nothing",
			scopeDesc: `∅ => { }`,
			scopeStr:  "",
			scopeJSON: `{}`,
			scope: &storage.SimpleAccessScope{
				Id:   accessScopeID,
				Name: accessScopeName,
			},
			expected: &ScopeTree{
				State:           Excluded,
				clusterIDToName: clusterIDs,
				Clusters: map[string]*clustersScopeSubTree{
					"Earth": {
						State: Excluded,
						Namespaces: namespacesTree(
							excluded(nsSkunkWorks),
							excluded(nsFraunhofer),
							excluded(nsCERN),
							excluded(nsJPL),
						),
						Attributes: earthAttributes,
					},
					"Arrakis": {
						State: Excluded,
						Namespaces: namespacesTree(
							excluded(nsAtreides),
							excluded(nsHarkonnen),
							excluded(nsSpacingGuild),
							excluded(nsBeneGesserit),
							excluded(nsFremen),
						),
						Attributes: arrakisAttributes,
					},
					"Giedi=Prime": {
						State: Excluded,
						Namespaces: namespacesTree(
							excluded(nsHarkonnenAtHome),
						),
						Attributes: giediPrimeAttributes,
					},
					"Not Found": notFoundCluster,
				},
			},
			detail:   v1.ComputeEffectiveAccessScopeRequest_HIGH,
			hasError: false,
		},
		"selector with empty requirements includes nothing": {
			desc:      "selector with empty requirements includes nothing",
			scopeDesc: `cluster.labels: ∅ => { }`,
			scopeStr:  "",
			scopeJSON: `{}`,
			scope: &storage.SimpleAccessScope{
				Id:   accessScopeID,
				Name: accessScopeName,
				Rules: &storage.SimpleAccessScope_Rules{
					IncludedClusters: []string{},
					ClusterLabelSelectors: []*storage.SetBasedLabelSelector{
						{},
					},
				},
			},
			expected: &ScopeTree{
				State:           Excluded,
				clusterIDToName: clusterIDs,
				Clusters: map[string]*clustersScopeSubTree{
					"Earth": {
						State: Excluded,
						Namespaces: namespacesTree(
							excluded(nsSkunkWorks),
							excluded(nsFraunhofer),
							excluded(nsCERN),
							excluded(nsJPL),
						),
						Attributes: earthAttributes,
					},
					"Arrakis": {
						State: Excluded,
						Namespaces: namespacesTree(
							excluded(nsAtreides),
							excluded(nsHarkonnen),
							excluded(nsSpacingGuild),
							excluded(nsBeneGesserit),
							excluded(nsFremen),
						),
						Attributes: arrakisAttributes,
					},
					"Giedi=Prime": {
						State: Excluded,
						Namespaces: namespacesTree(
							excluded(nsHarkonnenAtHome),
						),
						Attributes: giediPrimeAttributes,
					},
					"Not Found": notFoundCluster,
				},
			},
			detail:   v1.ComputeEffectiveAccessScopeRequest_HIGH,
			hasError: false,
		},
		"cluster included by name includes all its namespaces": {
			desc:      "cluster included by name includes all its namespaces",
			scopeDesc: `cluster: "Arrakis" => { "Arrakis::*" }`,
			scopeStr:  "Arrakis::*",
			scopeJSON: `{"Arrakis":["*"]}`,
			scope: &storage.SimpleAccessScope{
				Id:   accessScopeID,
				Name: accessScopeName,
				Rules: &storage.SimpleAccessScope_Rules{
					IncludedClusters: []string{"Arrakis"},
				},
			},
			expected: &ScopeTree{
				State:           Partial,
				clusterIDToName: clusterIDs,
				Clusters: map[string]*clustersScopeSubTree{
					"Earth": {
						State: Excluded,
						Namespaces: namespacesTree(
							excluded(nsSkunkWorks),
							excluded(nsFraunhofer),
							excluded(nsCERN),
							excluded(nsJPL),
						),
						Attributes: earthAttributes,
					},
					"Arrakis": {
						State: Included,
						Namespaces: namespacesTree(
							included(nsAtreides),
							included(nsHarkonnen),
							included(nsSpacingGuild),
							included(nsBeneGesserit),
							included(nsFremen),
						),
						Attributes: arrakisAttributes,
					},
					"Giedi=Prime": {
						State: Excluded,
						Namespaces: namespacesTree(
							excluded(nsHarkonnenAtHome),
						),
						Attributes: giediPrimeAttributes,
					},
					"Not Found": notFoundCluster,
				},
			},
			detail:   v1.ComputeEffectiveAccessScopeRequest_HIGH,
			hasError: false,
		},
		"cluster included by name (not matching k8s label syntax) includes all its namespaces": {
			desc:      "cluster included by name (not matching k8s label syntax) includes all its namespaces",
			scopeDesc: `cluster: "Giedi=Prime" => { "Giedi=Prime::*" }`,
			scopeStr:  "Giedi=Prime::*",
			scopeJSON: `{"Giedi=Prime":["*"]}`,
			scope: &storage.SimpleAccessScope{
				Id:   accessScopeID,
				Name: accessScopeName,
				Rules: &storage.SimpleAccessScope_Rules{
					IncludedClusters: []string{"Giedi=Prime"},
				},
			},
			expected: &ScopeTree{
				State:           Partial,
				clusterIDToName: clusterIDs,
				Clusters: map[string]*clustersScopeSubTree{
					"Earth": {
						State: Excluded,
						Namespaces: namespacesTree(
							excluded(nsSkunkWorks),
							excluded(nsFraunhofer),
							excluded(nsCERN),
							excluded(nsJPL),
						),
						Attributes: earthAttributes,
					},
					"Arrakis": {
						State: Excluded,
						Namespaces: namespacesTree(
							excluded(nsAtreides),
							excluded(nsHarkonnen),
							excluded(nsSpacingGuild),
							excluded(nsBeneGesserit),
							excluded(nsFremen),
						),
						Attributes: arrakisAttributes,
					},
					"Giedi=Prime": {
						State: Included,
						Namespaces: namespacesTree(
							included(nsHarkonnenAtHome),
						),
						Attributes: giediPrimeAttributes,
					},
					"Not Found": notFoundCluster,
				},
			},
			detail:   v1.ComputeEffectiveAccessScopeRequest_HIGH,
			hasError: false,
		},
		"cluster included have empty namespaces in minimal form": {
			desc:      "cluster included have empty namespaces in minimal form",
			scopeDesc: `cluster: "Arrakis" => { "Arrakis::*" }`,
			scopeStr:  "Arrakis::*",
			scopeJSON: `{"Arrakis":["*"]}`,
			scope: &storage.SimpleAccessScope{
				Id:   accessScopeID,
				Name: accessScopeName,
				Rules: &storage.SimpleAccessScope_Rules{
					IncludedClusters: []string{"Arrakis"},
				},
			},
			expected: &ScopeTree{
				State:           Partial,
				clusterIDToName: clusterIDs,
				Clusters: map[string]*clustersScopeSubTree{
					"Arrakis": {
						State:      Included,
						Attributes: treeNodeAttributes{ID: "planet.arrakis"},
					},
				},
			},
			detail:   v1.ComputeEffectiveAccessScopeRequest_MINIMAL,
			hasError: false,
		},
		"cluster(s) included by label include all underlying namespaces": {
			desc:      "cluster(s) included by label include all underlying namespaces",
			scopeDesc: `cluster.labels: focus in (melange) => { "Arrakis::*, Giedi=Prime::*" }`,
			scopeStr:  "Arrakis::*, Giedi=Prime::*",
			scopeJSON: `{"Arrakis":["*"],"Giedi=Prime":["*"]}`,
			scope: &storage.SimpleAccessScope{
				Id:   accessScopeID,
				Name: accessScopeName,
				Rules: &storage.SimpleAccessScope_Rules{
					ClusterLabelSelectors: labelUtils.LabelSelectors("focus", opIN, []string{"melange"}),
				},
			},
			expected: &ScopeTree{
				State:           Partial,
				clusterIDToName: clusterIDs,
				Clusters: map[string]*clustersScopeSubTree{
					"Earth": {
						State: Excluded,
						Namespaces: namespacesTree(
							excluded(nsSkunkWorks),
							excluded(nsFraunhofer),
							excluded(nsCERN),
							excluded(nsJPL),
						),
						Attributes: earthAttributes,
					},
					"Arrakis": {
						State: Included,
						Namespaces: namespacesTree(
							included(nsAtreides),
							included(nsHarkonnen),
							included(nsSpacingGuild),
							included(nsBeneGesserit),
							included(nsFremen),
						),
						Attributes: arrakisAttributes,
					},
					"Giedi=Prime": {
						State: Included,
						Namespaces: namespacesTree(
							included(nsHarkonnenAtHome),
						),
						Attributes: giediPrimeAttributes,
					},
					"Not Found": notFoundCluster,
				},
			},
			detail:   v1.ComputeEffectiveAccessScopeRequest_HIGH,
			hasError: false,
		},
		"namespace included by name (and cluster name) does not include anything else": {
			desc:      "namespace included by name does not include anything else",
			scopeDesc: `namespace: "Arrakis::Atreides" => { "Arrakis::Atreides" }`,
			scopeStr:  "Arrakis::Atreides",
			scopeJSON: `{"Arrakis":["Atreides"]}`,
			scope: &storage.SimpleAccessScope{
				Id:   accessScopeID,
				Name: accessScopeName,
				Rules: &storage.SimpleAccessScope_Rules{
					IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
						{
							ClusterName:   "Arrakis",
							NamespaceName: "Atreides",
						},
					},
				},
			},
			expected: &ScopeTree{
				State:           Partial,
				clusterIDToName: clusterIDs,
				Clusters: map[string]*clustersScopeSubTree{
					"Earth": {
						State: Excluded,
						Namespaces: namespacesTree(
							excluded(nsSkunkWorks),
							excluded(nsFraunhofer),
							excluded(nsCERN),
							excluded(nsJPL),
						),
						Attributes: earthAttributes,
					},
					"Arrakis": {
						State: Partial,
						Namespaces: namespacesTree(
							included(nsAtreides),
							excluded(nsHarkonnen),
							excluded(nsSpacingGuild),
							excluded(nsBeneGesserit),
							excluded(nsFremen),
						),
						Attributes: arrakisAttributes,
					},
					"Giedi=Prime": {
						State: Excluded,
						Namespaces: namespacesTree(
							excluded(nsHarkonnenAtHome),
						),
						Attributes: giediPrimeAttributes,
					},
					"Not Found": notFoundCluster,
				},
			},
			detail:   v1.ComputeEffectiveAccessScopeRequest_HIGH,
			hasError: false,
		},
		"namespace included by name (and cluster id) does not include anything else": {
			desc:      "namespace included by name does not include anything else",
			scopeDesc: `namespace: "Arrakis::Atreides" => { "Arrakis::Atreides" }`,
			scopeStr:  "Arrakis::Atreides",
			scopeJSON: `{"Arrakis":["Atreides"]}`,
			scope: &storage.SimpleAccessScope{
				Id:   accessScopeID,
				Name: accessScopeName,
				Rules: &storage.SimpleAccessScope_Rules{
					IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
						{
							ClusterId:     "planet.arrakis",
							NamespaceName: "Atreides",
						},
					},
				},
			},
			expected: &ScopeTree{
				State:           Partial,
				clusterIDToName: clusterIDs,
				Clusters: map[string]*clustersScopeSubTree{
					"Earth": {
						State: Excluded,
						Namespaces: namespacesTree(
							excluded(nsSkunkWorks),
							excluded(nsFraunhofer),
							excluded(nsCERN),
							excluded(nsJPL),
						),
						Attributes: earthAttributes,
					},
					"Arrakis": {
						State: Partial,
						Namespaces: namespacesTree(
							included(nsAtreides),
							excluded(nsHarkonnen),
							excluded(nsSpacingGuild),
							excluded(nsBeneGesserit),
							excluded(nsFremen),
						),
						Attributes: arrakisAttributes,
					},
					"Giedi=Prime": {
						State: Excluded,
						Namespaces: namespacesTree(
							excluded(nsHarkonnenAtHome),
						),
						Attributes: giediPrimeAttributes,
					},
					"Not Found": notFoundCluster,
				},
			},
			detail:   v1.ComputeEffectiveAccessScopeRequest_HIGH,
			hasError: false,
		},
		"namespace(s) included by label do not include anything else": {
			desc:      "namespace(s) included by label do not include anything else",
			scopeDesc: `namespace.labels: focus in (melange) => { "Arrakis::Atreides", "Arrakis::Harkonnen", "Giedi=Prime::Harkonnen" }`,
			scopeStr:  "Arrakis::{Atreides, Harkonnen}, Giedi=Prime::Harkonnen",
			scopeJSON: `{"Arrakis":["Atreides","Harkonnen"],"Giedi=Prime":["Harkonnen"]}`,
			scope: &storage.SimpleAccessScope{
				Id:   accessScopeID,
				Name: accessScopeName,
				Rules: &storage.SimpleAccessScope_Rules{
					NamespaceLabelSelectors: labelUtils.LabelSelectors("focus", opIN, []string{"melange"}),
				},
			},
			expected: &ScopeTree{
				State:           Partial,
				clusterIDToName: clusterIDs,
				Clusters: map[string]*clustersScopeSubTree{
					"Earth": {
						State: Excluded,
						Namespaces: namespacesTree(
							excluded(nsSkunkWorks),
							excluded(nsFraunhofer),
							excluded(nsCERN),
							excluded(nsJPL),
						),
						Attributes: earthAttributes,
					},
					"Arrakis": {
						State: Partial,
						Namespaces: namespacesTree(
							included(nsAtreides),
							included(nsHarkonnen),
							excluded(nsSpacingGuild),
							excluded(nsBeneGesserit),
							excluded(nsFremen),
						),
						Attributes: arrakisAttributes,
					},
					"Giedi=Prime": {
						State: Partial,
						Namespaces: namespacesTree(
							included(nsHarkonnenAtHome),
						),
						Attributes: giediPrimeAttributes,
					},
					"Not Found": notFoundCluster,
				},
			},
			detail:   v1.ComputeEffectiveAccessScopeRequest_HIGH,
			hasError: false,
		},
		"inclusion by label works across clusters": {
			desc:      "inclusion by label works across clusters",
			scopeDesc: `namespace.labels: focus in (transportation) => { "Earth::Skunk Works", "Arrakis::Spacing Guild" }`,
			scopeStr:  "Arrakis::Spacing Guild, Earth::Skunk Works",
			scopeJSON: `{"Arrakis":["Spacing Guild"],"Earth":["Skunk Works"]}`,
			scope: &storage.SimpleAccessScope{
				Id:   accessScopeID,
				Name: accessScopeName,
				Rules: &storage.SimpleAccessScope_Rules{
					NamespaceLabelSelectors: labelUtils.LabelSelectors("focus", opIN, []string{"transportation"}),
				},
			},
			expected: &ScopeTree{
				State:           Partial,
				clusterIDToName: clusterIDs,
				Clusters: map[string]*clustersScopeSubTree{
					"Earth": {
						State: Partial,
						Namespaces: namespacesTree(
							included(nsSkunkWorks),
							excluded(nsFraunhofer),
							excluded(nsCERN),
							excluded(nsJPL),
						),
						Attributes: earthAttributes,
					},
					"Arrakis": {
						State: Partial,
						Namespaces: namespacesTree(
							excluded(nsAtreides),
							excluded(nsHarkonnen),
							included(nsSpacingGuild),
							excluded(nsBeneGesserit),
							excluded(nsFremen),
						),
						Attributes: arrakisAttributes,
					},
					"Giedi=Prime": {
						State: Excluded,
						Namespaces: namespacesTree(
							excluded(nsHarkonnenAtHome),
						),
						Attributes: giediPrimeAttributes,
					},
					"Not Found": notFoundCluster,
				},
			},
			detail:   v1.ComputeEffectiveAccessScopeRequest_HIGH,
			hasError: false,
		},
		"inclusion by label groups labels by AND and set values by OR": {
			desc:      "inclusion by label groups labels by AND and set values by OR",
			scopeDesc: `namespace.labels: focus in (transportation, applied_research), region in (NA, dune_universe) => { "Earth::Skunk Works", "Earth::JPL", "Arrakis::Spacing Guild" }`,
			scopeStr:  "Arrakis::Spacing Guild, Earth::{JPL, Skunk Works}",
			scopeJSON: `{"Earth":["JPL","Skunk Works"],"Arrakis":["Spacing Guild"]}`,
			scope: &storage.SimpleAccessScope{
				Id:   accessScopeID,
				Name: accessScopeName,
				Rules: &storage.SimpleAccessScope_Rules{
					NamespaceLabelSelectors: []*storage.SetBasedLabelSelector{
						{
							Requirements: []*storage.SetBasedLabelSelector_Requirement{
								labelUtils.LabelSelectorRequirement("focus", opIN, []string{"transportation", "applied_research"}),
								labelUtils.LabelSelectorRequirement("region", opIN, []string{"NA", "dune_universe"}),
							},
						},
					},
				},
			},
			expected: &ScopeTree{
				State:           Partial,
				clusterIDToName: clusterIDs,
				Clusters: map[string]*clustersScopeSubTree{
					"Earth": {
						State: Partial,
						Namespaces: namespacesTree(
							included(nsSkunkWorks),
							excluded(nsFraunhofer),
							excluded(nsCERN),
							included(nsJPL),
						),
						Attributes: earthAttributes,
					},
					"Arrakis": {
						State: Partial,
						Namespaces: namespacesTree(
							excluded(nsAtreides),
							excluded(nsHarkonnen),
							included(nsSpacingGuild),
							excluded(nsBeneGesserit),
							excluded(nsFremen),
						),
						Attributes: arrakisAttributes,
					},
					"Giedi=Prime": {
						State: Excluded,
						Namespaces: namespacesTree(
							excluded(nsHarkonnenAtHome),
						),
						Attributes: giediPrimeAttributes,
					},
					"Not Found": notFoundCluster,
				},
			},
			detail:   v1.ComputeEffectiveAccessScopeRequest_HIGH,
			hasError: false,
		},
		"inclusion by label supports EXISTS, NOT_EXISTS, and NOTIN operators": {
			desc:      "inclusion by label supports EXISTS, NOT_EXISTS, and NOTIN operators",
			scopeDesc: `namespace.labels: focus notin (physics, melange), clearance, !founded => { "Earth::Skunk Works" }`,
			scopeStr:  "Earth::Skunk Works",
			scopeJSON: `{"Earth":["Skunk Works"]}`,
			scope: &storage.SimpleAccessScope{
				Id:   accessScopeID,
				Name: accessScopeName,
				Rules: &storage.SimpleAccessScope_Rules{
					NamespaceLabelSelectors: []*storage.SetBasedLabelSelector{
						{
							Requirements: []*storage.SetBasedLabelSelector_Requirement{
								labelUtils.LabelSelectorRequirement("focus", opNOTIN, []string{"physics", "melange"}),
								labelUtils.LabelSelectorRequirement("clearance", opEXISTS, nil),
								labelUtils.LabelSelectorRequirement("founded", opNOTEXISTS, nil),
							},
						},
					},
				},
			},
			expected: &ScopeTree{
				State:           Partial,
				clusterIDToName: clusterIDs,
				Clusters: map[string]*clustersScopeSubTree{
					"Earth": {
						State: Partial,
						Namespaces: namespacesTree(
							included(nsSkunkWorks),
							excluded(nsFraunhofer),
							excluded(nsCERN),
							excluded(nsJPL),
						),
						Attributes: earthAttributes,
					},
					"Arrakis": {
						State: Excluded,
						Namespaces: namespacesTree(
							excluded(nsAtreides),
							excluded(nsHarkonnen),
							excluded(nsSpacingGuild),
							excluded(nsBeneGesserit),
							excluded(nsFremen),
						),
						Attributes: arrakisAttributes,
					},
					"Giedi=Prime": {
						State: Excluded,
						Namespaces: namespacesTree(
							excluded(nsHarkonnenAtHome),
						),
						Attributes: giediPrimeAttributes,
					},
					"Not Found": notFoundCluster,
				},
			},
			detail:   v1.ComputeEffectiveAccessScopeRequest_HIGH,
			hasError: false,
		},
		"multiple label selectors are joined by OR": {
			desc:      "multiple label selectors are joined by OR",
			scopeDesc: `namespace.labels: focus in (transportation), region in (NA) OR region in (EU) OR founded in (1949) => { "Earth::Skunk Works", "Earth::Fraunhofer", "Earth::CERN" }`,
			scopeStr:  "Earth::{CERN, Fraunhofer, Skunk Works}",
			scopeJSON: `{"Earth":["CERN","Fraunhofer","Skunk Works"]}`,
			scope: &storage.SimpleAccessScope{
				Id:   accessScopeID,
				Name: accessScopeName,
				Rules: &storage.SimpleAccessScope_Rules{
					NamespaceLabelSelectors: []*storage.SetBasedLabelSelector{
						{
							Requirements: []*storage.SetBasedLabelSelector_Requirement{
								labelUtils.LabelSelectorRequirement("focus", opIN, []string{"transportation"}),
								labelUtils.LabelSelectorRequirement("region", opIN, []string{"NA"}),
							},
						},
						labelUtils.LabelSelector("region", opIN, []string{"EU"}),
						labelUtils.LabelSelector("founded", opIN, []string{"1949"}),
					},
				},
			},
			expected: &ScopeTree{
				State:           Partial,
				clusterIDToName: clusterIDs,
				Clusters: map[string]*clustersScopeSubTree{
					"Earth": {
						State: Partial,
						Namespaces: namespacesTree(
							included(nsSkunkWorks),
							included(nsFraunhofer),
							included(nsCERN),
							excluded(nsJPL),
						),
						Attributes: earthAttributes,
					},
					"Arrakis": {
						State: Excluded,
						Namespaces: namespacesTree(
							excluded(nsAtreides),
							excluded(nsHarkonnen),
							excluded(nsSpacingGuild),
							excluded(nsBeneGesserit),
							excluded(nsFremen),
						),
						Attributes: arrakisAttributes,
					},
					"Giedi=Prime": {
						State: Excluded,
						Namespaces: namespacesTree(
							excluded(nsHarkonnenAtHome),
						),
						Attributes: giediPrimeAttributes,
					},
					"Not Found": notFoundCluster,
				},
			},
			detail:   v1.ComputeEffectiveAccessScopeRequest_HIGH,
			hasError: false,
		},
		"rules are joined by OR": {
			desc:      "rules are joined by OR",
			scopeDesc: `namespace: "Earth::Skunk Works" OR cluster.labels: focus in (melange) OR namespace.labels: region in (EU) => { "Earth::Skunk Works", "Earth::Fraunhofer", "Earth::CERN", "Arrakis::*", "Giedi=Prime::*" }`,
			scopeStr:  "Arrakis::*, Earth::{CERN, Fraunhofer, Skunk Works}, Giedi=Prime::*",
			scopeJSON: `{"Earth":["CERN","Fraunhofer","Skunk Works"],"Arrakis":["*"],"Giedi=Prime":["*"]}`,
			scope: &storage.SimpleAccessScope{
				Id:   accessScopeID,
				Name: accessScopeName,
				Rules: &storage.SimpleAccessScope_Rules{
					IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
						{
							ClusterName:   "Earth",
							NamespaceName: "Skunk Works",
						},
					},
					ClusterLabelSelectors:   labelUtils.LabelSelectors("focus", opIN, []string{"melange"}),
					NamespaceLabelSelectors: labelUtils.LabelSelectors("region", opIN, []string{"EU"}),
				},
			},
			expected: &ScopeTree{
				State:           Partial,
				clusterIDToName: clusterIDs,
				Clusters: map[string]*clustersScopeSubTree{
					"Earth": {
						State: Partial,
						Namespaces: namespacesTree(
							included(nsSkunkWorks),
							included(nsFraunhofer),
							included(nsCERN),
							excluded(nsJPL),
						),
						Attributes: earthAttributes,
					},
					"Arrakis": {
						State: Included,
						Namespaces: namespacesTree(
							included(nsAtreides),
							included(nsHarkonnen),
							included(nsSpacingGuild),
							included(nsBeneGesserit),
							included(nsFremen),
						),
						Attributes: arrakisAttributes,
					},
					"Giedi=Prime": {
						State: Included,
						Namespaces: namespacesTree(
							included(nsHarkonnenAtHome),
						),
						Attributes: giediPrimeAttributes,
					},
					"Not Found": notFoundCluster,
				},
			},
			detail:   v1.ComputeEffectiveAccessScopeRequest_HIGH,
			hasError: false,
		},
		"all excluded namespaces are removed from cluster in minimal form": {
			desc:      "all excluded namespaces are removed from cluster in minimal form",
			scopeDesc: `"namespace.labels: focus in (melange)" => { "Arrakis::Atreides", "Arrakis::Harkonnen", "Giedi=Prime::Harkonnen" }`,
			scopeStr:  "Arrakis::{Atreides, Harkonnen}, Giedi=Prime::Harkonnen",
			scopeJSON: `{"Arrakis":["Atreides","Harkonnen"],"Giedi=Prime":["Harkonnen"]}`,
			scope: &storage.SimpleAccessScope{
				Id:   accessScopeID,
				Name: accessScopeName,
				Rules: &storage.SimpleAccessScope_Rules{
					NamespaceLabelSelectors: labelUtils.LabelSelectors("focus", opIN, []string{"melange"}),
				},
			},
			expected: &ScopeTree{
				State:           Partial,
				clusterIDToName: clusterIDs,
				Clusters: map[string]*clustersScopeSubTree{
					"Arrakis": {
						State: Partial,
						Namespaces: map[string]*namespacesScopeSubTree{
							"Atreides": {
								State:      Included,
								Attributes: treeNodeAttributes{ID: "house.atreides"},
							},
							"Harkonnen": {
								State:      Included,
								Attributes: treeNodeAttributes{ID: "house.harkonnen"},
							},
						},
						Attributes: treeNodeAttributes{ID: "planet.arrakis"},
					},
					"Giedi=Prime": {
						State: Partial,
						Namespaces: map[string]*namespacesScopeSubTree{
							"Harkonnen": {
								State:      Included,
								Attributes: treeNodeAttributes{ID: "house.harkonnen"},
							},
						},
						Attributes: treeNodeAttributes{ID: "planet.giedi=prime"},
					},
				},
			},
			detail:   v1.ComputeEffectiveAccessScopeRequest_MINIMAL,
			hasError: false,
		},
		"no labels in standard form": {
			desc:      "no labels in standard form",
			scopeDesc: `"namespace.labels: focus in (melange)" => { "Arrakis::Atreides", "Arrakis::Harkonnen", "Giedi=Prime::Harkonnen" }`,
			scopeStr:  "Arrakis::{Atreides, Harkonnen}, Giedi=Prime::Harkonnen",
			scopeJSON: `{"Arrakis":["Atreides","Harkonnen"],"Giedi=Prime":["Harkonnen"]}`,
			scope: &storage.SimpleAccessScope{
				Id:   accessScopeID,
				Name: accessScopeName,
				Rules: &storage.SimpleAccessScope_Rules{
					NamespaceLabelSelectors: labelUtils.LabelSelectors("focus", opIN, []string{"melange"}),
				},
			},
			expected: &ScopeTree{
				State:           Partial,
				clusterIDToName: clusterIDs,
				Clusters: map[string]*clustersScopeSubTree{
					"Earth": {
						State: Excluded,
						Namespaces: namespacesTree(
							excludedStandard(nsSkunkWorks),
							excludedStandard(nsFraunhofer),
							excludedStandard(nsCERN),
							excludedStandard(nsJPL),
						),
						Attributes: earthAttributes,
					},
					"Arrakis": {
						State: Partial,
						Namespaces: namespacesTree(
							includedStandard(nsAtreides),
							includedStandard(nsHarkonnen),
							excludedStandard(nsSpacingGuild),
							excludedStandard(nsBeneGesserit),
							excludedStandard(nsFremen),
						),
						Attributes: treeNodeAttributes{ID: "planet.arrakis", Name: "Arrakis"},
					},
					"Giedi=Prime": {
						State: Partial,
						Namespaces: namespacesTree(
							includedStandard(nsHarkonnenAtHome),
						),
						Attributes: treeNodeAttributes{ID: "planet.giedi=prime", Name: "Giedi=Prime"},
					},
					"Not Found": {
						State:      Excluded,
						Namespaces: namespacesTree(excludedStandard(nsErrored)),
						Attributes: treeNodeAttributes{
							Name: "Not Found",
						},
					},
				},
			},
			detail:   v1.ComputeEffectiveAccessScopeRequest_STANDARD,
			hasError: false,
		},
		"no key in cluster label selector": {
			desc: "no key in cluster label selector",
			scope: &storage.SimpleAccessScope{
				Id:   accessScopeID,
				Name: accessScopeName,
				Rules: &storage.SimpleAccessScope_Rules{
					ClusterLabelSelectors: labelUtils.LabelSelectors("", opIN, []string{"melange"}),
				},
			},
			hasError: true,
		},
		"no key in namespace label selector": {
			desc: "no key in namespace label selector",
			scope: &storage.SimpleAccessScope{
				Id:   accessScopeID,
				Name: accessScopeName,
				Rules: &storage.SimpleAccessScope_Rules{
					NamespaceLabelSelectors: labelUtils.LabelSelectors("", opIN, []string{"melange"}),
				},
			},
			hasError: true,
		},
	}

	for desc, tc := range testCaseMap {
		t.Run(desc, func(t *testing.T) {
			var clonedClusters []*storage.Cluster
			inputClusters := make([]Cluster, 0, len(clusters))
			for _, c := range clusters {
				clonedClusters = append(clonedClusters, c.CloneVT())
				inputClusters = append(inputClusters, c)
			}

			var clonedNamespaces []*storage.NamespaceMetadata
			inputNamespaces := make([]Namespace, 0, len(namespaces))
			for _, ns := range namespaces {
				clonedNamespaces = append(clonedNamespaces, ns.CloneVT())
				inputNamespaces = append(inputNamespaces, ns)
			}

			result, err := ComputeEffectiveAccessScope(tc.scope.GetRules(), inputClusters, inputNamespaces, tc.detail)
			assert.Truef(t, tc.hasError == (err != nil), "error: %v", err)
			assert.Equal(t, tc.expected, result, tc.scopeDesc)
			protoassert.SlicesEqual(t, clusters, clonedClusters, "clusters have been modified")
			protoassert.SlicesEqual(t, namespaces, clonedNamespaces, "namespaces have been modified")
			if tc.expected != nil {
				assert.Equal(t, tc.scopeStr, result.String())

				json, err := result.ToJSON()
				assert.NoError(t, err)
				assert.JSONEq(t, tc.scopeJSON, json)

				assert.Nil(t, result.GetClusterByID("unknown cluster id"))
				for _, c := range clonedClusters {
					assert.Equal(t, result.GetClusterByID(c.GetId()), tc.expected.Clusters[c.GetName()])
				}
			}
		})
	}
}

func TestMergeScopeTree(t *testing.T) {
	testCases := []struct {
		name    string
		a, b, c *ScopeTree
	}{
		{
			name: "∅ + all = all",
			a:    DenyAllEffectiveAccessScope(),
			b:    UnrestrictedEffectiveAccessScope(),
			c:    UnrestrictedEffectiveAccessScope(),
		},
		{
			name: "all + ∅ = all",
			a:    UnrestrictedEffectiveAccessScope(),
			b:    DenyAllEffectiveAccessScope(),
			c:    UnrestrictedEffectiveAccessScope(),
		},
		{
			name: "∅ + ∅ = ∅",
			a:    DenyAllEffectiveAccessScope(),
			b:    DenyAllEffectiveAccessScope(),
			c:    DenyAllEffectiveAccessScope(),
		},
		{
			name: "merge namespaces and clusters",
			a: &ScopeTree{
				State: Partial,
				clusterIDToName: map[string]string{
					"clusterid.earth":    "Earth",
					"clusterid.arrakis":  "Arrakis",
					"clusterid.notfound": "Not Found",
				},
				Clusters: map[string]*clustersScopeSubTree{
					"Earth": {
						State: Excluded,
					},
					"Arrakis": {
						State:      Partial,
						Attributes: treeNodeAttributes{ID: "planet.arrakis", Name: "Arrakis"},
					},
					"Not Found": {
						State:      Excluded,
						Namespaces: namespacesTree(excludedStandard(nsErrored)),
						Attributes: treeNodeAttributes{
							Name: "Not Found",
						},
					},
				},
			},
			b: &ScopeTree{
				State: Partial,
				clusterIDToName: map[string]string{
					"clusterid.earth":   "Earth",
					"clusterid.arrakis": "Arrakis",
				},
				Clusters: map[string]*clustersScopeSubTree{
					"Earth": {
						State: Partial,
						Namespaces: namespacesTree(
							includedStandard(nsSkunkWorks),
							includedStandard(nsFraunhofer),
							includedStandard(nsCERN),
							excludedStandard(nsJPL),
						),
					},
					"Arrakis": {
						State:      Partial,
						Attributes: treeNodeAttributes{ID: "planet.arrakis", Name: "Arrakis"},
					},
				},
			},
			c: &ScopeTree{
				State: Partial,
				clusterIDToName: map[string]string{
					"clusterid.earth":    "Earth",
					"clusterid.arrakis":  "Arrakis",
					"clusterid.notfound": "Not Found",
				},
				Clusters: map[string]*clustersScopeSubTree{
					"Arrakis": {
						State: Partial,
					},
					"Earth": {
						State: Partial,
						Namespaces: namespacesTree(
							includedStandard(nsSkunkWorks),
							includedStandard(nsFraunhofer),
							includedStandard(nsCERN),
						),
					},
				},
			},
		},
		{
			name: "short circuit when cluster is included",
			a: &ScopeTree{
				State: Partial,
				clusterIDToName: map[string]string{
					"clusterid.earth":   "Earth",
					"clusterid.arrakis": "Arrakis",
				},
				Clusters: map[string]*clustersScopeSubTree{
					"Earth": {
						State: Included,
						Namespaces: namespacesTree(
							includedStandard(nsSkunkWorks),
							includedStandard(nsFraunhofer),
							includedStandard(nsCERN),
						),
					},
					"Arrakis": {
						State:      Partial,
						Attributes: treeNodeAttributes{ID: "planet.arrakis", Name: "Arrakis"},
					},
				},
			},
			b: &ScopeTree{
				State: Partial,
				clusterIDToName: map[string]string{
					"clusterid.earth":    "Earth",
					"clusterid.arrakis":  "Arrakis",
					"clusterid.notfound": "Not Found",
				},
				Clusters: map[string]*clustersScopeSubTree{
					"Earth": {
						State: Excluded,
					},
					"Arrakis": {
						State:      Partial,
						Attributes: treeNodeAttributes{ID: "planet.arrakis", Name: "Arrakis"},
					},
					"Not Found": {
						State:      Excluded,
						Namespaces: namespacesTree(excludedStandard(nsErrored)),
						Attributes: treeNodeAttributes{
							Name: "Not Found",
						},
					},
				},
			},
			c: &ScopeTree{
				State: Partial,
				clusterIDToName: map[string]string{
					"clusterid.earth":    "Earth",
					"clusterid.arrakis":  "Arrakis",
					"clusterid.notfound": "Not Found",
				},
				Clusters: map[string]*clustersScopeSubTree{
					"Arrakis": {
						State: Partial,
					},
					"Earth": {
						State: Included,
					},
				},
			},
		},
		{
			name: "∅ + something = something",
			a:    DenyAllEffectiveAccessScope(),
			b: &ScopeTree{
				State: Partial,
				clusterIDToName: map[string]string{
					"clusterid.earth":   "Earth",
					"clusterid.arrakis": "Arrakis",
				},
				Clusters: map[string]*clustersScopeSubTree{
					"Earth": {
						State: Partial,
						Namespaces: namespacesTree(
							includedStandard(nsSkunkWorks),
							includedStandard(nsFraunhofer),
							includedStandard(nsCERN),
							excludedStandard(nsJPL),
						),
					},
					"Arrakis": {
						State:      Partial,
						Attributes: treeNodeAttributes{ID: "planet.arrakis", Name: "Arrakis"},
					},
				},
			},
			c: &ScopeTree{
				State: Partial,
				clusterIDToName: map[string]string{
					"clusterid.earth":   "Earth",
					"clusterid.arrakis": "Arrakis",
				},
				Clusters: map[string]*clustersScopeSubTree{
					"Earth": {
						State: Partial,
						Namespaces: namespacesTree(
							includedStandard(nsSkunkWorks),
							includedStandard(nsFraunhofer),
							includedStandard(nsCERN),
						),
					},
					"Arrakis": {
						State:      Partial,
						Attributes: treeNodeAttributes{ID: "planet.arrakis", Name: "Arrakis"},
					},
				},
			},
		},
		{
			name: "∅ + partial = partial",
			a:    DenyAllEffectiveAccessScope(),
			b: &ScopeTree{
				State: Partial,
				clusterIDToName: map[string]string{
					"clusterid.earth": "Earth",
				},
				Clusters: map[string]*clustersScopeSubTree{
					"Earth": {
						State: Included,
					},
				},
			},
			c: &ScopeTree{
				State: Partial,
				clusterIDToName: map[string]string{
					"clusterid.earth": "Earth",
				},
				Clusters: map[string]*clustersScopeSubTree{
					"Earth": {
						State: Included,
					},
				},
			},
		},
		{
			name: "nil + nil = nil",
			a:    nil,
			b:    nil,
			c:    nil,
		},
		{
			name: "∅ + nil = ∅",
			a:    DenyAllEffectiveAccessScope(),
			b:    nil,
			c:    DenyAllEffectiveAccessScope(),
		},
		{
			name: "excluded + included = included",
			a: &ScopeTree{
				State: Partial,
			},
			b: &ScopeTree{
				State: Partial,
				clusterIDToName: map[string]string{
					"clusterid.earth": "Earth",
				},
				Clusters: map[string]*clustersScopeSubTree{
					"Earth": {
						State: Included,
					},
				},
			},
			c: &ScopeTree{
				State: Partial,
				clusterIDToName: map[string]string{
					"clusterid.earth": "Earth",
				},
				Clusters: map[string]*clustersScopeSubTree{
					"Earth": {
						State: Included,
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		a, b, c := tc.a, tc.b, tc.c
		t.Run(tc.name, func(t *testing.T) {

			a.Merge(b)

			aJSON, err := a.ToJSON()
			assert.NoError(t, err)
			cJSON, err := c.ToJSON()
			assert.NoError(t, err)
			assert.JSONEq(t, cJSON, aJSON)

			if b != nil {
				for _, cluster := range b.Clusters {
					cluster.State = Excluded
					for name := range cluster.Namespaces {
						cluster.Namespaces[name].State = Excluded
					}
				}
			}
			aJSON, err = a.ToJSON()
			assert.NoError(t, err)
			assert.JSONEq(t, cJSON, aJSON, "values were not copied")
		})
	}
}

func TestUnrestrictedEffectiveAccessScope(t *testing.T) {
	expected := &ScopeTree{
		State:           Included,
		Clusters:        make(map[string]*clustersScopeSubTree),
		clusterIDToName: make(map[string]string),
	}
	expectedStr := "*::*"
	expectedJSON := `{"*":["*"]}`

	result := UnrestrictedEffectiveAccessScope()
	assert.Equal(t, expected, result)
	assert.Equal(t, expectedStr, result.String())

	json, err := result.ToJSON()
	assert.NoError(t, err)
	assert.JSONEq(t, expectedJSON, json)
}

// TestNewUnvalidatedRequirement covers both use cases we currently have:
//   - label value contains a forbidden token (scope separator);
//   - label value length exceeds 63 characters.
func TestNewUnvalidatedRequirement(t *testing.T) {
	validKey := "stackrox.io/authz.metadata.test.valid.key"
	operatorIn := selection.In
	tooLongValue := "i.am.a.fully.qualified.scope.name.for.some.namespace.longer.than.63"
	invalidTokenValue := "toto" + scopeSeparator + "tutu"

	// Check *labels.Requirement can be created with invalid values.
	req, err := newUnvalidatedRequirement(validKey, operatorIn, []string{tooLongValue, invalidTokenValue})
	assert.NoError(t, err)

	// Check the selector built from *labels.Requirement instance works.
	selector := labels.NewSelector()
	selector = selector.Add(*req)

	testCasesGood := []labels.Set{
		labels.Set(map[string]string{validKey: tooLongValue}),
		labels.Set(map[string]string{validKey: invalidTokenValue}),
	}
	for _, tc := range testCasesGood {
		t.Run(tc.String(), func(t *testing.T) {
			assert.Truef(t, selector.Matches(tc), "%q should match %q", selector.String(), tc.String())
		})
	}

	testCasesBad := []labels.Set{
		{},
		labels.Set(map[string]string{"random.key": tooLongValue}),
	}
	for _, tc := range testCasesBad {
		t.Run(tc.String(), func(t *testing.T) {
			assert.Falsef(t, selector.Matches(tc), "%q should not match %q", selector.String(), tc.String())
		})
	}
}

func TestSelectorsMatchCluster(t *testing.T) {
	focusOnMelangeRequirement, err := labels.NewRequirement("focus", selection.Equals, []string{"melange"})
	require.NoError(t, err)
	require.NotNil(t, focusOnMelangeRequirement)

	for name, tc := range map[string]struct {
		ruleSelector *selectors
		cluster      *storage.Cluster
		expected     scopeState
	}{
		"nil selector always excludes cluster": {
			ruleSelector: nil,
			cluster:      clusterEarth,
			expected:     Excluded,
		},
		"cluster matched by ID is included": {
			ruleSelector: &selectors{
				clustersByID: map[string]bool{
					clusterEarth.GetId(): true,
				},
			},
			cluster:  clusterEarth,
			expected: Included,
		},
		"cluster matched by name (matching k8s syntax) is included": {
			ruleSelector: &selectors{
				clustersByName: map[string]bool{
					clusterEarth.GetName(): true,
				},
			},
			cluster:  clusterEarth,
			expected: Included,
		},
		"cluster matched by name (NOT matching k8s syntax) is included": {
			ruleSelector: &selectors{
				clustersByName: map[string]bool{
					clusterGiediPrime.GetName(): true,
				},
			},
			cluster:  clusterGiediPrime,
			expected: Included,
		},
		"cluster matched by label is included": {
			ruleSelector: &selectors{
				clustersByLabel: []labels.Selector{
					labels.NewSelector().Add(*focusOnMelangeRequirement),
				},
			},
			cluster:  clusterArrakis,
			expected: Included,
		},
		"cluster NOT matched by label is excluded": {
			ruleSelector: &selectors{
				clustersByLabel: []labels.Selector{
					labels.NewSelector().Add(*focusOnMelangeRequirement),
				},
			},
			cluster:  clusterEarth,
			expected: Excluded,
		},
	} {
		t.Run(name, func(it *testing.T) {
			result := tc.ruleSelector.matchCluster(tc.cluster)
			assert.Equal(it, tc.expected, result)
		})
	}
}

func TestSelectorsMatchNamespace(t *testing.T) {
	focusOnMelangeRequirement, err := labels.NewRequirement("focus", selection.Equals, []string{"melange"})
	require.NoError(t, err)
	require.NotNil(t, focusOnMelangeRequirement)

	for name, tc := range map[string]struct {
		ruleSelectors *selectors
		namespace     *storage.NamespaceMetadata
		expected      scopeState
	}{
		"nil selector always exclude namespaces": {
			ruleSelectors: nil,
			namespace:     nsSkunkWorks,
			expected:      Excluded,
		},
		"namespace matched by cluster ID is included": {
			ruleSelectors: &selectors{
				namespacesByClusterID: map[string]map[string]bool{
					nsSkunkWorks.GetClusterId(): {
						nsSkunkWorks.GetName(): true,
					},
				},
			},
			namespace: nsSkunkWorks,
			expected:  Included,
		},
		"namespace matched by cluster name is included": {
			ruleSelectors: &selectors{
				namespacesByClusterName: map[string]map[string]bool{
					nsSkunkWorks.GetClusterName(): {
						nsSkunkWorks.GetName(): true,
					},
				},
			},
			namespace: nsSkunkWorks,
			expected:  Included,
		},
		"namespace matched by label is included": {
			ruleSelectors: &selectors{
				namespacesByLabel: []labels.Selector{
					labels.NewSelector().Add(*focusOnMelangeRequirement),
				},
			},
			namespace: nsAtreides,
			expected:  Included,
		},
		"namespace NOT matched by label is included": {
			ruleSelectors: &selectors{
				namespacesByLabel: []labels.Selector{
					labels.NewSelector().Add(*focusOnMelangeRequirement),
				},
			},
			namespace: nsSkunkWorks,
			expected:  Excluded,
		},
	} {
		t.Run(name, func(it *testing.T) {
			result := tc.ruleSelectors.matchNamespace(tc.namespace)
			assert.Equal(it, tc.expected, result)
		})
	}
}

func TestScopeTreePopulateStateForCluster(t *testing.T) {
	for name, tc := range map[string]struct {
		root          *ScopeTree
		ruleSelectors *selectors
		cluster       Cluster
		detail        v1.ComputeEffectiveAccessScopeRequest_Detail
		expected      *ScopeTree
	}{
		"matching cluster is added to the scope tree if not existing": {
			root:          newEffectiveAccessScopeTree(Excluded),
			ruleSelectors: &selectors{clustersByName: map[string]bool{clusterEarth.GetName(): true}},
			cluster:       clusterEarth,
			detail:        v1.ComputeEffectiveAccessScopeRequest_HIGH,
			expected: &ScopeTree{
				State: Excluded,
				Clusters: map[string]*clustersScopeSubTree{
					clusterEarth.GetName(): {
						State:      Included,
						Namespaces: make(map[string]*namespacesScopeSubTree),
						Attributes: treeNodeAttributes{
							ID:     clusterEarth.GetId(),
							Name:   clusterEarth.GetName(),
							Labels: clusterEarth.GetLabels(),
						},
					},
				},
				clusterIDToName: map[string]string{
					clusterEarth.GetId(): clusterEarth.GetName(),
				},
			},
		},
		"matching cluster state is updated if previously computed state is lower": {
			root: &ScopeTree{
				State: Excluded,
				Clusters: map[string]*clustersScopeSubTree{
					clusterEarth.GetName(): {
						State:      Excluded,
						Namespaces: make(map[string]*namespacesScopeSubTree),
						Attributes: treeNodeAttributes{},
					},
				},
				clusterIDToName: map[string]string{
					clusterEarth.GetId(): clusterEarth.GetName(),
				},
			},
			ruleSelectors: &selectors{clustersByName: map[string]bool{clusterEarth.GetName(): true}},
			cluster:       clusterEarth,
			detail:        v1.ComputeEffectiveAccessScopeRequest_MINIMAL,
			expected: &ScopeTree{
				State: Excluded,
				Clusters: map[string]*clustersScopeSubTree{
					clusterEarth.GetName(): {
						State:      Included,
						Namespaces: make(map[string]*namespacesScopeSubTree),
						Attributes: treeNodeAttributes{
							ID: clusterEarth.GetId(),
						},
					},
				},
				clusterIDToName: map[string]string{
					clusterEarth.GetId(): clusterEarth.GetName(),
				},
			},
		},
		"cluster state is NOT updated if previously computed state is greater": {
			root: &ScopeTree{
				State: Excluded,
				Clusters: map[string]*clustersScopeSubTree{
					clusterEarth.GetName(): {
						State:      Included,
						Namespaces: make(map[string]*namespacesScopeSubTree),
						Attributes: treeNodeAttributes{},
					},
				},
				clusterIDToName: map[string]string{
					clusterEarth.GetId(): clusterEarth.GetName(),
				},
			},
			ruleSelectors: nil, // Selection rules exclude the cluster.
			cluster:       clusterEarth,
			detail:        v1.ComputeEffectiveAccessScopeRequest_MINIMAL,
			expected: &ScopeTree{
				State: Excluded,
				Clusters: map[string]*clustersScopeSubTree{
					clusterEarth.GetName(): {
						State:      Included,
						Namespaces: make(map[string]*namespacesScopeSubTree),
						Attributes: treeNodeAttributes{},
					},
				},
				clusterIDToName: map[string]string{
					clusterEarth.GetId(): clusterEarth.GetName(),
				},
			},
		},
	} {
		t.Run(name, func(it *testing.T) {
			tc.root.populateStateForCluster(tc.cluster, tc.ruleSelectors, tc.detail)
			assert.Equal(it, tc.expected, tc.root)
		})
	}
}

func TestClusterScopeSubTreePopulateStateForNamespace(t *testing.T) {
	for name, tc := range map[string]struct {
		clusterSubTree *clustersScopeSubTree
		ruleSelectors  *selectors
		namespace      Namespace
		detail         v1.ComputeEffectiveAccessScopeRequest_Detail
		expected       *clustersScopeSubTree
	}{
		"Namespace from included cluster is added as included regardless of the selection rules": {
			clusterSubTree: &clustersScopeSubTree{
				State:      Included,
				Namespaces: make(map[string]*namespacesScopeSubTree),
			},
			ruleSelectors: nil, // nil selector excluded the namespace
			namespace:     nsJPL,
			detail:        v1.ComputeEffectiveAccessScopeRequest_HIGH,
			expected: &clustersScopeSubTree{
				State: Included,
				Namespaces: map[string]*namespacesScopeSubTree{
					nsJPL.GetName(): {
						State:      Included,
						Attributes: nodeAttributesForNamespace(nsJPL, v1.ComputeEffectiveAccessScopeRequest_HIGH),
					},
				},
			},
		},
		"State of already added namespace is updated if higher": {
			clusterSubTree: &clustersScopeSubTree{
				State: Excluded,
				Namespaces: map[string]*namespacesScopeSubTree{
					nsJPL.GetName(): {
						State:      Excluded,
						Attributes: nodeAttributesForNamespace(nsJPL, v1.ComputeEffectiveAccessScopeRequest_HIGH),
					},
				},
			},
			ruleSelectors: &selectors{
				namespacesByClusterName: map[string]map[string]bool{
					nsJPL.GetClusterName(): {nsJPL.GetName(): true},
				},
			},
			namespace: nsJPL,
			detail:    v1.ComputeEffectiveAccessScopeRequest_MINIMAL,
			expected: &clustersScopeSubTree{
				State: Excluded,
				Namespaces: map[string]*namespacesScopeSubTree{
					nsJPL.GetName(): {
						State:      Included,
						Attributes: nodeAttributesForNamespace(nsJPL, v1.ComputeEffectiveAccessScopeRequest_HIGH),
					},
				},
			},
		},
		"State of already added namespace is NOT updated if lower": {
			clusterSubTree: &clustersScopeSubTree{
				State: Excluded,
				Namespaces: map[string]*namespacesScopeSubTree{
					nsJPL.GetName(): {
						State:      Included,
						Attributes: nodeAttributesForNamespace(nsJPL, v1.ComputeEffectiveAccessScopeRequest_HIGH),
					},
				},
			},
			ruleSelectors: nil, // recomputed namespace state is Excluded
			namespace:     nsJPL,
			detail:        v1.ComputeEffectiveAccessScopeRequest_MINIMAL,
			expected: &clustersScopeSubTree{
				State: Excluded,
				Namespaces: map[string]*namespacesScopeSubTree{
					nsJPL.GetName(): {
						State:      Included,
						Attributes: nodeAttributesForNamespace(nsJPL, v1.ComputeEffectiveAccessScopeRequest_HIGH),
					},
				},
			},
		},
		"New namespace is added with computed state (Excluded)": {
			clusterSubTree: &clustersScopeSubTree{
				State:      Excluded,
				Namespaces: make(map[string]*namespacesScopeSubTree),
			},
			ruleSelectors: nil, // nil selector excluded the namespace
			namespace:     nsJPL,
			detail:        v1.ComputeEffectiveAccessScopeRequest_HIGH,
			expected: &clustersScopeSubTree{
				State: Excluded,
				Namespaces: map[string]*namespacesScopeSubTree{
					nsJPL.GetName(): {
						State:      Excluded,
						Attributes: nodeAttributesForNamespace(nsJPL, v1.ComputeEffectiveAccessScopeRequest_HIGH),
					},
				},
			},
		},
		"New namespace is added with computed state (Included)": {

			clusterSubTree: &clustersScopeSubTree{
				State:      Excluded,
				Namespaces: make(map[string]*namespacesScopeSubTree),
			},
			ruleSelectors: &selectors{
				namespacesByClusterName: map[string]map[string]bool{
					nsJPL.GetClusterName(): {nsJPL.GetName(): true},
				},
			},
			namespace: nsJPL,
			detail:    v1.ComputeEffectiveAccessScopeRequest_STANDARD,
			expected: &clustersScopeSubTree{
				State: Excluded,
				Namespaces: map[string]*namespacesScopeSubTree{
					nsJPL.GetName(): {
						State:      Included,
						Attributes: nodeAttributesForNamespace(nsJPL, v1.ComputeEffectiveAccessScopeRequest_STANDARD),
					},
				},
			},
		},
	} {
		t.Run(name, func(it *testing.T) {
			tc.clusterSubTree.populateStateForNamespace(tc.namespace, tc.ruleSelectors, tc.detail)
			assert.Equal(it, tc.expected, tc.clusterSubTree)
		})
	}
}
