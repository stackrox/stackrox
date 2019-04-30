package resolvers

import (
	"context"
	"sort"

	utils2 "github.com/stackrox/rox/central/rbac/utils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/k8srbac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddType("StringListEntry", []string{"key: String!", "values: [String!]!"}),
		schema.AddType("ScopedPermissions", []string{"scope: String!", "permissions: [StringListEntry!]!"}),
		schema.AddExtraResolver("ServiceAccount", `roles: [K8SRole!]!`),
		schema.AddExtraResolver("ServiceAccount", `scopedPermissions: [ScopedPermissions!]!`),
		schema.AddExtraResolver("ServiceAccount", `deployments: [Deployment!]!`),
	)
}

type stringListEntryResolver struct {
	key    string
	values set.StringSet
}

type scopedPermissionsResolver struct {
	scope       string
	permissions []*stringListEntryResolver
}

func (resolver *stringListEntryResolver) Key(ctx context.Context) string {
	return resolver.key
}

func (resolver *stringListEntryResolver) Values(ctx context.Context) []string {
	return resolver.values.AsSlice()
}

func wrapStringListEntries(values map[string]set.StringSet) []*stringListEntryResolver {
	if len(values) == 0 {
		return nil
	}

	output := make([]*stringListEntryResolver, 0, len(values))
	for i, v := range values {
		output = append(output, &stringListEntryResolver{i, v})
	}

	return output
}

func (resolver *scopedPermissionsResolver) Scope(ctx context.Context) string {
	return resolver.scope
}

func (resolver *scopedPermissionsResolver) Permissions(ctx context.Context) []*stringListEntryResolver {
	return resolver.permissions
}

func (resolver *serviceAccountResolver) wrapPermissions(values map[string]map[string]set.StringSet) []*scopedPermissionsResolver {
	if len(values) == 0 {
		return nil
	}
	output := make([]*scopedPermissionsResolver, 0, len(values))
	for scope, permissions := range values {
		output = append(output, &scopedPermissionsResolver{scope, wrapStringListEntries(permissions)})
	}

	sort.SliceStable(output, func(i, j int) bool { return output[i].scope < output[j].scope })
	return output
}

func (resolver *serviceAccountResolver) Roles(ctx context.Context) ([]*k8SRoleResolver, error) {
	if err := readK8sRoles(ctx); err != nil {
		return nil, err
	}
	if err := readK8sRoleBindings(ctx); err != nil {
		return nil, err
	}

	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetClusterId()).ProtoQuery()
	bindings, err := resolver.root.K8sRoleBindingStore.SearchRawRoleBindings(q)

	if err != nil {
		return nil, err
	}

	q = search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetClusterId()).ProtoQuery()
	roles, err := resolver.root.K8sRoleStore.SearchRawRoles(q)

	if err != nil {
		return nil, err
	}

	subject := &storage.Subject{
		Name:      resolver.data.GetName(),
		Namespace: resolver.data.GetNamespace(),
		Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
	}

	return resolver.root.wrapK8SRoles(k8srbac.NewEvaluator(roles, bindings).RolesForSubject(subject), nil)
}

func (resolver *serviceAccountResolver) Deployments(ctx context.Context) ([]*deploymentResolver, error) {
	if err := readDeployments(ctx); err != nil {
		return nil, err
	}

	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetClusterId()).
		AddExactMatches(search.ServiceAccountName, resolver.data.GetName()).ProtoQuery()
	deployments, err := resolver.root.DeploymentDataStore.SearchListDeployments(q)

	if err != nil {
		return nil, err
	}

	return resolver.root.wrapListDeployments(deployments, nil)
}

// Permission returns which scopes do the permissions for the service acc
func (resolver *serviceAccountResolver) ScopedPermissions(ctx context.Context) ([]*scopedPermissionsResolver, error) {
	if err := readK8sRoles(ctx); err != nil {
		return nil, err
	}

	if err := readK8sRoleBindings(ctx); err != nil {
		return nil, err
	}

	evaluators, err := resolver.getEvaluators(ctx)
	if err != nil {
		return nil, err
	}

	subject := &storage.Subject{
		Name:      resolver.data.GetName(),
		Namespace: resolver.data.GetNamespace(),
		Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
	}

	permissionScopeMap := make(map[string]map[string]set.StringSet)
	for scope, evaluator := range evaluators {
		permissions := evaluator.ForSubject(subject).GetPermissionMap()
		if len(permissions) != 0 {
			permissionScopeMap[scope] = permissions
		}
	}

	return resolver.wrapPermissions(permissionScopeMap), nil
}

func (resolver *serviceAccountResolver) getEvaluators(ctx context.Context) (map[string]k8srbac.Evaluator, error) {
	evaluators := make(map[string]k8srbac.Evaluator)
	saClusterID := resolver.data.GetClusterId()

	evaluators["Cluster"] =
		utils2.NewClusterPermissionEvaluator(saClusterID,
			resolver.root.K8sRoleStore, resolver.root.K8sRoleBindingStore)

	namespaces, err := resolver.root.Namespaces(ctx)
	if err != nil {
		return evaluators, err
	}
	for _, namespace := range namespaces {
		namespaceName := namespace.data.GetMetadata().GetName()
		evaluators[namespaceName] = utils2.NewNamespacePermissionEvaluator(saClusterID,
			namespaceName, resolver.root.K8sRoleStore, resolver.root.K8sRoleBindingStore)
	}

	return evaluators, nil
}
