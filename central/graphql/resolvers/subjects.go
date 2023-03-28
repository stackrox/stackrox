package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	rbacUtils "github.com/stackrox/rox/central/rbac/utils"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/k8srbac"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddQuery("subject(id: ID): Subject"),
		schema.AddQuery("subjects(query: String, pagination: Pagination): [Subject!]!"),
		schema.AddQuery("subjectCount(query: String): Int!"),
		schema.AddExtraResolver("Subject", `type: String!`),
		schema.AddExtraResolver("Subject", `k8sRoleCount(query: String): Int!`),
		schema.AddExtraResolver("Subject", `k8sRoles(query: String, pagination: Pagination): [K8SRole!]!`),
		schema.AddExtraResolver("Subject", `scopedPermissions: [ScopedPermissions!]!`),
		schema.AddExtraResolver("Subject", `clusterAdmin: Boolean!`),
	)
}

// Subject returns a GraphQL resolver for a given id
func (resolver *Resolver) Subject(ctx context.Context, args struct{ *graphql.ID }) (*subjectResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "Subject")
	if err := readK8sSubjects(ctx); err != nil {
		return nil, err
	}
	if args.ID == nil {
		return nil, errors.New("id required to lookup subject")
	}
	clusterID, subjectName, err := k8srbac.SplitSubjectID(string(*args.ID))
	if err != nil {
		return nil, err
	}
	bindings, err := resolver.K8sRoleBindingStore.SearchRawRoleBindings(ctx, search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery())
	if err != nil {
		return nil, err
	}
	return resolver.wrapSubject(k8srbac.GetSubject(subjectName, bindings))
}

// Subjects resolves list of subjects matching a query
func (resolver *Resolver) Subjects(ctx context.Context, args PaginatedQuery) ([]*subjectResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Subjects, "Subjects")

	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	filteredSubjects, err := resolver.getFilteredSubjects(ctx, query)
	if err != nil {
		return nil, err
	}

	var subjectResolvers []*subjectResolver
	for _, subject := range filteredSubjects {
		subjectResolvers = append(subjectResolvers, &subjectResolver{root: resolver, data: subject})
	}

	return paginate(query.Pagination, subjectResolvers, nil)
}

// SubjectCount returns count of all subjects across infrastructure
func (resolver *Resolver) SubjectCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "SubjectCount")

	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}

	filteredSubjects, err := resolver.getFilteredSubjects(ctx, query)
	if err != nil {
		return 0, err
	}

	return int32(len(filteredSubjects)), nil
}

func (resolver *Resolver) getFilteredSubjects(ctx context.Context, query *v1.Query) ([]*storage.Subject, error) {
	if err := readK8sSubjects(ctx); err != nil {
		return nil, err
	}

	// Subject return only users and groups, there is a separate resolver for service accounts.
	subjectKindQ :=
		search.NewQueryBuilder().AddExactMatches(search.SubjectKind, storage.SubjectKind_USER.String(), storage.SubjectKind_GROUP.String())
	q := search.ConjunctionQuery(subjectKindQ.ProtoQuery(), query)

	bindings, err := resolver.K8sRoleBindingStore.SearchRawRoleBindings(ctx, q)
	if err != nil {
		return nil, err
	}

	// Since the query already just gets users and group, this is effectively just a way to ensure only unique roles are returned
	return k8srbac.GetAllSubjects(bindings, storage.SubjectKind_USER, storage.SubjectKind_GROUP), nil
}

func (resolver *subjectResolver) Type(ctx context.Context) (string, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Subjects, "Type")
	if err := readK8sSubjects(ctx); err != nil {
		return "", err
	}

	subject := resolver.data
	switch subject.GetKind() {
	case storage.SubjectKind_USER:
		return "User", nil
	case storage.SubjectKind_GROUP:
		return "Group", nil
	default:
		return "", errors.New("invalid subject type")
	}
}

func (resolver *subjectResolver) K8sRoleCount(ctx context.Context, args RawQuery) (int32, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Subjects, "RoleCount")
	if err := readK8sRoles(ctx); err != nil {
		return 0, err
	}
	if err := readK8sRoleBindings(ctx); err != nil {
		return 0, err
	}

	filterQ, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return 0, err
	}

	roles, err := resolver.getRolesForSubject(ctx, filterQ)
	if err != nil {
		return 0, err
	}
	return int32(len(roles)), nil
}

func (resolver *subjectResolver) K8sRoles(ctx context.Context, args PaginatedQuery) ([]*k8SRoleResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Subjects, "Roles")
	if err := readK8sRoles(ctx); err != nil {
		return nil, err
	}
	if err := readK8sRoleBindings(ctx); err != nil {
		return nil, err
	}

	filterQ, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	roles, err := resolver.getRolesForSubject(ctx, filterQ)
	if err != nil {
		return nil, err
	}

	roleResolvers, err := resolver.root.wrapK8SRoles(roles, nil)
	if err != nil {
		return nil, err
	}
	return paginate(filterQ.Pagination, roleResolvers, nil)
}

func (resolver *subjectResolver) getRolesForSubject(ctx context.Context, filterQ *v1.Query) ([]*storage.K8SRole, error) {
	q := search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetClusterId()).
		AddExactMatches(search.SubjectName, resolver.data.GetName()).
		AddExactMatches(search.SubjectKind, resolver.data.GetKind().String()).
		ProtoQuery()

	bindings, err := resolver.root.K8sRoleBindingStore.SearchRawRoleBindings(ctx, q)
	if err != nil {
		return nil, err
	}

	q = search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetClusterId()).ProtoQuery()
	roles, err := resolver.root.K8sRoleStore.SearchRawRoles(ctx, search.ConjunctionQuery(q, filterQ))
	if err != nil {
		return nil, err
	}
	return k8srbac.NewEvaluator(roles, bindings).RolesForSubject(resolver.data), nil
}

// Permission returns which scopes do the permissions for the subject
func (resolver *subjectResolver) ScopedPermissions(ctx context.Context) ([]*scopedPermissionsResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Subjects, "ScopedPermissions")
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
		permissions := evaluator.ForSubject(ctx, resolver.data).GetPermissionMap()
		if len(permissions) != 0 {
			permissionScopeMap[scope] = permissions
		}
	}

	return wrapPermissions(permissionScopeMap), nil
}

// ClusterAdmin returns if the service account is a cluster admin or not
func (resolver *subjectResolver) ClusterAdmin(ctx context.Context) (bool, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Subjects, "ClusterAdmin")
	subject := resolver.data
	evaluator := resolver.getClusterEvaluator(ctx)

	return evaluator.IsClusterAdmin(ctx, subject), nil
}

func (resolver *subjectResolver) getEvaluators(ctx context.Context) (map[string]k8srbac.EvaluatorForContext, error) {
	evaluators := make(map[string]k8srbac.EvaluatorForContext)
	clusterID := resolver.data.GetClusterId()
	rootResolver := resolver.root

	evaluators["Cluster"] =
		rbacUtils.NewClusterPermissionEvaluator(clusterID,
			rootResolver.K8sRoleStore, rootResolver.K8sRoleBindingStore)

	namespaces, err := rootResolver.Namespaces(ctx, PaginatedQuery{})
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

func (resolver *subjectResolver) getClusterEvaluator(_ context.Context) k8srbac.EvaluatorForContext {
	rootResolver := resolver.root
	return rbacUtils.NewClusterPermissionEvaluator(resolver.data.GetClusterId(),
		rootResolver.K8sRoleStore, rootResolver.K8sRoleBindingStore)
}
