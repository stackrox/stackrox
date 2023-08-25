package effectiveaccessscope

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
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

// storage.Cluster objects
var (
	clusterEarth = &storage.Cluster{
		Id:   "planet.earth",
		Name: "Earth",
	}

	clusterArrakis = &storage.Cluster{
		Id:   "planet.arrakis",
		Name: "Arrakis",
		Labels: map[string]string{
			"focus": "melange",
		},
	}
)

// Cluster helpers
var (
	clusterIDs = map[string]string{
		clusterEarth.GetId():   clusterEarth.GetName(),
		clusterArrakis.GetId(): clusterArrakis.GetName(),
	}

	arrakisAttributes = treeNodeAttributes{
		ID:     "planet.arrakis",
		Name:   "Arrakis",
		Labels: map[string]string{"focus": "melange"},
	}

	earthAttributes = treeNodeAttributes{
		ID:   "planet.earth",
		Name: "Earth",
	}

	notFoundCluster = &clustersScopeSubTree{
		State:      Excluded,
		Namespaces: namespacesTree(excluded(nsErrored)),
		Attributes: treeNodeAttributes{
			Name: "Not Found",
		},
	}
)

// storage.NamespaceMetadata objects
var (
	nsSkunkWorks = &storage.NamespaceMetadata{
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

	nsFraunhofer = &storage.NamespaceMetadata{
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

	nsCERN = &storage.NamespaceMetadata{
		Id:          "lab.cern",
		Name:        "CERN",
		ClusterId:   "planet.earth",
		ClusterName: "Earth",
		Labels: map[string]string{
			"focus":  "physics",
			"region": "EU",
		},
	}

	nsJPL = &storage.NamespaceMetadata{
		Id:          "lab.jpl",
		Name:        "JPL",
		ClusterId:   "planet.earth",
		ClusterName: "Earth",
		Labels: map[string]string{
			"focus":  "applied_research",
			"region": "NA",
		},
	}

	nsAtreides = &storage.NamespaceMetadata{
		Id:          "house.atreides",
		Name:        "Atreides",
		ClusterId:   "planet.arrakis",
		ClusterName: "Arrakis",
		Labels: map[string]string{
			"focus":     "melange",
			"homeworld": "Caladan",
		},
	}

	nsHarkonnen = &storage.NamespaceMetadata{
		Id:          "house.harkonnen",
		Name:        "Harkonnen",
		ClusterId:   "planet.arrakis",
		ClusterName: "Arrakis",
		Labels: map[string]string{
			"focus": "melange",
		},
	}

	nsSpacingGuild = &storage.NamespaceMetadata{
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

	nsBeneGesserit = &storage.NamespaceMetadata{
		Id:          "org.benegesserit",
		Name:        "Bene Gesserit",
		ClusterId:   "planet.arrakis",
		ClusterName: "Arrakis",
		Labels: map[string]string{
			"region": "dune_universe",
			"alias":  "witches",
		},
	}

	nsFremen = &storage.NamespaceMetadata{
		Id:          "tribe.fremen",
		Name:        "Fremen",
		ClusterId:   "planet.arrakis",
		ClusterName: "Arrakis",
	}

	nsErrored = &storage.NamespaceMetadata{
		Id:          "not.found",
		Name:        "Not Found",
		ClusterId:   "not.found",
		ClusterName: "Not Found",
		Labels: map[string]string{
			"code": "404",
		},
	}
)

// TestTreeNil provides a nil mock ScopeTree for testing purposes.
func TestTreeNil(_ *testing.T) *ScopeTree {
	return nil
}

// TestTreeDenyAllEffectiveAccessScope provides a getter on DenyAllAccessScope for testing purposes.
func TestTreeDenyAllEffectiveAccessScope(_ *testing.T) *ScopeTree {
	return DenyAllEffectiveAccessScope()
}

// TestTreeUnrestrictedEffectiveAccessScope provides a getter on UnrestrictedEffectiveAccessScope for testing purposes.
func TestTreeUnrestrictedEffectiveAccessScope(_ *testing.T) *ScopeTree {
	return UnrestrictedEffectiveAccessScope()
}

// TestTreeAllExcluded provides a mock ScopeTree with an excluded root for testing purposes.
func TestTreeAllExcluded(_ *testing.T) *ScopeTree {
	return &ScopeTree{
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
			"Not Found": notFoundCluster,
		},
	}
}

// TestTreeInvalidPartialRootWithoutChildren provides a mock ScopeTree for testing purposes.
func TestTreeInvalidPartialRootWithoutChildren(_ *testing.T) *ScopeTree {
	return &ScopeTree{
		State: Partial,
	}
}

// TestTreeInvalidExcludedRootPartialBranch provides a mock ScopeTree with an excluded root for testing purposes.
func TestTreeInvalidExcludedRootPartialBranch(_ *testing.T) *ScopeTree {
	return &ScopeTree{
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
			"Not Found": notFoundCluster,
		},
	}
}

// TestTreeOneClusterTreeFullyIncluded provides a mock ScopeTree for testing purposes.
func TestTreeOneClusterTreeFullyIncluded(_ *testing.T) *ScopeTree {
	return &ScopeTree{
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
			"Not Found": notFoundCluster,
		},
	}
}

