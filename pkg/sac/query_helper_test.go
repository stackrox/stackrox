package sac

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
)

const (
	planetArrakis = "planet.arrakis"
	planetEarth   = "planet.earth"

	nsSkunkWorks = "Skunk Works"
	nsFraunhofer = "Fraunhofer"
	nsCERN       = "CERN"
	nsJPL        = "JPL"

	nsAtreides     = "Atreides"
	nsHarkonnen    = "Harkonnen"
	nsSpacingGuild = "Spacing Guild"
)

type testCase struct {
	description    string
	scopeGenerator func(*testing.T) *effectiveaccessscope.ScopeTree
	expected       *v1.Query
	hasError       bool
}

func TestClusterScopeFilterGeneration(topLevelTest *testing.T) {
	testCases := []testCase{
		{
			description:    "nil ScopeTree generates a MatchNone query filter",
			scopeGenerator: effectiveaccessscope.TestTreeNil,
			expected:       getMatchNoneQuery(),
		},
		{
			description:    "DenyAllAccessScope generates a MatchNone query filter",
			scopeGenerator: effectiveaccessscope.TestTreeDenyAllEffectiveAccessScope,
			expected:       getMatchNoneQuery(),
		},
		{
			description:    "UnrestrictedEffectiveAccessScope generates an empty (nil) query filter",
			scopeGenerator: effectiveaccessscope.TestTreeUnrestrictedEffectiveAccessScope,
			expected:       nil,
		},
		{
			description:    "All excluded scope tree generates a MatchNone query filter",
			scopeGenerator: effectiveaccessscope.TestTreeAllExcluded,
			expected:       getMatchNoneQuery(),
		},
		{
			description:    "Invalid scope tree with excluded root generates a MatchNone query filter",
			scopeGenerator: effectiveaccessscope.TestTreeInvalidExcludedRootPartialBranch,
			expected:       getMatchNoneQuery(),
		},
		{
			description:    "Invalid scope tree with partial root and no cluster nodes generates a MatchNone query filter",
			scopeGenerator: effectiveaccessscope.TestTreeInvalidPartialRootWithoutChildren,
			expected:       getMatchNoneQuery(),
		},
		{
			description:    "Scope tree with one fully included cluster tree generates a Match query filter",
			scopeGenerator: effectiveaccessscope.TestTreeOneClusterTreeFullyIncluded,
			expected:       clusterVerboseMatch(topLevelTest, planetArrakis),
		},
		{
			description:    "Scope tree with one fully included cluster node generates a Match query filter",
			scopeGenerator: effectiveaccessscope.TestTreeOneClusterRootFullyIncluded,
			expected:       clusterVerboseMatch(topLevelTest, planetArrakis),
		},
		{
			description:    "Scope tree with only partial cluster nodes generates a Conjunction of Match query filter",
			scopeGenerator: effectiveaccessscope.TestTreeClusterNamespaceMixIncluded,
			expected: search.DisjunctionQuery(
				clusterVerboseMatch(topLevelTest, planetArrakis),
				clusterVerboseMatch(topLevelTest, planetEarth),
			),
		},
		{
			description:    "Scope tree with one included cluster tree and partial clusters generate a Disjunction of Match for the included clusters",
			scopeGenerator: effectiveaccessscope.TestTreeClusterNamespaceFullClusterMixIncluded,
			expected: search.DisjunctionQuery(
				clusterVerboseMatch(topLevelTest, planetArrakis),
				clusterVerboseMatch(topLevelTest, planetEarth),
			),
		},
		{
			description:    "Scope tree with multiple included cluster trees generates a Disjunction of Match for the included clusters",
			scopeGenerator: effectiveaccessscope.TestTreeTwoClustersFullyIncluded,
			expected: search.DisjunctionQuery(
				clusterVerboseMatch(topLevelTest, planetArrakis),
				clusterVerboseMatch(topLevelTest, planetEarth),
			),
		},
	}

	for _, tc := range testCases {
		topLevelTest.Run(tc.description, func(t *testing.T) {
			eas := tc.scopeGenerator(t)
			filter, err := BuildClusterLevelSACQueryFilter(eas)
			assert.True(t, tc.hasError == (err != nil))
			correctFilter := isSameQuery(tc.expected, filter)
			assert.Truef(t, correctFilter, "mismatch between queries")
			if !correctFilter {
				// Expose the mismatch in the test output
				assert.Equal(t, tc.expected, filter)
			}
		})
	}
}

