package resolvers

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/rbac/service"
	rbacUtils "github.com/stackrox/rox/central/rbac/utils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/k8srbac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("subjects(query: String): [Subject!]!"),
		schema.AddExtraResolver("Subject", `subjectWithClusterID: [SubjectWithClusterID!]!`),
		schema.AddExtraResolver("SubjectWithClusterID", `name: String!`),
		schema.AddExtraResolver("SubjectWithClusterID", `namespace: String!`),
		schema.AddExtraResolver("SubjectWithClusterID", `type: String!`),
		schema.AddExtraResolver("SubjectWithClusterID", `type: String!`),
		schema.AddExtraResolver("SubjectWithClusterID", `roles(query: String): [K8SRole!]!`),
		schema.AddExtraResolver("SubjectWithClusterID", `scopedPermissions: [ScopedPermissions!]!`),
		schema.AddExtraResolver("SubjectWithClusterID", `clusterAdmin: Boolean!`),
	)
}

func (resolver *subjectResolver) SubjectWithClusterID(ctx context.Context) ([]*subjectWithClusterIDResolver, error) {
	if err := readK8sSubjects(ctx); err != nil {
		return nil, err
	}
	if err := readK8sRoleBindings(ctx); err != nil {
		return nil, err
	}
	clusters, err := resolver.root.ClusterDataStore.GetClusters(ctx)
	if err != nil {
		return nil, err
	}
	var resolvers []*subjectWithClusterIDResolver
	for _, cluster := range clusters {
		q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, cluster.Id).ProtoQuery()
		bindings, err := resolver.root.K8sRoleBindingStore.SearchRawRoleBindings(ctx, q)
		if err != nil {
			continue
		}

		subjectResolver, err := resolver.root.wrapSubject(k8srbac.GetSubject(resolver.Name(ctx), bindings))
		if err != nil {
			continue
		}
		resolvers = append(resolvers, wrapSubject(cluster.GetId(), subjectResolver))
	}
	return resolvers, nil
}

func (resolver *subjectWithClusterIDResolver) Name(ctx context.Context) (string, error) {
	if err := readK8sSubjects(ctx); err != nil {
		return "", err
	}

	return resolver.subject.Name(ctx), nil
}

func (resolver *subjectWithClusterIDResolver) Namespace(ctx context.Context) (string, error) {
	if err := readK8sSubjects(ctx); err != nil {
		return "", err
	}

	return resolver.subject.Namespace(ctx), nil
}

// Subjects resolves list of subjects matching a query
func (resolver *Resolver) Subjects(ctx context.Context, args rawQuery) ([]*subjectResolver, error) {
	if err := readK8sSubjects(ctx); err != nil {
		return nil, err
	}

	query, err := args.AsV1Query()
	if err != nil {
		return nil, err
	}

	bindings, err := resolver.K8sRoleBindingStore.SearchRawRoleBindings(ctx, query)
	if err != nil {
		return nil, err
	}
	// Subject return only users and groups, there is a separate resolver for service accounts.
	subjects := k8srbac.GetAllSubjects(bindings, storage.SubjectKind_USER, storage.SubjectKind_GROUP)

	filteredSubjects, err := service.GetFilteredSubjects(query, subjects)
	if err != nil {
		return nil, err
	}

	var subjectResolvers []*subjectResolver
	for _, subject := range filteredSubjects {
		subjectResolvers = append(subjectResolvers, &subjectResolver{root: resolver, data: subject})
	}

	return subjectResolvers, nil
}

func (resolver *subjectWithClusterIDResolver) Type(ctx context.Context) (string, error) {
	if err := readK8sSubjects(ctx); err != nil {
		return "", err
	}

	subject := resolver.subject.data
	switch subject.GetKind() {
	case storage.SubjectKind_USER:
		return "User", nil
	case storage.SubjectKind_GROUP:
		return "Group", nil
	default:
		return "", errors.Errorf("invalid subject type")
	}
}

func (resolver *subjectWithClusterIDResolver) Roles(ctx context.Context, args rawQuery) ([]*k8SRoleResolver, error) {
	if err := readK8sRoles(ctx); err != nil {
		return nil, err
	}
	if err := readK8sRoleBindings(ctx); err != nil {
		return nil, err
	}

	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.clusterID).ProtoQuery()
	bindings, err := resolver.subject.root.K8sRoleBindingStore.SearchRawRoleBindings(ctx, q)

	if err != nil {
		return nil, err
	}

	filterQ, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	q = search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.clusterID).ProtoQuery()
	roles, err := resolver.subject.root.K8sRoleStore.SearchRawRoles(ctx, search.NewConjunctionQuery(q, filterQ))

	if err != nil {
		return nil, err
	}

	return resolver.subject.root.wrapK8SRoles(k8srbac.NewEvaluator(roles, bindings).RolesForSubject(resolver.subject.data), nil)

}

// Permission returns which scopes do the permissions for the subject
func (resolver *subjectWithClusterIDResolver) ScopedPermissions(ctx context.Context) ([]*scopedPermissionsResolver, error) {
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

	permissionScopeMap := make(map[string]map[string]set.StringSet)
	for scope, evaluator := range evaluators {
		permissions := evaluator.ForSubject(ctx, resolver.subject.data).GetPermissionMap()
		if len(permissions) != 0 {
			permissionScopeMap[scope] = permissions
		}
	}

	return wrapPermissions(permissionScopeMap), nil
}

// ClusterAdmin returns if the service account is a cluster admin or not
func (resolver *subjectWithClusterIDResolver) ClusterAdmin(ctx context.Context) (bool, error) {
	subject := resolver.subject.data
	evaluator := resolver.getClusterEvaluator(ctx)

	return evaluator.IsClusterAdmin(ctx, subject), nil
}

func (resolver *subjectWithClusterIDResolver) getEvaluators(ctx context.Context) (map[string]k8srbac.EvaluatorForContext, error) {
	evaluators := make(map[string]k8srbac.EvaluatorForContext)
	clusterID := resolver.clusterID
	rootResolver := resolver.subject.root

	evaluators["Cluster"] =
		rbacUtils.NewClusterPermissionEvaluator(clusterID,
			rootResolver.K8sRoleStore, rootResolver.K8sRoleBindingStore)

	namespaces, err := rootResolver.Namespaces(ctx, rawQuery{nil})
	if err != nil {
		return evaluators, err
	}
	for _, namespace := range namespaces {
		namespaceName := namespace.data.GetMetadata().GetName()
		evaluators[namespaceName] = rbacUtils.NewNamespacePermissionEvaluator(clusterID,
			namespaceName, rootResolver.K8sRoleStore, rootResolver.K8sRoleBindingStore)
	}

	return evaluators, nil
}

func (resolver *subjectWithClusterIDResolver) getClusterEvaluator(ctx context.Context) k8srbac.EvaluatorForContext {
	rootResolver := resolver.subject.root
	return rbacUtils.NewClusterPermissionEvaluator(resolver.clusterID,
		rootResolver.K8sRoleStore, rootResolver.K8sRoleBindingStore)
}
