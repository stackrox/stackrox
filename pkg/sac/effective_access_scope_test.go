package sac

import (
	"testing"

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
	// Earth
	{
		Id:          "lab.skunkworks",
		Name:        "Skunk Works",
		ClusterId:   "planet.earth",
		ClusterName: "Earth",
		Labels: map[string]string{
			"focus":     "transportation",
			"region":    "NA",
			"clearance": "yes",
		},
	},
	{
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
	},
	{
		Id:          "lab.cern",
		Name:        "CERN",
		ClusterId:   "planet.earth",
		ClusterName: "Earth",
		Labels: map[string]string{
			"focus":  "physics",
			"region": "EU",
		},
	},
	{
		Id:          "lab.jpl",
		Name:        "JPL",
		ClusterId:   "planet.earth",
		ClusterName: "Earth",
		Labels: map[string]string{
			"focus":  "applied_research",
			"region": "NA",
		},
	},
	// Arrakis
	{
		Id:          "house.atreides",
		Name:        "Atreides",
		ClusterId:   "planet.arrakis",
		ClusterName: "Arrakis",
		Labels: map[string]string{
			"focus":     "melange",
			"homeworld": "Caladan",
		},
	},
	{
		Id:          "house.harkonnen",
		Name:        "Harkonnen",
		ClusterId:   "planet.arrakis",
		ClusterName: "Arrakis",
		Labels: map[string]string{
			"focus": "melange",
		},
	},
	{
		Id:          "org.spacingguild",
		Name:        "Spacing Guild",
		ClusterId:   "planet.arrakis",
		ClusterName: "Arrakis",
		Labels: map[string]string{
			"focus":     "transportation",
			"region":    "dune_universe",
			"depend-on": "melange",
		},
	},
	{
		Id:          "org.benegesserit",
		Name:        "Bene Gesserit",
		ClusterId:   "planet.arrakis",
		ClusterName: "Arrakis",
		Labels: map[string]string{
			"region": "dune_universe",
			"alias":  "witches",
		},
	},
	{
		Id:          "tribe.fremen",
		Name:        "Fremen",
		ClusterId:   "planet.arrakis",
		ClusterName: "Arrakis",
	},
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

// TODO(ROX-7136): Add tests to cover error paths (matcher can't be constructed
//   because of violated constraints) and empty cluster / namespaces.
func TestComputeEffectiveAccessScope(t *testing.T) {
	type testCase struct {
		desc      string
		scopeDesc string
		scope     *storage.SimpleAccessScope
		expected  *EffectiveAccessScopeTree
		hasError  bool
	}

	goodTestCases := []testCase{
		{
			desc:      "no access scope includes nothing",
			scopeDesc: `nil => { }`,
			scope:     nil,
			expected: &EffectiveAccessScopeTree{
				Excluded,
				map[string]*ClustersScopeSubTree{
					"Earth": {
						State: Excluded,
						Namespaces: map[string]*NamespacesScopeSubTree{
							"Skunk Works": {State: Excluded},
							"Fraunhofer":  {State: Excluded},
							"CERN":        {State: Excluded},
							"JPL":         {State: Excluded},
						},
					},
					"Arrakis": {
						State: Excluded,
						Namespaces: map[string]*NamespacesScopeSubTree{
							"Atreides":      {State: Excluded},
							"Harkonnen":     {State: Excluded},
							"Spacing Guild": {State: Excluded},
							"Bene Gesserit": {State: Excluded},
							"Fremen":        {State: Excluded},
						},
					},
				},
			},
			hasError: false,
		},
		{
			desc:      "empty access scope includes nothing",
			scopeDesc: `∅ => { }`,
			scope: &storage.SimpleAccessScope{
				Id:   accessScopeID,
				Name: accessScopeName,
			},
			expected: &EffectiveAccessScopeTree{
				Excluded,
				map[string]*ClustersScopeSubTree{
					"Earth": {
						State: Excluded,
						Namespaces: map[string]*NamespacesScopeSubTree{
							"Skunk Works": {State: Excluded},
							"Fraunhofer":  {State: Excluded},
							"CERN":        {State: Excluded},
							"JPL":         {State: Excluded},
						},
					},
					"Arrakis": {
						State: Excluded,
						Namespaces: map[string]*NamespacesScopeSubTree{
							"Atreides":      {State: Excluded},
							"Harkonnen":     {State: Excluded},
							"Spacing Guild": {State: Excluded},
							"Bene Gesserit": {State: Excluded},
							"Fremen":        {State: Excluded},
						},
					},
				},
			},
			hasError: false,
		},
		{
			desc:      "cluster included by name includes all its namespaces",
			scopeDesc: `cluster: "Arrakis" => { "Arrakis::*" }`,
			scope: &storage.SimpleAccessScope{
				Id:   accessScopeID,
				Name: accessScopeName,
				Rules: &storage.SimpleAccessScope_Rules{
					IncludedClusters: []string{"Arrakis"},
				},
			},
			expected: &EffectiveAccessScopeTree{
				Partial,
				map[string]*ClustersScopeSubTree{
					"Earth": {
						State: Excluded,
						Namespaces: map[string]*NamespacesScopeSubTree{
							"Skunk Works": {State: Excluded},
							"Fraunhofer":  {State: Excluded},
							"CERN":        {State: Excluded},
							"JPL":         {State: Excluded},
						},
					},
					"Arrakis": {
						State: Included,
						Namespaces: map[string]*NamespacesScopeSubTree{
							"Atreides":      {State: Included},
							"Harkonnen":     {State: Included},
							"Spacing Guild": {State: Included},
							"Bene Gesserit": {State: Included},
							"Fremen":        {State: Included},
						},
					},
				},
			},
			hasError: false,
		},
		{
			desc:      "cluster(s) included by label include all underlying namespaces",
			scopeDesc: `cluster.labels: focus in (melange) => { "Arrakis::*" }`,
			scope: &storage.SimpleAccessScope{
				Id:   accessScopeID,
				Name: accessScopeName,
				Rules: &storage.SimpleAccessScope_Rules{
					ClusterLabelSelectors: labelUtils.LabelSelectors("focus", opIN, []string{"melange"}),
				},
			},
			expected: &EffectiveAccessScopeTree{
				Partial,
				map[string]*ClustersScopeSubTree{
					"Earth": {
						State: Excluded,
						Namespaces: map[string]*NamespacesScopeSubTree{
							"Skunk Works": {State: Excluded},
							"Fraunhofer":  {State: Excluded},
							"CERN":        {State: Excluded},
							"JPL":         {State: Excluded},
						},
					},
					"Arrakis": {
						State: Included,
						Namespaces: map[string]*NamespacesScopeSubTree{
							"Atreides":      {State: Included},
							"Harkonnen":     {State: Included},
							"Spacing Guild": {State: Included},
							"Bene Gesserit": {State: Included},
							"Fremen":        {State: Included},
						},
					},
				},
			},
			hasError: false,
		},
		{
			desc:      "namespace included by name does not include anything else",
			scopeDesc: `namespace: "Arrakis::Atreides" => { "Arrakis::Atreides" }`,
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
			expected: &EffectiveAccessScopeTree{
				Partial,
				map[string]*ClustersScopeSubTree{
					"Earth": {
						State: Excluded,
						Namespaces: map[string]*NamespacesScopeSubTree{
							"Skunk Works": {State: Excluded},
							"Fraunhofer":  {State: Excluded},
							"CERN":        {State: Excluded},
							"JPL":         {State: Excluded},
						},
					},
					"Arrakis": {
						State: Partial,
						Namespaces: map[string]*NamespacesScopeSubTree{
							"Atreides":      {State: Included},
							"Harkonnen":     {State: Excluded},
							"Spacing Guild": {State: Excluded},
							"Bene Gesserit": {State: Excluded},
							"Fremen":        {State: Excluded},
						},
					},
				},
			},
			hasError: false,
		},
		{
			desc:      "namespace(s) included by label do not include anything else",
			scopeDesc: `namespace.labels: focus in (melange) => { "Arrakis::Atreides", "Arrakis::Harkonnen" }`,
			scope: &storage.SimpleAccessScope{
				Id:   accessScopeID,
				Name: accessScopeName,
				Rules: &storage.SimpleAccessScope_Rules{
					NamespaceLabelSelectors: labelUtils.LabelSelectors("focus", opIN, []string{"melange"}),
				},
			},
			expected: &EffectiveAccessScopeTree{
				Partial,
				map[string]*ClustersScopeSubTree{
					"Earth": {
						State: Excluded,
						Namespaces: map[string]*NamespacesScopeSubTree{
							"Skunk Works": {State: Excluded},
							"Fraunhofer":  {State: Excluded},
							"CERN":        {State: Excluded},
							"JPL":         {State: Excluded},
						},
					},
					"Arrakis": {
						State: Partial,
						Namespaces: map[string]*NamespacesScopeSubTree{
							"Atreides":      {State: Included},
							"Harkonnen":     {State: Included},
							"Spacing Guild": {State: Excluded},
							"Bene Gesserit": {State: Excluded},
							"Fremen":        {State: Excluded},
						},
					},
				},
			},
			hasError: false,
		},
		{
			desc:      "inclusion by label works across clusters",
			scopeDesc: `namespace.labels: focus in (transportation) => { "Earth::Skunk Works", "Arrakis::Spacing Guild" }`,
			scope: &storage.SimpleAccessScope{
				Id:   accessScopeID,
				Name: accessScopeName,
				Rules: &storage.SimpleAccessScope_Rules{
					NamespaceLabelSelectors: labelUtils.LabelSelectors("focus", opIN, []string{"transportation"}),
				},
			},
			expected: &EffectiveAccessScopeTree{
				Partial,
				map[string]*ClustersScopeSubTree{
					"Earth": {
						State: Partial,
						Namespaces: map[string]*NamespacesScopeSubTree{
							"Skunk Works": {State: Included},
							"Fraunhofer":  {State: Excluded},
							"CERN":        {State: Excluded},
							"JPL":         {State: Excluded},
						},
					},
					"Arrakis": {
						State: Partial,
						Namespaces: map[string]*NamespacesScopeSubTree{
							"Atreides":      {State: Excluded},
							"Harkonnen":     {State: Excluded},
							"Spacing Guild": {State: Included},
							"Bene Gesserit": {State: Excluded},
							"Fremen":        {State: Excluded},
						},
					},
				},
			},
			hasError: false,
		},
		{
			desc:      "inclusion by label groups labels by AND and set values by OR",
			scopeDesc: `namespace.labels: focus in (transportation, applied_research), region in (NA, dune_universe) => { "Earth::Skunk Works", "Earth::JPL", "Arrakis::Spacing Guild" }`,
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
			expected: &EffectiveAccessScopeTree{
				Partial,
				map[string]*ClustersScopeSubTree{
					"Earth": {
						State: Partial,
						Namespaces: map[string]*NamespacesScopeSubTree{
							"Skunk Works": {State: Included},
							"Fraunhofer":  {State: Excluded},
							"CERN":        {State: Excluded},
							"JPL":         {State: Included},
						},
					},
					"Arrakis": {
						State: Partial,
						Namespaces: map[string]*NamespacesScopeSubTree{
							"Atreides":      {State: Excluded},
							"Harkonnen":     {State: Excluded},
							"Spacing Guild": {State: Included},
							"Bene Gesserit": {State: Excluded},
							"Fremen":        {State: Excluded},
						},
					},
				},
			},
			hasError: false,
		},
		{
			desc:      "inclusion by label supports EXISTS, NOT_EXISTS, and NOTIN operators",
			scopeDesc: `namespace.labels: focus notin (physics, melange), clearance, !founded => { "Earth::Skunk Works" }`,
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
			expected: &EffectiveAccessScopeTree{
				Partial,
				map[string]*ClustersScopeSubTree{
					"Earth": {
						State: Partial,
						Namespaces: map[string]*NamespacesScopeSubTree{
							"Skunk Works": {State: Included},
							"Fraunhofer":  {State: Excluded},
							"CERN":        {State: Excluded},
							"JPL":         {State: Excluded},
						},
					},
					"Arrakis": {
						State: Excluded,
						Namespaces: map[string]*NamespacesScopeSubTree{
							"Atreides":      {State: Excluded},
							"Harkonnen":     {State: Excluded},
							"Spacing Guild": {State: Excluded},
							"Bene Gesserit": {State: Excluded},
							"Fremen":        {State: Excluded},
						},
					},
				},
			},
			hasError: false,
		},
		{
			desc:      "multiple label selectors are joined by OR",
			scopeDesc: `namespace.labels: focus in (transportation), region in (NA) OR region in (EU) OR founded in (1949) => { "Earth::Skunk Works", "Earth::Fraunhofer", "Earth::CERN" }`,
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
			expected: &EffectiveAccessScopeTree{
				Partial,
				map[string]*ClustersScopeSubTree{
					"Earth": {
						State: Partial,
						Namespaces: map[string]*NamespacesScopeSubTree{
							"Skunk Works": {State: Included},
							"Fraunhofer":  {State: Included},
							"CERN":        {State: Included},
							"JPL":         {State: Excluded},
						},
					},
					"Arrakis": {
						State: Excluded,
						Namespaces: map[string]*NamespacesScopeSubTree{
							"Atreides":      {State: Excluded},
							"Harkonnen":     {State: Excluded},
							"Spacing Guild": {State: Excluded},
							"Bene Gesserit": {State: Excluded},
							"Fremen":        {State: Excluded},
						},
					},
				},
			},
			hasError: false,
		},
		{
			desc:      "rules are joined by OR",
			scopeDesc: `namespace: "Earth::Skunk Works" OR cluster.labels: focus in (melange) OR namespace.labels: region in (EU) => { "Earth::Skunk Works", "Earth::Fraunhofer", "Earth::CERN", "Arrakis::*" }`,
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
			expected: &EffectiveAccessScopeTree{
				Partial,
				map[string]*ClustersScopeSubTree{
					"Earth": {
						State: Partial,
						Namespaces: map[string]*NamespacesScopeSubTree{
							"Skunk Works": {State: Included},
							"Fraunhofer":  {State: Included},
							"CERN":        {State: Included},
							"JPL":         {State: Excluded},
						},
					},
					"Arrakis": {
						State: Included,
						Namespaces: map[string]*NamespacesScopeSubTree{
							"Atreides":      {State: Included},
							"Harkonnen":     {State: Included},
							"Spacing Guild": {State: Included},
							"Bene Gesserit": {State: Included},
							"Fremen":        {State: Included},
						},
					},
				},
			},
			hasError: false,
		},
	}

	for _, tc := range goodTestCases {
		t.Run(tc.desc, func(t *testing.T) {
			var clonedClusters []*storage.Cluster
			for _, c := range clusters {
				clonedClusters = append(clonedClusters, c.Clone())
			}

			var clonedNamespaces []*storage.NamespaceMetadata
			for _, ns := range namespaces {
				clonedNamespaces = append(clonedNamespaces, ns.Clone())
			}

			result, err := ComputeEffectiveAccessScope(tc.scope.GetRules(), clusters, namespaces)
			assert.Truef(t, tc.hasError == (err != nil), "error: %v", err)
			assert.Exactly(t, tc.expected, result, tc.scopeDesc)
			assert.Exactly(t, clusters, clonedClusters, "clusters have been modified")
			assert.Exactly(t, namespaces, clonedNamespaces, "namespaces have been modified")
		})
	}
}

func TestEffectiveAccessScopeAllowEverything(t *testing.T) {
	expected := &EffectiveAccessScopeTree{
		Included,
		map[string]*ClustersScopeSubTree{
			"Earth": {
				State: Included,
				Namespaces: map[string]*NamespacesScopeSubTree{
					"Skunk Works": {State: Included},
					"Fraunhofer":  {State: Included},
					"CERN":        {State: Included},
					"JPL":         {State: Included},
				},
			},
			"Arrakis": {
				State: Included,
				Namespaces: map[string]*NamespacesScopeSubTree{
					"Atreides":      {State: Included},
					"Harkonnen":     {State: Included},
					"Spacing Guild": {State: Included},
					"Bene Gesserit": {State: Included},
					"Fremen":        {State: Included},
				},
			},
		},
	}

	var clonedClusters []*storage.Cluster
	for _, c := range clusters {
		clonedClusters = append(clonedClusters, c.Clone())
	}

	var clonedNamespaces []*storage.NamespaceMetadata
	for _, ns := range namespaces {
		clonedNamespaces = append(clonedNamespaces, ns.Clone())
	}

	result := EffectiveAccessScopeAllowEverything(clusters, namespaces)
	assert.Exactly(t, expected, result)
	assert.Exactly(t, clusters, clonedClusters, "clusters have been modified")
	assert.Exactly(t, namespaces, clonedNamespaces, "namespaces have been modified")
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
		assert.Truef(t, selector.Matches(tc), "%q should match %q", selector.String(), tc.String())
	}

	testCasesBad := []labels.Set{
		{},
		labels.Set(map[string]string{"random.key": tooLongValue}),
	}
	for _, tc := range testCasesBad {
		assert.Falsef(t, selector.Matches(tc), "%q should not match %q", selector.String(), tc.String())
	}
}