func TestNamespaceScopeFilterGeneration(topLevelTest *testing.T) {
	testCases := []testCase{
		{
			description:    "Generated query filter for nil scope tree is MatchNone",
			scopeGenerator: effectiveaccessscope.TestTreeNil,
			expected:       getMatchNoneQuery(),
		},
		{
			description:    "Generated query filter for DenyAllEffectiveAccessScope is MatchNone",
			scopeGenerator: effectiveaccessscope.TestTreeDenyAllEffectiveAccessScope,
			expected:       getMatchNoneQuery(),
		},
		{
			description:    "Generated query filter for UnrestrictedEffectiveAccessScope is nil",
			scopeGenerator: effectiveaccessscope.TestTreeUnrestrictedEffectiveAccessScope,
			expected:       nil,
		},
		{
			description:    "Generated query filter for all excluded subtree is MatchNone",
			scopeGenerator: effectiveaccessscope.TestTreeAllExcluded,
			expected:       getMatchNoneQuery(),
		},
		{
			description:    "Generated query filter for invalid tree with excluded root is MatchNone",
			scopeGenerator: effectiveaccessscope.TestTreeInvalidExcludedRootPartialBranch,
			expected:       getMatchNoneQuery(),
		},
		{
			description:    "Generated query filter for invalid tree with partial root but no cluster children is MatchNone",
			scopeGenerator: effectiveaccessscope.TestTreeInvalidPartialRootWithoutChildren,
			expected:       getMatchNoneQuery(),
		},
		{
			description:    "Generated query filter for fully included cluster subtree is simple cluster match",
			scopeGenerator: effectiveaccessscope.TestTreeOneClusterTreeFullyIncluded,
			expected: search.ConjunctionQuery(
				clusterVerboseMatch(topLevelTest, planetArrakis),
				getAnyNamespaceMatchQuery(),
			),
		},
		{
			description:    "Generated query filter for included cluster node is simple cluster match",
			scopeGenerator: effectiveaccessscope.TestTreeOneClusterRootFullyIncluded,
			expected: search.ConjunctionQuery(
				clusterVerboseMatch(topLevelTest, planetArrakis),
				getAnyNamespaceMatchQuery(),
			),
		},
		{
			description:    "Generated query filter for single included namespace is the conjunction of the cluster and namespace matches",
			scopeGenerator: effectiveaccessscope.TestTreeOneClusterNamespacePairOnlyIncluded,
			expected: search.ConjunctionQuery(
				clusterVerboseMatch(topLevelTest, planetArrakis),
				namespaceVerboseMatch(topLevelTest, nsAtreides),
			),
		},
		{
			description:    "Generated query filter for two namespaces in the same cluster is the conjunction of the cluster and the disjunction of the namespaces",
			scopeGenerator: effectiveaccessscope.TestTreeOneClusterTwoNamespacesIncluded,
			expected: search.ConjunctionQuery(
				clusterVerboseMatch(topLevelTest, planetArrakis),
				search.DisjunctionQuery(
					namespaceVerboseMatch(topLevelTest, nsAtreides),
					namespaceVerboseMatch(topLevelTest, nsHarkonnen),
				),
			),
		},
		{
			description:    "Generated query filter for multiple namespaces in the same cluster is the conjunction of the cluster and the disjunction of the namespaces",
			scopeGenerator: effectiveaccessscope.TestTreeOneClusterMultipleNamespacesIncluded,
			expected: search.ConjunctionQuery(
				clusterVerboseMatch(topLevelTest, planetEarth),
				search.DisjunctionQuery(
					namespaceVerboseMatch(topLevelTest, nsSkunkWorks),
					namespaceVerboseMatch(topLevelTest, nsFraunhofer),
					namespaceVerboseMatch(topLevelTest, nsCERN),
				),
			),
		},
		{
			description:    "Genreated query filter for multiple cluster-namespace pairs is the disjunction of each cluster-namespace query",
			scopeGenerator: effectiveaccessscope.TestTreeTwoClusterNamespacePairsIncluded,
			expected: search.DisjunctionQuery(
				search.ConjunctionQuery(
					clusterVerboseMatch(topLevelTest, planetEarth),
					namespaceVerboseMatch(topLevelTest, nsSkunkWorks),
				),
				search.ConjunctionQuery(
					clusterVerboseMatch(topLevelTest, planetArrakis),
					namespaceVerboseMatch(topLevelTest, nsSpacingGuild),
				),
			),
		},
		{
			description:    "Generated query filter for a mix of cluster-namespace combination is the disjunction of the cluster queries",
			scopeGenerator: effectiveaccessscope.TestTreeClusterNamespaceMixIncluded,
			expected: search.DisjunctionQuery(
				search.ConjunctionQuery(
					clusterVerboseMatch(topLevelTest, planetArrakis),
					namespaceVerboseMatch(topLevelTest, nsSpacingGuild),
				),
				search.ConjunctionQuery(
					clusterVerboseMatch(topLevelTest, planetEarth),
					search.DisjunctionQuery(
						namespaceVerboseMatch(topLevelTest, nsSkunkWorks),
						namespaceVerboseMatch(topLevelTest, nsJPL),
					),
				),
			),
		},
		{
			description:    "Generated query filter for a full and a partial cluster is the disjunction of the full cluster match and the partial cluster tree",
			scopeGenerator: effectiveaccessscope.TestTreeClusterNamespaceFullClusterMixIncluded,
			expected: search.DisjunctionQuery(
				search.ConjunctionQuery(
					clusterVerboseMatch(topLevelTest, planetArrakis),
					getAnyNamespaceMatchQuery(),
				),
				search.ConjunctionQuery(
					clusterVerboseMatch(topLevelTest, planetEarth),
					search.DisjunctionQuery(
						namespaceVerboseMatch(topLevelTest, nsSkunkWorks),
						namespaceVerboseMatch(topLevelTest, nsFraunhofer),
						namespaceVerboseMatch(topLevelTest, nsCERN),
					),
				),
			),
		},
		{
			description:    "Generated query filter for a minimal scope tree matches exactly the tree structure",
			scopeGenerator: effectiveaccessscope.TestTreeMinimalPartialTree,
			expected: search.ConjunctionQuery(
				clusterVerboseMatch(topLevelTest, planetArrakis),
				search.DisjunctionQuery(
					namespaceVerboseMatch(topLevelTest, nsAtreides),
					namespaceVerboseMatch(topLevelTest, nsHarkonnen),
				),
			),
		},
		{
			description:    "Generated query filter for two fully included cluster tree is the disjunction of the cluster ID matches",
			scopeGenerator: effectiveaccessscope.TestTreeTwoClustersFullyIncluded,
			expected: search.DisjunctionQuery(
				search.ConjunctionQuery(
					clusterVerboseMatch(topLevelTest, planetArrakis),
					getAnyNamespaceMatchQuery(),
				),
				search.ConjunctionQuery(
					clusterVerboseMatch(topLevelTest, planetEarth),
					getAnyNamespaceMatchQuery(),
				),
			),
		},
	}

	for _, tc := range testCases {
		topLevelTest.Run(tc.description, func(t *testing.T) {
			eas := tc.scopeGenerator(t)
			filter, err := BuildClusterNamespaceLevelSACQueryFilter(eas)
			assert.True(t, tc.hasError == (err != nil))
			correctFilter := isSameQuery(tc.expected, filter)
			assert.Truef(t, correctFilter, "mismatch between queries")
			if !correctFilter {
				// Expose the mismatch in the test output
				assert.Equal(t, tc.expected, filter)
			}
		})
	}
}

