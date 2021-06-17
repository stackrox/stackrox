package authorizer

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/stackrox/default-authz-plugin/pkg/payload"
	clusterDataStoreMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	namespaceMocks "github.com/stackrox/rox/central/namespace/datastore/mocks"
	roleMocks "github.com/stackrox/rox/central/role/datastore/mocks"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/sac/client"
	"github.com/stretchr/testify/assert"
)

var (
	testRole     = "test-role"
	firstCluster = payload.Cluster{
		ID:   "cluster-1",
		Name: "FirstCluster",
	}
	secondCluster = payload.Cluster{
		ID:   "cluster-2",
		Name: "SecondCluster",
	}

	firstNamespaceName  = "FirstNamespace"
	secondNamespaceName = "SecondNamespace"
)

func TestBuiltInScopeAuthorizer_ForUser(t *testing.T) {
	cluster := string(resources.Cluster.Resource)
	namespace := string(resources.Namespace.Resource)
	globalResource := string(resources.APIToken.Resource)

	allResourcesView := mapResourcesToAccess(resources.AllResourcesViewPermissions())
	allResourcesEdit := mapResourcesToAccess(resources.AllResourcesModifyPermissions())
	firstClusterScope := payload.AccessScope{
		Verb:       view,
		Noun:       cluster,
		Attributes: payload.NounAttributes{Cluster: firstCluster},
	}
	secondClusterScope := payload.AccessScope{
		Verb: view, Noun: cluster,
		Attributes: payload.NounAttributes{Cluster: secondCluster},
	}
	adminRolePrincipal := payload.Principal{
		Roles: []string{testRole},
	}
	principalWithNoRoles := payload.Principal{}
	mockError := errors.New("some error")

	tests := []struct {
		name      string
		mockSetup func(clusterStore *clusterDataStoreMocks.MockDataStore, nsStore *namespaceMocks.MockDataStore, roleStore *roleMocks.MockDataStore)
		principal payload.Principal
		scopes    []payload.AccessScope
		allowed   []payload.AccessScope
		denied    []payload.AccessScope
		wantErr   bool
	}{
		{
			name: "get clusters error => error",
			mockSetup: func(clusterStore *clusterDataStoreMocks.MockDataStore, nsStore *namespaceMocks.MockDataStore, roleStore *roleMocks.MockDataStore) {
				clusterStore.EXPECT().GetClusters(gomock.Any()).Return(nil, mockError).Times(1)
			},
			principal: principalWithNoRoles,
			wantErr:   true,
		},
		{
			name: "get namespaces error => error",
			mockSetup: func(clusterStore *clusterDataStoreMocks.MockDataStore, nsStore *namespaceMocks.MockDataStore, roleStore *roleMocks.MockDataStore) {
				clusterStore.EXPECT().GetClusters(gomock.Any()).Return(nil, nil).Times(1)
				nsStore.EXPECT().GetNamespaces(gomock.Any()).Return(nil, mockError).Times(1)
			},
			principal: principalWithNoRoles,
			wantErr:   true,
		},
		{
			name: "get roles error => error",
			mockSetup: func(clusterStore *clusterDataStoreMocks.MockDataStore, nsStore *namespaceMocks.MockDataStore, roleStore *roleMocks.MockDataStore) {
				roleStore.EXPECT().GetAndResolveRole(gomock.Any(), gomock.Any()).Return(nil, mockError).Times(1)
			},
			principal: adminRolePrincipal,
			wantErr:   true,
		},
		{
			name:      "invalid resource name => error",
			mockSetup: noInteractions,
			principal: adminRolePrincipal,
			scopes: []payload.AccessScope{
				{
					Verb:       view,
					Noun:       "unknown",
					Attributes: payload.NounAttributes{},
				},
			},
			wantErr: true,
		},
		{
			name: "compute access scope error => error",
			mockSetup: func(clusterStore *clusterDataStoreMocks.MockDataStore, nsStore *namespaceMocks.MockDataStore, roleStore *roleMocks.MockDataStore) {
				withTwoClusters(clusterStore)
				withNoNamespaces(nsStore)
				withRoles(roleStore, role(
					withResourceToAccess(allResourcesEdit),
					&storage.SimpleAccessScope{
						Id: "2",
						Rules: &storage.SimpleAccessScope_Rules{
							ClusterLabelSelectors: []*storage.SetBasedLabelSelector{{
								Requirements: []*storage.SetBasedLabelSelector_Requirement{
									{Key: "invalid key"},
								}}}}}))
			},
			principal: adminRolePrincipal,
			scopes: []payload.AccessScope{
				firstClusterScope,
				secondClusterScope,
			},
			wantErr: true,
		},
		{
			name: "simple sac with no resources to access",
			mockSetup: func(clusterStore *clusterDataStoreMocks.MockDataStore, nsStore *namespaceMocks.MockDataStore, roleStore *roleMocks.MockDataStore) {
				withTwoClusters(clusterStore)
				withNoNamespaces(nsStore)
				withRoles(roleStore, role(
					withResourceToAccess(nil),
					withAccessTo1Cluster()),
				)
			},
			principal: adminRolePrincipal,
			scopes: []payload.AccessScope{
				firstClusterScope,
				secondClusterScope,
			},
			denied: []payload.AccessScope{firstClusterScope, {
				Verb: view, Noun: cluster,
				Attributes: payload.NounAttributes{Cluster: secondCluster},
			}},
		},
		{
			name: "simple sac with no access to cluster",
			mockSetup: func(clusterStore *clusterDataStoreMocks.MockDataStore, nsStore *namespaceMocks.MockDataStore, roleStore *roleMocks.MockDataStore) {
				withTwoClusters(clusterStore)
				withNoNamespaces(nsStore)
				withRoles(roleStore, role(
					withResourceToAccess(map[string]storage.Access{cluster: storage.Access_NO_ACCESS}),
					withAccessTo1Cluster(),
				))
			},
			principal: adminRolePrincipal,
			scopes: []payload.AccessScope{
				firstClusterScope,
				secondClusterScope,
			},
			denied: []payload.AccessScope{firstClusterScope, {
				Verb: view, Noun: cluster,
				Attributes: payload.NounAttributes{Cluster: secondCluster},
			}},
		},
		{
			name: "simple sac global scope resource",
			mockSetup: func(clusterStore *clusterDataStoreMocks.MockDataStore, nsStore *namespaceMocks.MockDataStore, roleStore *roleMocks.MockDataStore) {
				withTwoClusters(clusterStore)
				withNoNamespaces(nsStore)
				withRoles(roleStore, role(
					withResourceToAccess(map[string]storage.Access{globalResource: storage.Access_READ_ACCESS}),
					withAccessTo1Cluster()))
			},
			principal: adminRolePrincipal,
			scopes: []payload.AccessScope{{
				Verb: view, Noun: globalResource,
			}},
			allowed: []payload.AccessScope{{
				Verb: view, Noun: globalResource,
			}},
		},
		{
			name: "simple sac with included cluster and no namespaces",
			mockSetup: func(clusterStore *clusterDataStoreMocks.MockDataStore, nsStore *namespaceMocks.MockDataStore, roleStore *roleMocks.MockDataStore) {
				withTwoClusters(clusterStore)
				withNoNamespaces(nsStore)
				withRoles(roleStore, role(
					withResourceToAccess(allResourcesEdit),
					withAccessTo1Cluster()))
			},
			principal: adminRolePrincipal,
			scopes: []payload.AccessScope{
				firstClusterScope,
				secondClusterScope,
			},
			allowed: []payload.AccessScope{firstClusterScope},
			denied: []payload.AccessScope{{
				Verb: view, Noun: cluster,
				Attributes: payload.NounAttributes{Cluster: secondCluster},
			}},
		},
		{
			name: "simple sac with included cluster and no namespace name in attributes",
			mockSetup: func(clusterStore *clusterDataStoreMocks.MockDataStore, nsStore *namespaceMocks.MockDataStore, roleStore *roleMocks.MockDataStore) {
				withTwoClusters(clusterStore)
				withTwoNamespaces(nsStore)
				withRoles(roleStore, role(
					withResourceToAccess(allResourcesEdit),
					withAccessTo1Namespace()))
			},
			principal: adminRolePrincipal,
			scopes: []payload.AccessScope{
				{
					Verb: view, Noun: namespace,
					Attributes: payload.NounAttributes{Cluster: firstCluster},
				},
			},
			denied: []payload.AccessScope{{
				Verb: view, Noun: namespace,
				Attributes: payload.NounAttributes{Cluster: firstCluster},
			}},
		},
		{
			name: "simple sac with empty permission set and access scope",
			mockSetup: func(clusterStore *clusterDataStoreMocks.MockDataStore, nsStore *namespaceMocks.MockDataStore, roleStore *roleMocks.MockDataStore) {
				withTwoClusters(clusterStore)
				withNoNamespaces(nsStore)
				withRoles(roleStore, role(withResourceToAccess(nil), nil))
			},
			principal: adminRolePrincipal,
			scopes: []payload.AccessScope{
				firstClusterScope,
				secondClusterScope,
			},
			denied: []payload.AccessScope{
				firstClusterScope,
				{
					Verb: view, Noun: cluster,
					Attributes: payload.NounAttributes{Cluster: secondCluster},
				}},
		},
		{
			name:      "scope omits noun but declares cluster => error",
			mockSetup: noInteractions,
			principal: adminRolePrincipal,
			scopes: []payload.AccessScope{
				{Attributes: payload.NounAttributes{Cluster: firstCluster}},
				{Attributes: payload.NounAttributes{Cluster: secondCluster}},
			},
			wantErr: true,
		},
		{
			name: "just a verb and no noun => require verb to all resources",
			mockSetup: func(clusterStore *clusterDataStoreMocks.MockDataStore, nsStore *namespaceMocks.MockDataStore, roleStore *roleMocks.MockDataStore) {
				withTwoClusters(clusterStore)
				withNoNamespaces(nsStore)
				withRoles(roleStore, role(
					withResourceToAccess(allResourcesEdit),
					withAccessTo1Cluster()))
			},
			principal: adminRolePrincipal,
			scopes:    []payload.AccessScope{{Verb: view}},
			allowed:   []payload.AccessScope{{Verb: view}},
		},
		{
			name: "no verb and no noun => require edit to all",
			mockSetup: func(clusterStore *clusterDataStoreMocks.MockDataStore, nsStore *namespaceMocks.MockDataStore, roleStore *roleMocks.MockDataStore) {
				withTwoClusters(clusterStore)
				withNoNamespaces(nsStore)
				withRoles(roleStore, role(
					withResourceToAccess(allResourcesEdit),
					withAccessTo1Cluster()))
			},
			principal: adminRolePrincipal,
			scopes:    []payload.AccessScope{{}},
			allowed:   []payload.AccessScope{{}},
		},
		{
			name: "no verb and no noun => require edit to all",
			mockSetup: func(clusterStore *clusterDataStoreMocks.MockDataStore, nsStore *namespaceMocks.MockDataStore, roleStore *roleMocks.MockDataStore) {
				withTwoClusters(clusterStore)
				withNoNamespaces(nsStore)
				withRoles(roleStore, role(
					withResourceToAccess(allResourcesView),
					withAccessTo1Cluster()))
			},
			principal: adminRolePrincipal,
			scopes:    []payload.AccessScope{{}},
			denied:    []payload.AccessScope{{}},
		},
		{
			name: "no access scope id => allow everything in permission set",
			mockSetup: func(clusterStore *clusterDataStoreMocks.MockDataStore, nsStore *namespaceMocks.MockDataStore, roleStore *roleMocks.MockDataStore) {
				withTwoClusters(clusterStore)
				withNoNamespaces(nsStore)
				withRoles(roleStore, role(
					withResourceToAccess(allResourcesView),
					nil))
			},
			principal: adminRolePrincipal,
			scopes: []payload.AccessScope{{
				Verb: view, Noun: namespace,
				Attributes: payload.NounAttributes{Cluster: firstCluster, Namespace: firstNamespaceName},
			}, {
				Verb: view, Noun: namespace,
				Attributes: payload.NounAttributes{Cluster: firstCluster, Namespace: secondNamespaceName},
			}},
			allowed: []payload.AccessScope{{
				Verb: view, Noun: namespace,
				Attributes: payload.NounAttributes{Cluster: firstCluster, Namespace: firstNamespaceName},
			}, {
				Verb: view, Noun: namespace,
				Attributes: payload.NounAttributes{Cluster: firstCluster, Namespace: secondNamespaceName},
			}},
		},
		{
			name: "<Verb, Noun, Cluster, *> => allows partial clusters",
			mockSetup: func(clusterStore *clusterDataStoreMocks.MockDataStore, nsStore *namespaceMocks.MockDataStore, roleStore *roleMocks.MockDataStore) {
				withTwoClusters(clusterStore)
				withTwoNamespaces(nsStore)
				withRoles(roleStore, role(
					withResourceToAccess(allResourcesView),
					withAccessTo1Namespace()))
			},
			principal: adminRolePrincipal,
			scopes:    []payload.AccessScope{firstClusterScope},
			allowed:   []payload.AccessScope{firstClusterScope},
		},
		{
			name: "no roles => no access",
			mockSetup: func(clusterStore *clusterDataStoreMocks.MockDataStore, nsStore *namespaceMocks.MockDataStore, roleStore *roleMocks.MockDataStore) {
				withTwoClusters(clusterStore)
				withTwoNamespaces(nsStore)
			},
			principal: principalWithNoRoles,
			scopes:    []payload.AccessScope{firstClusterScope},
			denied:    []payload.AccessScope{firstClusterScope},
		},
		{
			name: "<Verb, Noun, Cluster, Namespace> => allows only included namespaces",
			mockSetup: func(clusterStore *clusterDataStoreMocks.MockDataStore, nsStore *namespaceMocks.MockDataStore, roleStore *roleMocks.MockDataStore) {
				withTwoClusters(clusterStore)
				withTwoNamespaces(nsStore)
				withRoles(roleStore, role(
					withResourceToAccess(allResourcesView),
					withAccessTo1Namespace()))
			},
			principal: adminRolePrincipal,
			scopes: []payload.AccessScope{{
				Verb: view, Noun: namespace,
				Attributes: payload.NounAttributes{Cluster: firstCluster, Namespace: firstNamespaceName},
			}, {
				Verb: view, Noun: namespace,
				Attributes: payload.NounAttributes{Cluster: firstCluster, Namespace: secondNamespaceName},
			}},
			allowed: []payload.AccessScope{{
				Verb: view, Noun: namespace,
				Attributes: payload.NounAttributes{Cluster: firstCluster, Namespace: firstNamespaceName},
			}},
			denied: []payload.AccessScope{{
				Verb: view, Noun: namespace,
				Attributes: payload.NounAttributes{Cluster: firstCluster, Namespace: secondNamespaceName},
			}},
		},
		{
			name: "multiple roles => union of all roles permissions",
			mockSetup: func(clusterStore *clusterDataStoreMocks.MockDataStore, nsStore *namespaceMocks.MockDataStore, roleStore *roleMocks.MockDataStore) {
				withTwoClusters(clusterStore)
				withTwoNamespaces(nsStore)
				withRoles(roleStore,
					roleWithName("firstRole",
						withResourceToAccess(allResourcesView),
						withAccessTo1Namespace()),
					roleWithName("secondRole",
						withResourceToAccess(allResourcesView),
						&storage.SimpleAccessScope{
							Rules: &storage.SimpleAccessScope_Rules{
								IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{{
									ClusterName: firstCluster.Name, NamespaceName: secondNamespaceName}}},
						}))
			},
			principal: payload.Principal{
				Roles: []string{"firstRole", "secondRole", "secondRole"},
			},
			scopes: []payload.AccessScope{{
				Verb: view, Noun: namespace,
				Attributes: payload.NounAttributes{Cluster: firstCluster, Namespace: firstNamespaceName},
			}, {
				Verb: view, Noun: namespace,
				Attributes: payload.NounAttributes{Cluster: firstCluster, Namespace: secondNamespaceName},
			}},
			allowed: []payload.AccessScope{{
				Verb: view, Noun: namespace,
				Attributes: payload.NounAttributes{Cluster: firstCluster, Namespace: firstNamespaceName},
			}, {
				Verb: view, Noun: namespace,
				Attributes: payload.NounAttributes{Cluster: firstCluster, Namespace: secondNamespaceName},
			}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, ctrl := mockedClient(t, tc.mockSetup)
			defer ctrl.Finish()
			allowed, denied, err := c.ForUser(context.Background(), tc.principal, tc.scopes...)
			assert.Truef(t, (err != nil) == tc.wantErr, "got %+v", err)
			assert.Equal(t, tc.allowed, allowed, "allowed mismatch")
			assert.Equal(t, tc.denied, denied, "denied mismatch")
		})
	}
}

