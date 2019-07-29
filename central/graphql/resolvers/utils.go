package resolvers

import (
	"context"
	"sort"

	complianceStandards "github.com/stackrox/rox/central/compliance/standards"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/k8srbac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
)

// StringListEntryResolver represents a set of values keyed by a string
type stringListEntryResolver struct {
	key    string
	values set.StringSet
}

// ScopedPermissionsResolver represents the scoped permissions of a subject/service account
type scopedPermissionsResolver struct {
	scope       string
	permissions []*stringListEntryResolver
}

// Key represents the key value of the string list entry
func (resolver *stringListEntryResolver) Key(ctx context.Context) string {
	return resolver.key
}

// Values represents the set of values of the string list entry
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

// Scope represents the scope of the permissions - cluster wide or the namespace name to which the permissions are scoped
func (resolver *scopedPermissionsResolver) Scope(ctx context.Context) string {
	return resolver.scope
}

// Permissions represents the verbs and the resources to which those verbs are granted
func (resolver *scopedPermissionsResolver) Permissions(ctx context.Context) []*stringListEntryResolver {
	return resolver.permissions
}

// WrapPermissions wraps the input into a scopedPermissionsResolver
func wrapPermissions(values map[string]map[string]set.StringSet) []*scopedPermissionsResolver {
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

type subjectWithClusterIDResolver struct {
	clusterID string
	subject   *subjectResolver
}

func (resolver *subjectWithClusterIDResolver) ClusterID(ctx context.Context) string {
	return resolver.clusterID
}

func (resolver *subjectWithClusterIDResolver) Subject(ctx context.Context) *subjectResolver {
	return resolver.subject
}

func wrapSubjects(clusterID string, subjects []*subjectResolver) []*subjectWithClusterIDResolver {
	if len(subjects) == 0 {
		return nil
	}

	output := make([]*subjectWithClusterIDResolver, 0, len(subjects))
	for _, s := range subjects {
		output = append(output, &subjectWithClusterIDResolver{clusterID, s})
	}

	return output
}

func wrapSubject(clusterID string, subject *subjectResolver) *subjectWithClusterIDResolver {
	return &subjectWithClusterIDResolver{clusterID, subject}
}

func getStandardIDs(ctx context.Context, cs complianceStandards.Repository) ([]string, error) {
	if err := readCompliance(ctx); err != nil {
		return nil, err
	}
	standards, err := cs.Standards()
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(standards))
	for _, s := range standards {
		result = append(result, s.GetId())
	}
	return result, nil
}

func (resolver *clusterResolver) getRoleBindings(ctx context.Context, args rawQuery) ([]*storage.K8SRoleBinding, error) {
	if err := readK8sRoleBindings(ctx); err != nil {
		return nil, err
	}
	q, err := resolver.getConjunctionQuery(args)
	if err != nil {
		return nil, err
	}
	bindings, err := resolver.root.K8sRoleBindingStore.SearchRawRoleBindings(ctx, q)
	if err != nil {
		return nil, err
	}
	return bindings, nil
}

func (resolver *clusterResolver) getSubjects(ctx context.Context, args rawQuery) ([]*storage.Subject, error) {
	if err := readK8sSubjects(ctx); err != nil {
		return nil, err
	}
	bindings, err := resolver.getRoleBindings(ctx, args)
	if err != nil {
		return nil, err
	}
	subjects := k8srbac.GetAllSubjects(bindings, storage.SubjectKind_USER, storage.SubjectKind_GROUP)
	return subjects, nil
}

func (resolver *clusterResolver) getQuery() *v1.Query {
	return search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetId()).ProtoQuery()
}

func (resolver *clusterResolver) getConjunctionQuery(args rawQuery) (*v1.Query, error) {
	q1 := resolver.getQuery()
	if args.String() == "" {
		return q1, nil
	}
	q2, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}
	return search.NewConjunctionQuery(q2, q1), nil
}

// SubjectCount returns the count of Subjects which have any permission on this namespace or the cluster it belongs to
func (resolver *namespaceResolver) getSubjects(ctx context.Context, baseQuery *v1.Query) ([]*storage.Subject, error) {
	if err := readK8sSubjects(ctx); err != nil {
		return nil, err
	}
	if err := readK8sRoleBindings(ctx); err != nil {
		return nil, err
	}
	q := search.NewConjunctionQuery(
		search.NewDisjunctionQuery(
			search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetMetadata().GetClusterId()).
				AddExactMatches(search.Namespace, resolver.data.GetMetadata().GetName()).ProtoQuery(),
			search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetMetadata().GetClusterId()).
				AddBools(search.ClusterRole, true).ProtoQuery()),
		baseQuery)
	bindings, err := resolver.root.K8sRoleBindingStore.SearchRawRoleBindings(ctx, q)
	if err != nil {
		return nil, err
	}
	subjects := k8srbac.GetAllSubjects(bindings, storage.SubjectKind_USER, storage.SubjectKind_GROUP)
	return subjects, nil
}