func TestNonVerboseClusterScopeFilterGeneration(topLevelTest *testing.T) {
	testCases := []testCase{
		{
			description:    "nil ScopeTree generates a MatchNone query filter",
			scopeGenerator: effectiveaccessscope.TestTreeNil,
			expected:       getMatchNoneQuery(),
		},
		{
			description:    "DenyAllAccessScope generates a MatchNone query filter",
			scopeGenerator: effectiveaccessscope.TestTreeDenyAllEffectiveAccessScope,
			expected:       getMatchNoneQuery(),
		},
		{
			description:    "UnrestrictedEffectiveAccessScope generates an empty (nil) query filter",
			scopeGenerator: effectiveaccessscope.TestTreeUnrestrictedEffectiveAccessScope,
			expected:       nil,
		},
		{
			description:    "All excluded scope tree generates a MatchNone query filter",
			scopeGenerator: effectiveaccessscope.TestTreeAllExcluded,
			expected:       getMatchNoneQuery(),
		},
		{
			description:    "Invalid scope tree with excluded root generates a MatchNone query filter",
			scopeGenerator: effectiveaccessscope.TestTreeInvalidExcludedRootPartialBranch,
			expected:       getMatchNoneQuery(),
		},
		{
			description:    "Invalid scope tree with partial root and no cluster nodes generates a MatchNone query filter",
			scopeGenerator: effectiveaccessscope.TestTreeInvalidPartialRootWithoutChildren,
			expected:       getMatchNoneQuery(),
		},
		{
			description:    "Scope tree with one fully included cluster tree generates a Match query filter",
			scopeGenerator: effectiveaccessscope.TestTreeOneClusterTreeFullyIncluded,
			expected:       clusterNonVerboseMatch(topLevelTest, planetArrakis),
		},
		{
			description:    "Scope tree with one fully included cluster node generates a Match query filter",
			scopeGenerator: effectiveaccessscope.TestTreeOneClusterRootFullyIncluded,
			expected:       clusterNonVerboseMatch(topLevelTest, planetArrakis),
		},
		{
			description:    "Scope tree with only partial cluster nodes generates a MatchNone query filter",
			scopeGenerator: effectiveaccessscope.TestTreeClusterNamespaceMixIncluded,
			expected:       getMatchNoneQuery(),
		},
		{
			description:    "Scope tree with one included cluster tree and partial clusters generate only a Match for the included cluster",
			scopeGenerator: effectiveaccessscope.TestTreeClusterNamespaceFullClusterMixIncluded,
			expected:       clusterNonVerboseMatch(topLevelTest, planetArrakis),
		},
		{
			description:    "Scope tree with multiple included cluster trees generates a Disjunction of Match for the included clusters",
			scopeGenerator: effectiveaccessscope.TestTreeTwoClustersFullyIncluded,
			expected: search.DisjunctionQuery(
				clusterNonVerboseMatch(topLevelTest, planetArrakis),
				clusterNonVerboseMatch(topLevelTest, planetEarth)),
		},
	}

	for _, tc := range testCases {
		topLevelTest.Run(tc.description, func(t *testing.T) {
			eas := tc.scopeGenerator(t)
			filter, err := BuildNonVerboseClusterLevelSACQueryFilter(eas)
			assert.True(t, tc.hasError == (err != nil))
			correctFilter := isSameQuery(tc.expected, filter)
			assert.Truef(t, correctFilter, "mismatch between queries")
			if !correctFilter {
				// Expose the mismatch in the test output
				assert.Equal(t, tc.expected, filter)
			}
		})
	}
}

