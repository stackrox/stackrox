package effectiveaccessscope

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	labelUtils "github.com/stackrox/rox/pkg/labels"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

////////////////////////////////////////////////////////////////////////////////
// Cluster and namespace configuration                                        //
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

var clusters = []*storage.Cluster{
	{
		Id:   "planet.earth",
		Name: "Earth",
	},
	{
		Id:   "planet.arrakis",
		Name: "Arrakis",
		Labels: map[string]string{
			"focus": "melange",
		},
	},
}

var namespaces = []*storage.NamespaceMetadata{
	errored,
	// Earth
	skunkWorks,
	fraunhofer,
	cern,
	jpl,
	// Arrakis
	atreides,
	harkonnen,
	spacingGuild,
	bene,
	fremen,
}
var (
	skunkWorks = &storage.NamespaceMetadata{
		Id:          "lab.skunkworks",
		Name:        "Skunk Works",
		ClusterId:   "planet.earth",
		ClusterName: "Earth",
		Labels: map[string]string{
			"focus":     "transportation",
			"region":    "NA",
			"clearance": "yes",
		},
	}
	fraunhofer = &storage.NamespaceMetadata{
		Id:          "lab.fraunhofer",
		Name:        "Fraunhofer",
		ClusterId:   "planet.earth",
		ClusterName: "Earth",
		Labels: map[string]string{
			"focus":     "applied_research",
			"region":    "EU",
			"clearance": "no",
			"founded":   "1949",
		},
	}
	cern = &storage.NamespaceMetadata{
		Id:          "lab.cern",
		Name:        "CERN",
		ClusterId:   "planet.earth",
		ClusterName: "Earth",
		Labels: map[string]string{
			"focus":  "physics",
			"region": "EU",
		},
	}
	jpl = &storage.NamespaceMetadata{
		Id:          "lab.jpl",
		Name:        "JPL",
		ClusterId:   "planet.earth",
		ClusterName: "Earth",
		Labels: map[string]string{
			"focus":  "applied_research",
			"region": "NA",
		},
	}

	atreides = &storage.NamespaceMetadata{
		Id:          "house.atreides",
		Name:        "Atreides",
		ClusterId:   "planet.arrakis",
		ClusterName: "Arrakis",
		Labels: map[string]string{
			"focus":     "melange",
			"homeworld": "Caladan",
		},
	}
	harkonnen = &storage.NamespaceMetadata{
		Id:          "house.harkonnen",
		Name:        "Harkonnen",
		ClusterId:   "planet.arrakis",
		ClusterName: "Arrakis",
		Labels: map[string]string{
			"focus": "melange",
		},
	}
	spacingGuild = &storage.NamespaceMetadata{
		Id:          "org.spacingguild",
		Name:        "Spacing Guild",
		ClusterId:   "planet.arrakis",
		ClusterName: "Arrakis",
		Labels: map[string]string{
			"focus":     "transportation",
			"region":    "dune_universe",
			"depend-on": "melange",
		},
	}
	bene = &storage.NamespaceMetadata{
		Id:          "org.benegesserit",
		Name:        "Bene Gesserit",
		ClusterId:   "planet.arrakis",
		ClusterName: "Arrakis",
		Labels: map[string]string{
			"region": "dune_universe",
			"alias":  "witches",
		},
	}
	fremen = &storage.NamespaceMetadata{
		Id:          "tribe.fremen",
		Name:        "Fremen",
		ClusterId:   "planet.arrakis",
		ClusterName: "Arrakis",
	}
	errored = &storage.NamespaceMetadata{
		Id:          "not.found",
		Name:        "Not Found",
		ClusterId:   "not.found",
		ClusterName: "Not Found",
		Labels: map[string]string{
			"code": "404",
		},
	}
)

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

	notFoundCluster := &clustersScopeSubTree{
		State:      Excluded,
		Namespaces: namespacesTree(excluded(errored)),
		Attributes: treeNodeAttributes{
			Name: "Not Found",
		},
	}
	arrakisAttributes := treeNodeAttributes{ID: "planet.arrakis", Name: "Arrakis", Labels: map[string]string{"focus": "melange"}}
	earthAttributes := treeNodeAttributes{ID: "planet.earth", Name: "Earth"}
	clusterIDs := map[string]string{}
	for _, c := range clusters {
		clusterIDs[c.GetId()] = c.GetName()
	}
	testCases := []testCase{
		{
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
							excluded(skunkWorks),
							excluded(fraunhofer),
							excluded(cern),
							excluded(jpl),
						),
						Attributes: earthAttributes,
					},
					"Arrakis": {
						State: Excluded,
						Namespaces: namespacesTree(
							excluded(atreides),
							excluded(harkonnen),
							excluded(spacingGuild),
							excluded(bene),
							excluded(fremen),
						),
						Attributes: arrakisAttributes,
					},
					"Not Found": notFoundCluster,
				},
			},
			detail:   v1.ComputeEffectiveAccessScopeRequest_HIGH,
			hasError: false,
		},
		{
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
							excluded(skunkWorks),
							excluded(fraunhofer),
							excluded(cern),
							excluded(jpl),
						),
						Attributes: earthAttributes,
					},
					"Arrakis": {
						State: Excluded,
						Namespaces: namespacesTree(
							excluded(atreides),
							excluded(harkonnen),
							excluded(spacingGuild),
							excluded(bene),
							excluded(fremen),
						),
						Attributes: arrakisAttributes,
					},
					"Not Found": notFoundCluster,
				},
			},
			detail:   v1.ComputeEffectiveAccessScopeRequest_HIGH,
			hasError: false,
		},
		{
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
							excluded(skunkWorks),
							excluded(fraunhofer),
							excluded(cern),
							excluded(jpl),
						),
						Attributes: earthAttributes,
					},
					"Arrakis": {
						State: Excluded,
						Namespaces: namespacesTree(
							excluded(atreides),
							excluded(harkonnen),
							excluded(spacingGuild),
							excluded(bene),
							excluded(fremen),
						),
						Attributes: arrakisAttributes,
					},
					"Not Found": notFoundCluster,
				},
			},
			detail:   v1.ComputeEffectiveAccessScopeRequest_HIGH,
			hasError: false,
		},
		{
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
							excluded(skunkWorks),
							excluded(fraunhofer),
							excluded(cern),
							excluded(jpl),
						),
						Attributes: earthAttributes,
					},
					"Arrakis": {
						State: Included,
						Namespaces: namespacesTree(
							included(atreides),
							included(harkonnen),
							included(spacingGuild),
							included(bene),
							included(fremen),
						),
						Attributes: arrakisAttributes,
					},
					"Not Found": notFoundCluster,
				},
			},
			detail:   v1.ComputeEffectiveAccessScopeRequest_HIGH,
			hasError: false,
		},
		{
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
		{
			desc:      "cluster(s) included by label include all underlying namespaces",
			scopeDesc: `cluster.labels: focus in (melange) => { "Arrakis::*" }`,
			scopeStr:  "Arrakis::*",
			scopeJSON: `{"Arrakis":["*"]}`,
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
							excluded(skunkWorks),
							excluded(fraunhofer),
							excluded(cern),
							excluded(jpl),
						),
						Attributes: earthAttributes,
					},
					"Arrakis": {
						State: Included,
						Namespaces: namespacesTree(
							included(atreides),
							included(harkonnen),
							included(spacingGuild),
							included(bene),
							included(fremen),
						),
						Attributes: arrakisAttributes,
					},
					"Not Found": notFoundCluster,
				},
			},
			detail:   v1.ComputeEffectiveAccessScopeRequest_HIGH,
			hasError: false,
		},
		{
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
							excluded(skunkWorks),
							excluded(fraunhofer),
							excluded(cern),
							excluded(jpl),
						),
						Attributes: earthAttributes,
					},
					"Arrakis": {
						State: Partial,
						Namespaces: namespacesTree(
							included(atreides),
							excluded(harkonnen),
							excluded(spacingGuild),
							excluded(bene),
							excluded(fremen),
						),
						Attributes: arrakisAttributes,
					},
					"Not Found": notFoundCluster,
				},
			},
			detail:   v1.ComputeEffectiveAccessScopeRequest_HIGH,
			hasError: false,
		},
		{
			desc:      "namespace(s) included by label do not include anything else",
			scopeDesc: `namespace.labels: focus in (melange) => { "Arrakis::Atreides", "Arrakis::Harkonnen" }`,
			scopeStr:  "Arrakis::{Atreides, Harkonnen}",
			scopeJSON: `{"Arrakis":["Atreides","Harkonnen"]}`,
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
							excluded(skunkWorks),
							excluded(fraunhofer),
							excluded(cern),
							excluded(jpl),
						),
						Attributes: earthAttributes,
					},
					"Arrakis": {
						State: Partial,
						Namespaces: namespacesTree(
							included(atreides),
							included(harkonnen),
							excluded(spacingGuild),
							excluded(bene),
							excluded(fremen),
						),
						Attributes: arrakisAttributes,
					},
					"Not Found": notFoundCluster,
				},
			},
			detail:   v1.ComputeEffectiveAccessScopeRequest_HIGH,
			hasError: false,
		},
		{
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
							included(skunkWorks),
							excluded(fraunhofer),
							excluded(cern),
							excluded(jpl),
						),
						Attributes: earthAttributes,
					},
					"Arrakis": {
						State: Partial,
						Namespaces: namespacesTree(
							excluded(atreides),
							excluded(harkonnen),
							included(spacingGuild),
							excluded(bene),
							excluded(fremen),
						),
						Attributes: arrakisAttributes,
					},
					"Not Found": notFoundCluster,
				},
			},
			detail:   v1.ComputeEffectiveAccessScopeRequest_HIGH,
			hasError: false,
		},
		{
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
							included(skunkWorks),
							excluded(fraunhofer),
							excluded(cern),
							included(jpl),
						),
						Attributes: earthAttributes,
					},
					"Arrakis": {
						State: Partial,
						Namespaces: namespacesTree(
							excluded(atreides),
							excluded(harkonnen),
							included(spacingGuild),
							excluded(bene),
							excluded(fremen),
						),
						Attributes: arrakisAttributes,
					},
					"Not Found": notFoundCluster,
				},
			},
			detail:   v1.ComputeEffectiveAccessScopeRequest_HIGH,
			hasError: false,
		},
		{
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
							included(skunkWorks),
							excluded(fraunhofer),
							excluded(cern),
							excluded(jpl),
						),
						Attributes: earthAttributes,
					},
					"Arrakis": {
						State: Excluded,
						Namespaces: namespacesTree(
							excluded(atreides),
							excluded(harkonnen),
							excluded(spacingGuild),
							excluded(bene),
							excluded(fremen),
						),
						Attributes: arrakisAttributes,
					},
					"Not Found": notFoundCluster,
				},
			},
			detail:   v1.ComputeEffectiveAccessScopeRequest_HIGH,
			hasError: false,
		},
		{
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
							included(skunkWorks),
							included(fraunhofer),
							included(cern),
							excluded(jpl),
						),
						Attributes: earthAttributes,
					},
					"Arrakis": {
						State: Excluded,
						Namespaces: namespacesTree(
							excluded(atreides),
							excluded(harkonnen),
							excluded(spacingGuild),
							excluded(bene),
							excluded(fremen),
						),
						Attributes: arrakisAttributes,
					},
					"Not Found": notFoundCluster,
				},
			},
			detail:   v1.ComputeEffectiveAccessScopeRequest_HIGH,
			hasError: false,
		},
		{
			desc:      "rules are joined by OR",
			scopeDesc: `namespace: "Earth::Skunk Works" OR cluster.labels: focus in (melange) OR namespace.labels: region in (EU) => { "Earth::Skunk Works", "Earth::Fraunhofer", "Earth::CERN", "Arrakis::*" }`,
			scopeStr:  "Arrakis::*, Earth::{CERN, Fraunhofer, Skunk Works}",
			scopeJSON: `{"Earth":["CERN","Fraunhofer","Skunk Works"],"Arrakis":["*"]}`,
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
							included(skunkWorks),
							included(fraunhofer),
							included(cern),
							excluded(jpl),
						),
						Attributes: earthAttributes,
					},
					"Arrakis": {
						State: Included,
						Namespaces: namespacesTree(
							included(atreides),
							included(harkonnen),
							included(spacingGuild),
							included(bene),
							included(fremen),
						),
						Attributes: arrakisAttributes,
					},
					"Not Found": notFoundCluster,
				},
			},
			detail:   v1.ComputeEffectiveAccessScopeRequest_HIGH,
			hasError: false,
		},
		{
			desc:      "all excluded namespaces are removed from cluster in minimal form",
			scopeDesc: `"namespace.labels: focus in (melange)" => { "Arrakis::Atreides", "Arrakis::Harkonnen" }`,
			scopeStr:  "Arrakis::{Atreides, Harkonnen}",
			scopeJSON: `{"Arrakis":["Atreides","Harkonnen"]}`,
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
				},
			},
			detail:   v1.ComputeEffectiveAccessScopeRequest_MINIMAL,
			hasError: false,
		},
		{
			desc:      "no labels in standard form",
			scopeDesc: `"namespace.labels: focus in (melange)" => { "Arrakis::Atreides", "Arrakis::Harkonnen" }`,
			scopeStr:  "Arrakis::{Atreides, Harkonnen}",
			scopeJSON: `{"Arrakis":["Atreides","Harkonnen"]}`,
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
							excludedStandard(skunkWorks),
							excludedStandard(fraunhofer),
							excludedStandard(cern),
							excludedStandard(jpl),
						),
						Attributes: earthAttributes,
					},
					"Arrakis": {
						State: Partial,
						Namespaces: namespacesTree(
							includedStandard(atreides),
							includedStandard(harkonnen),
							excludedStandard(spacingGuild),
							excludedStandard(bene),
							excludedStandard(fremen),
						),
						Attributes: treeNodeAttributes{ID: "planet.arrakis", Name: "Arrakis"},
					},
					"Not Found": {
						State:      Excluded,
						Namespaces: namespacesTree(excludedStandard(errored)),
						Attributes: treeNodeAttributes{
							Name: "Not Found",
						},
					},
				},
			},
			detail:   v1.ComputeEffectiveAccessScopeRequest_STANDARD,
			hasError: false,
		},
		{
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
		{
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

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			var clonedClusters []*storage.Cluster
			for _, c := range clusters {
				clonedClusters = append(clonedClusters, c.Clone())
			}

			var clonedNamespaces []*storage.NamespaceMetadata
			for _, ns := range namespaces {
				clonedNamespaces = append(clonedNamespaces, ns.Clone())
			}

			result, err := ComputeEffectiveAccessScope(tc.scope.GetRules(), clusters, namespaces, tc.detail)
			assert.Truef(t, tc.hasError == (err != nil), "error: %v", err)
			assert.Equal(t, tc.expected, result, tc.scopeDesc)
			assert.Equal(t, clusters, clonedClusters, "clusters have been modified")
			assert.Equal(t, namespaces, clonedNamespaces, "namespaces have been modified")
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
	t.Parallel()
	testCases := []struct {
		a, b, c *ScopeTree
	}{
		{
			a: DenyAllEffectiveAccessScope(),
			b: UnrestrictedEffectiveAccessScope(),
			c: UnrestrictedEffectiveAccessScope(),
		},
		{
			a: UnrestrictedEffectiveAccessScope(),
			b: DenyAllEffectiveAccessScope(),
			c: UnrestrictedEffectiveAccessScope(),
		},
		{
			a: DenyAllEffectiveAccessScope(),
			b: DenyAllEffectiveAccessScope(),
			c: DenyAllEffectiveAccessScope(),
		},
		{
			a: &ScopeTree{
				State: Partial,
				Clusters: map[string]*clustersScopeSubTree{
					"Earth": {
						State: Excluded,
						Namespaces: namespacesTree(
							excludedStandard(skunkWorks),
							excludedStandard(fraunhofer),
							excludedStandard(cern),
							excludedStandard(jpl),
						),
					},
					"Arrakis": {
						State: Partial,
						Namespaces: namespacesTree(
							includedStandard(atreides),
							includedStandard(harkonnen),
							excludedStandard(spacingGuild),
							excludedStandard(bene),
							excludedStandard(fremen),
						),
						Attributes: treeNodeAttributes{ID: "planet.arrakis", Name: "Arrakis"},
					},
					"Not Found": {
						State:      Excluded,
						Namespaces: namespacesTree(excludedStandard(errored)),
						Attributes: treeNodeAttributes{
							Name: "Not Found",
						},
					},
				},
			},
			b: &ScopeTree{
				State: Partial,
				Clusters: map[string]*clustersScopeSubTree{
					"Earth": {
						State: Partial,
						Namespaces: namespacesTree(
							includedStandard(skunkWorks),
							includedStandard(fraunhofer),
							includedStandard(cern),
							excludedStandard(jpl),
						),
					},
					"Arrakis": {
						State: Partial,
						Namespaces: namespacesTree(
							excludedStandard(atreides),
							excludedStandard(harkonnen),
							includedStandard(spacingGuild),
							includedStandard(bene),
							includedStandard(fremen),
						),
						Attributes: treeNodeAttributes{ID: "planet.arrakis", Name: "Arrakis"},
					},
				},
			},
			c: &ScopeTree{
				State: Partial,
				Clusters: map[string]*clustersScopeSubTree{
					"Earth": {
						State: Partial,
						Namespaces: namespacesTree(
							includedStandard(skunkWorks),
							includedStandard(fraunhofer),
							includedStandard(cern),
						),
					},
					"Arrakis": {
						State: Partial,
						Namespaces: namespacesTree(
							includedStandard(atreides),
							includedStandard(harkonnen),
							includedStandard(spacingGuild),
							includedStandard(bene),
							includedStandard(fremen),
						),
					},
				},
			},
		},
		{
			a: &ScopeTree{
				State: Partial,
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
						Namespaces: namespacesTree(excludedStandard(errored)),
						Attributes: treeNodeAttributes{
							Name: "Not Found",
						},
					},
				},
			},
			b: &ScopeTree{
				State: Partial,
				Clusters: map[string]*clustersScopeSubTree{
					"Earth": {
						State: Partial,
						Namespaces: namespacesTree(
							includedStandard(skunkWorks),
							includedStandard(fraunhofer),
							includedStandard(cern),
							excludedStandard(jpl),
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
				Clusters: map[string]*clustersScopeSubTree{
					"Arrakis": {
						State: Partial,
					},
					"Earth": {
						State: Partial,
						Namespaces: namespacesTree(
							includedStandard(skunkWorks),
							includedStandard(fraunhofer),
							includedStandard(cern),
							excludedStandard(jpl),
						),
					},
				},
			},
		},
		{
			a: DenyAllEffectiveAccessScope(),
			b: &ScopeTree{
				State: Partial,
				Clusters: map[string]*clustersScopeSubTree{
					"Earth": {
						State: Included,
					},
				},
			},
			c: &ScopeTree{
				State: Partial,
				Clusters: map[string]*clustersScopeSubTree{
					"Earth": {
						State: Included,
					},
				},
			},
		},
		{
			a: &ScopeTree{
				State: Partial,
				Clusters: map[string]*clustersScopeSubTree{
					"Earth": {
						State: Excluded,
					},
				},
			},
			b: &ScopeTree{
				State: Partial,
				Clusters: map[string]*clustersScopeSubTree{
					"Earth": {
						State: Included,
					},
				},
			},
			c: &ScopeTree{
				State: Partial,
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
		t.Run("", func(t *testing.T) {
			t.Parallel()
			a.Merge(b)
			aJSON, err := a.ToJSON()
			assert.NoError(t, err)
			cJSON, err := c.ToJSON()
			assert.NoError(t, err)
			assert.JSONEq(t, cJSON, aJSON)
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
//   * label value contains a forbidden token (scope separator);
//   * label value length exceeds 63 characters.
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

func namespacesTree(namespaces ...*namespacesScopeSubTree) map[string]*namespacesScopeSubTree {
	m := map[string]*namespacesScopeSubTree{}
	for _, n := range namespaces {
		m[n.Attributes.Name] = n
	}
	return m
}

func included(n *storage.NamespaceMetadata) *namespacesScopeSubTree {
	return namespace(Included, n)
}

func includedStandard(n *storage.NamespaceMetadata) *namespacesScopeSubTree {
	return namespaceStandard(Included, n)
}

func excluded(n *storage.NamespaceMetadata) *namespacesScopeSubTree {
	return namespace(Excluded, n)
}

func excludedStandard(n *storage.NamespaceMetadata) *namespacesScopeSubTree {
	return namespaceStandard(Excluded, n)
}

func namespace(scope scopeState, n *storage.NamespaceMetadata) *namespacesScopeSubTree {
	return &namespacesScopeSubTree{State: scope, Attributes: treeNodeAttributes{
		ID:     n.Id,
		Name:   n.Name,
		Labels: n.Labels,
	}}
}

func namespaceStandard(scope scopeState, n *storage.NamespaceMetadata) *namespacesScopeSubTree {
	return &namespacesScopeSubTree{State: scope, Attributes: treeNodeAttributes{
		ID:   n.Id,
		Name: n.Name,
	}}
}
