//go:build sql_integration

package service

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/labels"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stretchr/testify/assert"
)

////////////////////////////////////////////////////////////////////////////////
// Cluster and namespace configuration                                        //
//                                                                            //
// Queen       { genre: rock }                                                //
//   Queen        { released: 1973 }                                          //
//   Jazz         { released: 1978 }                                          //
//   Innuendo     { released: 1991 }                                          //
//                                                                            //
// Pink Floyd  { genre: psychedelic_rock }                                    //
//   The Wall     { released: 1979 }                                          //
//                                                                            //
// Deep Purple { genre: hard_rock }                                           //
//   Machine Head { released: 1972 }                                          //
//                                                                            //

var (
	clusterQueen = &storage.Cluster{
		Id:   "band.queen",
		Name: "Queen",
		Labels: map[string]string{
			"genre": "rock",
		},
	}

	clusterPinkFloyd = &storage.Cluster{
		Id:   "band.pinkfloyd",
		Name: "Pink Floyd",
		Labels: map[string]string{
			"genre": "psychedelic_rock",
		},
	}

	clusterDeepPurple = &storage.Cluster{
		Id:   "band.deeppurple",
		Name: "Deep Purple",
		Labels: map[string]string{
			"genre": "hard_rock",
		},
	}
)

var clusters = []effectiveaccessscope.Cluster{
	clusterQueen,
	clusterPinkFloyd,
	clusterDeepPurple,
}

var storageClusters = []*storage.Cluster{
	clusterQueen,
	clusterPinkFloyd,
	clusterDeepPurple,
}

var (
	namespaceQueenInClusterQueen = &storage.NamespaceMetadata{
		Id:          "album.queen",
		Name:        "Queen",
		ClusterId:   "band.queen",
		ClusterName: "Queen",
		Labels: map[string]string{
			"released": "1973",
		},
	}

	namespaceJazzInClusterQueen = &storage.NamespaceMetadata{
		Id:          "album.jazz",
		Name:        "Jazz",
		ClusterId:   "band.queen",
		ClusterName: "Queen",
		Labels: map[string]string{
			"released": "1978",
		},
	}

	namespaceInnuendoInClusterQueen = &storage.NamespaceMetadata{
		Id:          "album.innuendo",
		Name:        "Innuendo",
		ClusterId:   "band.queen",
		ClusterName: "Queen",
		Labels: map[string]string{
			"released": "1991",
		},
	}

	namespaceTheWallInClusterPinkFloyd = &storage.NamespaceMetadata{
		Id:          "album.thewall",
		Name:        "The Wall",
		ClusterId:   "band.pinkfloyd",
		ClusterName: "Pink Floyd",
		Labels: map[string]string{
			"released": "1979",
		},
	}

	namespaceMachineHeadInClusterDeepPurple = &storage.NamespaceMetadata{
		Id:          "album.machinehead",
		Name:        "Machine Head",
		ClusterId:   "band.deeppurple",
		ClusterName: "Deep Purple",
		Labels: map[string]string{
			"released": "1972",
		},
	}
)

var namespaces = []effectiveaccessscope.Namespace{
	// Queen
	namespaceQueenInClusterQueen,
	namespaceJazzInClusterQueen,
	namespaceInnuendoInClusterQueen,
	// Pink Floyd
	namespaceTheWallInClusterPinkFloyd,
	// Deep Purple
	namespaceMachineHeadInClusterDeepPurple,
}

var storageNamespaces = []*storage.NamespaceMetadata{
	// Queen
	namespaceQueenInClusterQueen,
	namespaceJazzInClusterQueen,
	namespaceInnuendoInClusterQueen,
	// Pink Floyd
	namespaceTheWallInClusterPinkFloyd,
	// Deep Purple
	namespaceMachineHeadInClusterDeepPurple,
}

////////////////////////////////////////////////////////////////////////////////
// Access scope rules and expected effective access scopes                    //
//                                                                            //
// Valid rules:                                                               //
//   `namespace: "Queen::Jazz" OR cluster.labels: genre in (psychedelic_rock)`//
//     => { "Queen::Jazz", "Pink Floyd::*" }                                  //
//                                                                            //
// Invalid rules:                                                             //
//   `namespace: "::Jazz"` => { }                                             //
//                                                                            //

var validRules = &storage.SimpleAccessScope_Rules{
	IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
		{
			ClusterName:   "Queen",
			NamespaceName: "Jazz",
		},
	},
	ClusterLabelSelectors: labels.LabelSelectors("genre", storage.SetBasedLabelSelector_IN, []string{"psychedelic_rock"}),
}