func TestNonVerboseNamespaceScopeFilterGeneration(topLevelTest *testing.T) {
	testCases := []testCase{
		{
			description:    "Generated query filter for nil scope tree is MatchNone",
			scopeGenerator: effectiveaccessscope.TestTreeNil,
			expected:       getMatchNoneQuery(),
		},
		{
			description:    "Generated query filter for DenyAllEffectiveAccessScope is MatchNone",
			scopeGenerator: effectiveaccessscope.TestTreeDenyAllEffectiveAccessScope,
			expected:       getMatchNoneQuery(),
		},
		{
			description:    "Generated query filter for UnrestrictedEffectiveAccessScope is nil",
			scopeGenerator: effectiveaccessscope.TestTreeUnrestrictedEffectiveAccessScope,
			expected:       nil,
		},
		{
			description:    "Generated query filter for all excluded subtree is MatchNone",
			scopeGenerator: effectiveaccessscope.TestTreeAllExcluded,
			expected:       getMatchNoneQuery(),
		},
		{
			description:    "Generated query filter for invalid tree with excluded root is MatchNone",
			scopeGenerator: effectiveaccessscope.TestTreeInvalidExcludedRootPartialBranch,
			expected:       getMatchNoneQuery(),
		},
		{
			description:    "Generated query filter for invalid tree with partial root but no cluster children is MatchNone",
			scopeGenerator: effectiveaccessscope.TestTreeInvalidPartialRootWithoutChildren,
			expected:       getMatchNoneQuery(),
		},
		{
			description:    "Generated query filter for fully included cluster subtree is simple cluster match",
			scopeGenerator: effectiveaccessscope.TestTreeOneClusterTreeFullyIncluded,
			expected:       clusterNonVerboseMatch(topLevelTest, planetArrakis),
		},
		{
			description:    "Generated query filter for included cluster node is simple cluster match",
			scopeGenerator: effectiveaccessscope.TestTreeOneClusterRootFullyIncluded,
			expected:       clusterNonVerboseMatch(topLevelTest, planetArrakis),
		},
		{
			description:    "Generated query filter for single included namespace is the conjunction of the cluster and namespace matches",
			scopeGenerator: effectiveaccessscope.TestTreeOneClusterNamespacePairOnlyIncluded,
			expected: search.ConjunctionQuery(
				clusterNonVerboseMatch(topLevelTest, planetArrakis),
				namespaceNonVerboseMatch(topLevelTest, nsAtreides),
			),
		},
		{
			description:    "Generated query filter for two namespaces in the same cluster is the conjunction of the cluster and the disjunction of the namespaces",
			scopeGenerator: effectiveaccessscope.TestTreeOneClusterTwoNamespacesIncluded,
			expected: search.ConjunctionQuery(
				clusterNonVerboseMatch(topLevelTest, planetArrakis),
				search.DisjunctionQuery(
					namespaceNonVerboseMatch(topLevelTest, nsAtreides),
					namespaceNonVerboseMatch(topLevelTest, nsHarkonnen),
				),
			),
		},
		{
			description:    "Generated query filter for multiple namespaces in the same cluster is the conjunction of the cluster and the disjunction of the namespaces",
			scopeGenerator: effectiveaccessscope.TestTreeOneClusterMultipleNamespacesIncluded,
			expected: search.ConjunctionQuery(
				clusterNonVerboseMatch(topLevelTest, planetEarth),
				search.DisjunctionQuery(
					namespaceNonVerboseMatch(topLevelTest, nsSkunkWorks),
					namespaceNonVerboseMatch(topLevelTest, nsFraunhofer),
					namespaceNonVerboseMatch(topLevelTest, nsCERN),
				),
			),
		},
		{
			description:    "Genreated query filter for multiple cluster-namespace pairs is the disjunction of each cluster-namespace query",
			scopeGenerator: effectiveaccessscope.TestTreeTwoClusterNamespacePairsIncluded,
			expected: search.DisjunctionQuery(
				search.ConjunctionQuery(
					clusterNonVerboseMatch(topLevelTest, planetEarth),
					namespaceNonVerboseMatch(topLevelTest, nsSkunkWorks),
				),
				search.ConjunctionQuery(
					clusterNonVerboseMatch(topLevelTest, planetArrakis),
					namespaceNonVerboseMatch(topLevelTest, nsSpacingGuild),
				),
			),
		},
		{
			description:    "Generated query filter for a mix of cluster-namespace combination is the disjunction of the cluster queries",
			scopeGenerator: effectiveaccessscope.TestTreeClusterNamespaceMixIncluded,
			expected: search.DisjunctionQuery(
				search.ConjunctionQuery(
					clusterNonVerboseMatch(topLevelTest, planetArrakis),
					namespaceNonVerboseMatch(topLevelTest, nsSpacingGuild),
				),
				search.ConjunctionQuery(
					clusterNonVerboseMatch(topLevelTest, planetEarth),
					search.DisjunctionQuery(
						namespaceNonVerboseMatch(topLevelTest, nsSkunkWorks),
						namespaceNonVerboseMatch(topLevelTest, nsJPL)),
				),
			),
		},
		{
			description:    "Generated query filter for a full and a partial cluster is the disjunction of the full cluster match and the partial cluster tree",
			scopeGenerator: effectiveaccessscope.TestTreeClusterNamespaceFullClusterMixIncluded,
			expected: search.DisjunctionQuery(
				clusterNonVerboseMatch(topLevelTest, planetArrakis),
				search.ConjunctionQuery(
					clusterNonVerboseMatch(topLevelTest, planetEarth),
					search.DisjunctionQuery(
						namespaceNonVerboseMatch(topLevelTest, nsSkunkWorks),
						namespaceNonVerboseMatch(topLevelTest, nsFraunhofer),
						namespaceNonVerboseMatch(topLevelTest, nsCERN),
					),
				),
			),
		},
		{
			description:    "Generated query filter for a minimal scope tree matches exactly the tree structure",
			scopeGenerator: effectiveaccessscope.TestTreeMinimalPartialTree,
			expected: search.ConjunctionQuery(
				clusterNonVerboseMatch(topLevelTest, planetArrakis),
				search.DisjunctionQuery(
					namespaceNonVerboseMatch(topLevelTest, nsAtreides),
					namespaceNonVerboseMatch(topLevelTest, nsHarkonnen),
				),
			),
		},
		{
			description:    "Generated query filter for two fully included cluster tree is the disjunction of the cluster ID matches",
			scopeGenerator: effectiveaccessscope.TestTreeTwoClustersFullyIncluded,
			expected: search.DisjunctionQuery(
				clusterNonVerboseMatch(topLevelTest, planetArrakis),
				clusterNonVerboseMatch(topLevelTest, planetEarth),
			),
		},
	}

	for _, tc := range testCases {
		topLevelTest.Run(tc.description, func(t *testing.T) {
			eas := tc.scopeGenerator(t)
			filter, err := BuildNonVerboseClusterNamespaceLevelSACQueryFilter(eas)
			assert.True(t, tc.hasError == (err != nil))
			correctFilter := isSameQuery(tc.expected, filter)
			assert.Truef(t, correctFilter, "mismatch between queries")
			if !correctFilter {
				// Expose the mismatch in the test output
				assert.Equal(t, tc.expected, filter)
			}
		})
	}
}
