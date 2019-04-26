package resolvers

import (
	"context"

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
		schema.AddExtraResolver("K8SRole", `subjects: [Subject!]!`),
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
func (resolver *k8SRoleResolver) Subjects(ctx context.Context) ([]*subjectResolver, error) {
	subjects := make([]*subjectResolver, 0)
	if err := readK8sSubjects(ctx); err != nil {
		return nil, err
	}

	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetClusterId()).
		AddExactMatches(search.RoleID, resolver.data.GetId()).ProtoQuery()
	bindings, err := resolver.root.K8sRoleBindingStore.SearchRawRoleBindings(q)

	if err != nil {
		return subjects, err
	}

	return resolver.root.wrapSubjects(
		k8srbac.GetAllSubjects(bindings,
			storage.SubjectKind_USER, storage.SubjectKind_GROUP, storage.SubjectKind_SERVICE_ACCOUNT), nil)
}