var validExpectedHigh = &storage.EffectiveAccessScope{
	Clusters: []*storage.EffectiveAccessScope_Cluster{
		{
			Id:    "band.deeppurple",
			Name:  "Deep Purple",
			State: storage.EffectiveAccessScope_EXCLUDED,
			Labels: map[string]string{
				"genre": "hard_rock",
			},
			Namespaces: []*storage.EffectiveAccessScope_Namespace{
				{
					Id:    "album.machinehead",
					Name:  "Machine Head",
					State: storage.EffectiveAccessScope_EXCLUDED,
					Labels: map[string]string{
						"released": "1972",
					},
				},
			},
		},
		{
			Id:    "band.pinkfloyd",
			Name:  "Pink Floyd",
			State: storage.EffectiveAccessScope_INCLUDED,
			Labels: map[string]string{
				"genre": "psychedelic_rock",
			},
			Namespaces: []*storage.EffectiveAccessScope_Namespace{
				{
					Id:    "album.thewall",
					Name:  "The Wall",
					State: storage.EffectiveAccessScope_INCLUDED,
					Labels: map[string]string{
						"released": "1979",
					},
				},
			},
		},
		{
			Id:    "band.queen",
			Name:  "Queen",
			State: storage.EffectiveAccessScope_PARTIAL,
			Labels: map[string]string{
				"genre": "rock",
			},
			Namespaces: []*storage.EffectiveAccessScope_Namespace{
				{
					Id:    "album.innuendo",
					Name:  "Innuendo",
					State: storage.EffectiveAccessScope_EXCLUDED,
					Labels: map[string]string{
						"released": "1991",
					},
				},
				{
					Id:    "album.jazz",
					Name:  "Jazz",
					State: storage.EffectiveAccessScope_INCLUDED,
					Labels: map[string]string{
						"released": "1978",
					},
				},
				{
					Id:    "album.queen",
					Name:  "Queen",
					State: storage.EffectiveAccessScope_EXCLUDED,
					Labels: map[string]string{
						"released": "1973",
					},
				},
			},
		},
	},
}

var validExpectedStandard = &storage.EffectiveAccessScope{
	Clusters: []*storage.EffectiveAccessScope_Cluster{
		{
			Id:    "band.deeppurple",
			Name:  "Deep Purple",
			State: storage.EffectiveAccessScope_EXCLUDED,
			Namespaces: []*storage.EffectiveAccessScope_Namespace{
				{
					Id:    "album.machinehead",
					Name:  "Machine Head",
					State: storage.EffectiveAccessScope_EXCLUDED,
				},
			},
		},
		{
			Id:    "band.pinkfloyd",
			Name:  "Pink Floyd",
			State: storage.EffectiveAccessScope_INCLUDED,
			Namespaces: []*storage.EffectiveAccessScope_Namespace{
				{
					Id:    "album.thewall",
					Name:  "The Wall",
					State: storage.EffectiveAccessScope_INCLUDED,
				},
			},
		},
		{
			Id:    "band.queen",
			Name:  "Queen",
			State: storage.EffectiveAccessScope_PARTIAL,
			Namespaces: []*storage.EffectiveAccessScope_Namespace{
				{
					Id:    "album.innuendo",
					Name:  "Innuendo",
					State: storage.EffectiveAccessScope_EXCLUDED,
				},
				{
					Id:    "album.jazz",
					Name:  "Jazz",
					State: storage.EffectiveAccessScope_INCLUDED,
				},
				{
					Id:    "album.queen",
					Name:  "Queen",
					State: storage.EffectiveAccessScope_EXCLUDED,
				},
			},
		},
	},
}

var validExpectedMinimal = &storage.EffectiveAccessScope{
	Clusters: []*storage.EffectiveAccessScope_Cluster{
		{
			Id:    "band.pinkfloyd",
			State: storage.EffectiveAccessScope_INCLUDED,
		},
		{
			Id:    "band.queen",
			State: storage.EffectiveAccessScope_PARTIAL,
			Namespaces: []*storage.EffectiveAccessScope_Namespace{
				{
					Id:    "album.jazz",
					State: storage.EffectiveAccessScope_INCLUDED,
				},
			},
		},
	},
}

var invalidRules = &storage.SimpleAccessScope_Rules{
	IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{
		{
			NamespaceName: "Jazz",
		},
	},
}