// TestTreeOneClusterRootFullyIncluded provides a mock ScopeTree for testing purposes.
func TestTreeOneClusterRootFullyIncluded(_ *testing.T) *ScopeTree {
	return &ScopeTree{
		State:           Partial,
		clusterIDToName: clusterIDs,
		Clusters: map[string]*clustersScopeSubTree{
			clusterArrakis.GetName(): {
				State:      Included,
				Attributes: treeNodeAttributes{ID: "planet.arrakis"},
			},
		},
	}
}

// TestTreeOneClusterNamespacePairOnlyIncluded provides a mock ScopeTree for testing purposes.
func TestTreeOneClusterNamespacePairOnlyIncluded(_ *testing.T) *ScopeTree {
	return &ScopeTree{
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
			"Not Found": notFoundCluster,
		},
	}
}

// TestTreeOneClusterTwoNamespacesIncluded provides a mock ScopeTree for testing purposes.
func TestTreeOneClusterTwoNamespacesIncluded(_ *testing.T) *ScopeTree {
	return &ScopeTree{
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
			"Not Found": notFoundCluster,
		},
	}
}

// TestTreeOneClusterMultipleNamespacesIncluded provides a mock ScopeTree for testing purposes.
func TestTreeOneClusterMultipleNamespacesIncluded(_ *testing.T) *ScopeTree {
	return &ScopeTree{
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
			"Not Found": notFoundCluster,
		},
	}
}

// TestTreeTwoClusterNamespacePairsIncluded provides a mock ScopeTree for testing purposes.
func TestTreeTwoClusterNamespacePairsIncluded(_ *testing.T) *ScopeTree {
	return &ScopeTree{
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
			"Not Found": notFoundCluster,
		},
	}
}

// TestTreeClusterNamespaceMixIncluded provides a mock ScopeTree for testing purposes.
func TestTreeClusterNamespaceMixIncluded(_ *testing.T) *ScopeTree {
	return &ScopeTree{
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
			"Not Found": notFoundCluster,
		},
	}
}

// TestTreeClusterNamespaceFullClusterMixIncluded provides a mock ScopeTree for testing purposes.
func TestTreeClusterNamespaceFullClusterMixIncluded(_ *testing.T) *ScopeTree {
	return &ScopeTree{
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
			"Not Found": notFoundCluster,
		},
	}
}

// TestTreeMinimalPartialTree provides a mock ScopeTree for testing purposes
func TestTreeMinimalPartialTree(_ *testing.T) *ScopeTree {
	return &ScopeTree{
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
	}
}

// TestTreeTwoClustersFullyIncluded provides a mock ScopeTree for testing purposes
func TestTreeTwoClustersFullyIncluded(_ *testing.T) *ScopeTree {
	return &ScopeTree{
		State:           Partial,
		clusterIDToName: clusterIDs,
		Clusters: map[string]*clustersScopeSubTree{
			"Earth": {
				State: Included,
				Namespaces: namespacesTree(
					included(nsSkunkWorks),
					included(nsFraunhofer),
					included(nsCERN),
					included(nsJPL),
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
			"Not Found": notFoundCluster,
		},
	}
}

// Helper functions

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
