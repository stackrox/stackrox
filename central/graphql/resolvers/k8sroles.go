package resolvers

import (
	"context"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/k8srbac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddExtraResolver("K8SRole", `type: String!`),
		schema.AddExtraResolver("K8SRole", `verbs: [String!]!`),
		schema.AddExtraResolver("K8SRole", `resources: [String!]!`),
		schema.AddExtraResolver("K8SRole", `urls: [String!]!`),
		schema.AddExtraResolver("K8SRole", `subjects: [SubjectWithClusterID!]!`),
		schema.AddExtraResolver("K8SRole", `serviceAccounts: [ServiceAccount!]!`),
		schema.AddExtraResolver("K8SRole", `roleNamespace: Namespace!`),
	)
}

func (resolver *k8SRoleResolver) Type(ctx context.Context) (string, error) {
	if err := readK8sRoles(ctx); err != nil {
		return "", err
	}

	if resolver.data.GetClusterRole() {
		return "ClusterRole", nil
	}
	return "Role", nil
}

// Verbs returns the set of verbs granted by a given k8s role
func (resolver *k8SRoleResolver) Verbs(ctx context.Context) ([]string, error) {
	if err := readK8sRoles(ctx); err != nil {
		return nil, err
	}

	return k8srbac.GetVerbsForRole(resolver.data).AsSlice(), nil

}

// Resources returns the set of resources that have been granted permissions to by a given k8s role
func (resolver *k8SRoleResolver) Resources(ctx context.Context) ([]string, error) {
	if err := readK8sRoles(ctx); err != nil {
		return nil, err
	}
	return k8srbac.GetResourcesForRole(resolver.data).AsSlice(), nil

}

// NonResourceURLs returns the set of non resource urls granted permissions to by a given k8s role
func (resolver *k8SRoleResolver) Urls(ctx context.Context) ([]string, error) {
	if err := readK8sRoles(ctx); err != nil {
		return nil, err
	}
	return k8srbac.GetNonResourceURLsForRole(resolver.data).AsSlice(), nil
}

// Subjects returns the set of subjects granted permissions to by a given k8s role
func (resolver *k8SRoleResolver) Subjects(ctx context.Context) ([]*subjectWithClusterIDResolver, error) {
	subjects := make([]*subjectWithClusterIDResolver, 0)
	if err := readK8sSubjects(ctx); err != nil {
		return nil, err
	}

	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetClusterId()).
		AddExactMatches(search.RoleID, resolver.data.GetId()).ProtoQuery()
	bindings, err := resolver.root.K8sRoleBindingStore.SearchRawRoleBindings(ctx, q)

	if err != nil {
		return subjects, err
	}

	subs, err := resolver.root.wrapSubjects(
		k8srbac.GetAllSubjects(bindings,
			storage.SubjectKind_USER, storage.SubjectKind_GROUP), nil)
	if err != nil {
		return subjects, err
	}

	return wrapSubjects(resolver.data.GetClusterId(), subs), nil
}

// ServiceAccounts returns the set of service accounts granted permissions to by a given k8s role
func (resolver *k8SRoleResolver) ServiceAccounts(ctx context.Context) ([]*serviceAccountResolver, error) {
	if err := readServiceAccounts(ctx); err != nil {
		return nil, err
	}

	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetClusterId()).
		AddExactMatches(search.RoleID, resolver.data.GetId()).ProtoQuery()
	bindings, err := resolver.root.K8sRoleBindingStore.SearchRawRoleBindings(ctx, q)

	if err != nil {
		return nil, err
	}

	subjects := k8srbac.GetAllSubjects(bindings, storage.SubjectKind_SERVICE_ACCOUNT)
	saResolvers := make([]*serviceAccountResolver, 0, len(subjects))

	for i, subject := range subjects {
		sa, err := resolver.convertSubjectToServiceAccount(ctx, resolver.data.GetClusterId(), subject)
		if err != nil {
			continue
		}
		saResolvers[i], err = resolver.root.wrapServiceAccount(sa, true, nil)
		if err != nil {
			return nil, err
		}
	}

	return saResolvers, nil
}

// RoleNamespace returns the namespace of the k8s role
func (resolver *k8SRoleResolver) RoleNamespace(ctx context.Context) (*namespaceResolver, error) {
	role := resolver.data
	r, err := resolver.root.NamespaceByClusterIDAndName(ctx, clusterIDAndNameQuery{graphql.ID(role.GetClusterId()), role.GetNamespace()})

	if err != nil {
		return resolver.root.wrapNamespace(r.data, false, err)
	}

	return resolver.root.wrapNamespace(r.data, true, err)
}

func (resolver *k8SRoleResolver) convertSubjectToServiceAccount(ctx context.Context, clusterID string, subject *storage.Subject) (*storage.ServiceAccount, error) {
	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).
		AddExactMatches(search.ServiceAccountName, subject.GetName()).ProtoQuery()

	serviceAccounts, err := resolver.root.ServiceAccountsDataStore.SearchRawServiceAccounts(ctx, q)
	if err != nil {
		return nil, err
	}
	if len(serviceAccounts) == 0 {
		return nil, nil
	}
	return serviceAccounts[0], nil
}