var invalidExpectedHigh = &storage.EffectiveAccessScope{
	Clusters: []*storage.EffectiveAccessScope_Cluster{
		{
			Id:    "band.deeppurple",
			Name:  "Deep Purple",
			State: storage.EffectiveAccessScope_EXCLUDED,
			Labels: map[string]string{
				"genre": "hard_rock",
			},
			Namespaces: []*storage.EffectiveAccessScope_Namespace{
				{
					Id:    "album.machinehead",
					Name:  "Machine Head",
					State: storage.EffectiveAccessScope_EXCLUDED,
					Labels: map[string]string{
						"released": "1972",
					},
				},
			},
		},
		{
			Id:    "band.pinkfloyd",
			Name:  "Pink Floyd",
			State: storage.EffectiveAccessScope_EXCLUDED,
			Labels: map[string]string{
				"genre": "psychedelic_rock",
			},
			Namespaces: []*storage.EffectiveAccessScope_Namespace{
				{
					Id:    "album.thewall",
					Name:  "The Wall",
					State: storage.EffectiveAccessScope_EXCLUDED,
					Labels: map[string]string{
						"released": "1979",
					},
				},
			},
		},
		{
			Id:    "band.queen",
			Name:  "Queen",
			State: storage.EffectiveAccessScope_EXCLUDED,
			Labels: map[string]string{
				"genre": "rock",
			},
			Namespaces: []*storage.EffectiveAccessScope_Namespace{
				{
					Id:    "album.innuendo",
					Name:  "Innuendo",
					State: storage.EffectiveAccessScope_EXCLUDED,
					Labels: map[string]string{
						"released": "1991",
					},
				},
				{
					Id:    "album.jazz",
					Name:  "Jazz",
					State: storage.EffectiveAccessScope_EXCLUDED,
					Labels: map[string]string{
						"released": "1978",
					},
				},
				{
					Id:    "album.queen",
					Name:  "Queen",
					State: storage.EffectiveAccessScope_EXCLUDED,
					Labels: map[string]string{
						"released": "1973",
					},
				},
			},
		},
	},
}

var invalidExpectedStandard = &storage.EffectiveAccessScope{
	Clusters: []*storage.EffectiveAccessScope_Cluster{
		{
			Id:    "band.deeppurple",
			Name:  "Deep Purple",
			State: storage.EffectiveAccessScope_EXCLUDED,
			Namespaces: []*storage.EffectiveAccessScope_Namespace{
				{
					Id:    "album.machinehead",
					Name:  "Machine Head",
					State: storage.EffectiveAccessScope_EXCLUDED,
				},
			},
		},
		{
			Id:    "band.pinkfloyd",
			Name:  "Pink Floyd",
			State: storage.EffectiveAccessScope_EXCLUDED,
			Namespaces: []*storage.EffectiveAccessScope_Namespace{
				{
					Id:    "album.thewall",
					Name:  "The Wall",
					State: storage.EffectiveAccessScope_EXCLUDED,
				},
			},
		},
		{
			Id:    "band.queen",
			Name:  "Queen",
			State: storage.EffectiveAccessScope_EXCLUDED,
			Namespaces: []*storage.EffectiveAccessScope_Namespace{
				{
					Id:    "album.innuendo",
					Name:  "Innuendo",
					State: storage.EffectiveAccessScope_EXCLUDED,
				},
				{
					Id:    "album.jazz",
					Name:  "Jazz",
					State: storage.EffectiveAccessScope_EXCLUDED,
				},
				{
					Id:    "album.queen",
					Name:  "Queen",
					State: storage.EffectiveAccessScope_EXCLUDED,
				},
			},
		},
	},
}

var invalidExpectedMinimal = &storage.EffectiveAccessScope{}

////////////////////////////////////////////////////////////////////////////////
// Tests                                                                      //
//                                                                            //

func TestEffectiveAccessScopeForSimpleAccessScope(t *testing.T) {
	type testCase struct {
		desc             string
		rules            *storage.SimpleAccessScope_Rules
		expectedHigh     *storage.EffectiveAccessScope
		expectedStandard *storage.EffectiveAccessScope
		expectedMinimal  *storage.EffectiveAccessScope
	}

	testCases := []testCase{
		{
			"valid access scope rules",
			validRules,
			validExpectedHigh,
			validExpectedStandard,
			validExpectedMinimal,
		},
		{
			"invalid access scope rules",
			invalidRules,
			invalidExpectedHigh,
			invalidExpectedStandard,
			invalidExpectedMinimal,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc+"detail: HIGH", func(t *testing.T) {
			resHigh, err := effectiveAccessScopeForSimpleAccessScope(tc.rules, clusters, namespaces, v1.ComputeEffectiveAccessScopeRequest_HIGH)
			assert.NoError(t, err)
			protoassert.Equal(t, tc.expectedHigh, resHigh)
		})
		t.Run(tc.desc+"detail: STANDARD", func(t *testing.T) {
			resStandard, err := effectiveAccessScopeForSimpleAccessScope(tc.rules, clusters, namespaces, v1.ComputeEffectiveAccessScopeRequest_STANDARD)
			assert.NoError(t, err)
			protoassert.Equal(t, tc.expectedStandard, resStandard)
		})
		t.Run(tc.desc+"detail: MINIMAL", func(t *testing.T) {
			resMinimal, err := effectiveAccessScopeForSimpleAccessScope(tc.rules, clusters, namespaces, v1.ComputeEffectiveAccessScopeRequest_MINIMAL)
			assert.NoError(t, err)
			protoassert.Equal(t, tc.expectedMinimal, resMinimal)
		})
		t.Run(tc.desc+"unknown detail maps to STANDARD", func(t *testing.T) {
			resUnknown, err := effectiveAccessScopeForSimpleAccessScope(tc.rules, clusters, namespaces, 42)
			assert.NoError(t, err)
			protoassert.Equal(t, tc.expectedStandard, resUnknown)
		})
	}
}