func withRoles(roleStore *roleMocks.MockDataStore, roles ...*permissions.ResolvedRole) {
	for _, role := range roles {
		roleStore.EXPECT().GetAndResolveRole(gomock.Any(), role.Role.GetName()).Return(role, nil)
	}
}

func withAccessTo1Cluster() *storage.SimpleAccessScope {
	return &storage.SimpleAccessScope{
		Id: "2",
		Rules: &storage.SimpleAccessScope_Rules{
			IncludedClusters: []string{firstCluster.Name},
		},
	}
}

func withAccessTo1Namespace() *storage.SimpleAccessScope {
	return &storage.SimpleAccessScope{
		Id: "2",
		Rules: &storage.SimpleAccessScope_Rules{
			IncludedNamespaces: []*storage.SimpleAccessScope_Rules_Namespace{{
				ClusterName:   firstCluster.Name,
				NamespaceName: firstNamespaceName,
			}},
		},
	}
}

func withResourceToAccess(resourceToAccess map[string]storage.Access) *storage.PermissionSet {
	return &storage.PermissionSet{
		Id:               "1",
		ResourceToAccess: resourceToAccess,
	}
}

func withTwoNamespaces(nsStore *namespaceMocks.MockDataStore) *gomock.Call {
	return nsStore.EXPECT().GetNamespaces(gomock.Any()).Return([]*storage.NamespaceMetadata{{
		Id:          "namespace-1",
		Name:        firstNamespaceName,
		ClusterId:   firstCluster.ID,
		ClusterName: firstCluster.Name,
	}, {
		Id:          "namespace-2",
		Name:        secondNamespaceName,
		ClusterId:   firstCluster.ID,
		ClusterName: firstCluster.Name,
	}}, nil).Times(1)
}

