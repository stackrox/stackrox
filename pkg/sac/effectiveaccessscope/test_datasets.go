package effectiveaccessscope

import (
	"testing"
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
	clusterEarth = &clusterForSAC{
		ID:   "planet.earth",
		Name: "Earth",
	}

	clusterArrakis = &clusterForSAC{
		ID:   "planet.arrakis",
		Name: "Arrakis",
		Labels: map[string]string{
			"focus": "melange",
		},
	}
)

// Cluster helpers
var (
	clusterIDs = map[string]string{
		clusterEarth.GetID():   clusterEarth.GetName(),
		clusterArrakis.GetID(): clusterArrakis.GetName(),
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
	nsSkunkWorks = &namespaceForSAC{
		ID:          "lab.skunkworks",
		Name:        "Skunk Works",
		ClusterID:   "planet.earth",
		ClusterName: "Earth",
		Labels: map[string]string{
			"focus":     "transportation",
			"region":    "NA",
			"clearance": "yes",
		},
	}

	nsFraunhofer = &namespaceForSAC{
		ID:          "lab.fraunhofer",
		Name:        "Fraunhofer",
		ClusterID:   "planet.earth",
		ClusterName: "Earth",
		Labels: map[string]string{
			"focus":     "applied_research",
			"region":    "EU",
			"clearance": "no",
			"founded":   "1949",
		},
	}

	nsCERN = &namespaceForSAC{
		ID:          "lab.cern",
		Name:        "CERN",
		ClusterID:   "planet.earth",
		ClusterName: "Earth",
		Labels: map[string]string{
			"focus":  "physics",
			"region": "EU",
		},
	}

	nsJPL = &namespaceForSAC{
		ID:          "lab.jpl",
		Name:        "JPL",
		ClusterID:   "planet.earth",
		ClusterName: "Earth",
		Labels: map[string]string{
			"focus":  "applied_research",
			"region": "NA",
		},
	}

	nsAtreides = &namespaceForSAC{
		ID:          "house.atreides",
		Name:        "Atreides",
		ClusterID:   "planet.arrakis",
		ClusterName: "Arrakis",
		Labels: map[string]string{
			"focus":     "melange",
			"homeworld": "Caladan",
		},
	}

	nsHarkonnen = &namespaceForSAC{
		ID:          "house.harkonnen",
		Name:        "Harkonnen",
		ClusterID:   "planet.arrakis",
		ClusterName: "Arrakis",
		Labels: map[string]string{
			"focus": "melange",
		},
	}

	nsSpacingGuild = &namespaceForSAC{
		ID:          "org.spacingguild",
		Name:        "Spacing Guild",
		ClusterID:   "planet.arrakis",
		ClusterName: "Arrakis",
		Labels: map[string]string{
			"focus":     "transportation",
			"region":    "dune_universe",
			"depend-on": "melange",
		},
	}

	nsBeneGesserit = &namespaceForSAC{
		ID:          "org.benegesserit",
		Name:        "Bene Gesserit",
		ClusterID:   "planet.arrakis",
		ClusterName: "Arrakis",
		Labels: map[string]string{
			"region": "dune_universe",
			"alias":  "witches",
		},
	}

	nsFremen = &namespaceForSAC{
		ID:          "tribe.fremen",
		Name:        "Fremen",
		ClusterID:   "planet.arrakis",
		ClusterName: "Arrakis",
	}

	nsErrored = &namespaceForSAC{
		ID:          "not.found",
		Name:        "Not Found",
		ClusterID:   "not.found",
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

func included(n *namespaceForSAC) *namespacesScopeSubTree {
	return namespace(Included, n)
}

func includedStandard(n *namespaceForSAC) *namespacesScopeSubTree {
	return namespaceStandard(Included, n)
}

func excluded(n *namespaceForSAC) *namespacesScopeSubTree {
	return namespace(Excluded, n)
}

func excludedStandard(n *namespaceForSAC) *namespacesScopeSubTree {
	return namespaceStandard(Excluded, n)
}

func namespace(scope scopeState, n *namespaceForSAC) *namespacesScopeSubTree {
	return &namespacesScopeSubTree{State: scope, Attributes: treeNodeAttributes{
		ID:     n.ID,
		Name:   n.Name,
		Labels: n.Labels,
	}}
}

func namespaceStandard(scope scopeState, n *namespaceForSAC) *namespacesScopeSubTree {
	return &namespacesScopeSubTree{State: scope, Attributes: treeNodeAttributes{
		ID:   n.ID,
		Name: n.Name,
	}}
}

func cloneCluster(c ClusterForSAC) ClusterForSAC {
	clonedLabels := make(map[string]string, len(c.GetLabels()))
	for k, v := range c.GetLabels() {
		clonedLabels[k] = v
	}
	if c.GetLabels() == nil {
		clonedLabels = nil
	}
	return &clusterForSAC{
		ID:     c.GetID(),
		Name:   c.GetName(),
		Labels: clonedLabels,
	}
}

func cloneNamespace(ns NamespaceForSAC) NamespaceForSAC {
	clonedLabels := make(map[string]string, len(ns.GetLabels()))
	for k, v := range ns.GetLabels() {
		clonedLabels[k] = v
	}
	if ns.GetLabels() == nil {
		clonedLabels = nil
	}
	return &namespaceForSAC{
		ID:          ns.GetID(),
		Name:        ns.GetName(),
		ClusterID:   ns.GetClusterID(),
		ClusterName: ns.GetClusterName(),
		Labels:      clonedLabels,
	}
}
