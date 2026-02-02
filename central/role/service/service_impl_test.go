//go:build sql_integration

package service

import (
	"context"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn/basic"
	"github.com/stackrox/rox/pkg/labels"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
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

func TestGetMyPermissions(t *testing.T) {
	suite.Run(t, new(roleServiceGetMyPermissionsTestSuite))
}

const (
	getMyPermissionsServiceName = "/v1.RoleService/GetMyPermissions"
)

type roleServiceGetMyPermissionsTestSuite struct {
	suite.Suite

	svc *serviceImpl

	withAdminRoleCtx context.Context
	withNoneRoleCtx  context.Context
	withNoAccessCtx  context.Context
	withNoRoleCtx    context.Context
	anonymousCtx     context.Context
}

func (s *roleServiceGetMyPermissionsTestSuite) SetupTest() {
	s.svc = &serviceImpl{}

	authProvider, err := authproviders.NewProvider(
		authproviders.WithEnabled(true),
		authproviders.WithID(uuid.NewDummy().String()),
		authproviders.WithName("Test Auth Provider"),
	)
	s.Require().NoError(err)
	s.withAdminRoleCtx = basic.ContextWithAdminIdentity(s.T(), authProvider)
	s.withNoneRoleCtx = basic.ContextWithNoneIdentity(s.T(), authProvider)
	s.withNoAccessCtx = basic.ContextWithNoAccessIdentity(s.T(), authProvider)
	s.withNoRoleCtx = basic.ContextWithNoRoleIdentity(s.T(), authProvider)
	s.anonymousCtx = context.Background()
}

type testCase struct {
	name string
	ctx  context.Context

	expectedPermissionCount int
	expectedAuthorizerError error
	expectedServiceError    error
}

func (s *roleServiceGetMyPermissionsTestSuite) getTestCases() []testCase {
	return []testCase{
		{
			name: accesscontrol.Admin,
			ctx:  s.withAdminRoleCtx,

			expectedPermissionCount: len(resources.ListAll()),
			expectedServiceError:    nil,
			expectedAuthorizerError: nil,
		},
		{
			name: accesscontrol.None,
			ctx:  s.withNoneRoleCtx,

			expectedPermissionCount: 0,
			expectedServiceError:    nil,
			expectedAuthorizerError: errox.NoCredentials,
		},
		{
			name: "No Access",
			ctx:  s.withNoAccessCtx,

			expectedPermissionCount: len(resources.ListAll()),
			expectedServiceError:    nil,
			expectedAuthorizerError: nil,
		},
		{
			name: "No Role",
			ctx:  s.withNoRoleCtx,

			expectedPermissionCount: 0,
			expectedServiceError:    nil,
			expectedAuthorizerError: errox.NoCredentials,
		},
		{
			name: "Anonymous",
			ctx:  s.anonymousCtx,

			expectedPermissionCount: 0,
			expectedServiceError:    errox.NoCredentials,
			expectedAuthorizerError: errox.NoCredentials,
		},
	}
}

func (s *roleServiceGetMyPermissionsTestSuite) TestAuthorizer() {
	for _, c := range s.getTestCases() {
		s.Run(c.name, func() {
			ctx, err := s.svc.AuthFuncOverride(c.ctx, getMyPermissionsServiceName)
			s.ErrorIs(err, c.expectedAuthorizerError)
			s.Equal(c.ctx, ctx)
		})
	}
}

func (s *roleServiceGetMyPermissionsTestSuite) TestGetMyPermissions() {
	emptyRequest := &v1.Empty{}
	for _, c := range s.getTestCases() {
		s.Run(c.name, func() {
			rsp, err := s.svc.GetMyPermissions(c.ctx, emptyRequest)
			s.ErrorIs(err, c.expectedServiceError)
			if c.expectedServiceError == nil {
				s.NotNil(rsp)
				if rsp != nil {
					s.Len(rsp.GetResourceToAccess(), c.expectedPermissionCount)
				}
			} else {
				s.Nil(rsp)
			}
		})
	}
}