func role(ps *storage.PermissionSet, as *storage.SimpleAccessScope) *permissions.ResolvedRole {
	return roleWithName(testRole, ps, as)
}

func roleWithName(name string, ps *storage.PermissionSet, as *storage.SimpleAccessScope) *permissions.ResolvedRole {
	return &permissions.ResolvedRole{
		Role: &storage.Role{
			Name:            name,
			PermissionSetId: ps.GetId(),
			AccessScopeId:   as.GetId(),
		},
		PermissionSet: ps,
		AccessScope:   as,
	}
}

func withNoNamespaces(nsStore *namespaceMocks.MockDataStore) *gomock.Call {
	return nsStore.EXPECT().GetNamespaces(gomock.Any()).Return(nil, nil).Times(1)
}

func withTwoClusters(clusterStore *clusterDataStoreMocks.MockDataStore) *gomock.Call {
	return clusterStore.EXPECT().GetClusters(gomock.Any()).Return([]*storage.Cluster{
		{Id: firstCluster.ID, Name: firstCluster.Name}, {Id: secondCluster.ID, Name: secondCluster.Name}}, nil).Times(1)
}

func mapResourcesToAccess(res []permissions.ResourceWithAccess) map[string]storage.Access {
	idToAccess := make(map[string]storage.Access, len(res))
	for _, rwa := range res {
		idToAccess[rwa.Resource.String()] = rwa.Access
	}
	return idToAccess
}

func noInteractions(_ *clusterDataStoreMocks.MockDataStore, _ *namespaceMocks.MockDataStore, _ *roleMocks.MockDataStore) {
}

func mockedClient(t *testing.T, setupMocks func(clusterStore *clusterDataStoreMocks.MockDataStore, nsStore *namespaceMocks.MockDataStore, roleStore *roleMocks.MockDataStore)) (client.Client, *gomock.Controller) {
	mockCtrl := gomock.NewController(t)
	clusterStore := clusterDataStoreMocks.NewMockDataStore(mockCtrl)
	nsStore := namespaceMocks.NewMockDataStore(mockCtrl)
	roleStore := roleMocks.NewMockDataStore(mockCtrl)

	setupMocks(clusterStore, nsStore, roleStore)

	return &builtInScopeAuthorizer{
		clusterStore:   clusterStore,
		namespaceStore: nsStore,
		roleStore:      roleStore,
	}, mockCtrl
}
