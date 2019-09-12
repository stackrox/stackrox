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
	clusterID   string
	clusterName string
	subject     *subjectResolver
}

func (resolver *subjectWithClusterIDResolver) ClusterID(ctx context.Context) string {
	return resolver.clusterID
}

func (resolver *subjectWithClusterIDResolver) Subject(ctx context.Context) *subjectResolver {
	return resolver.subject
}

func wrapSubjects(clusterID string, clusterName string, subjects []*subjectResolver) []*subjectWithClusterIDResolver {
	if len(subjects) == 0 {
		return nil
	}

	output := make([]*subjectWithClusterIDResolver, 0, len(subjects))
	for _, s := range subjects {
		output = append(output, &subjectWithClusterIDResolver{clusterID, clusterName, s})
	}

	return output
}

func wrapSubject(clusterID string, clusterName string, subject *subjectResolver) *subjectWithClusterIDResolver {
	return &subjectWithClusterIDResolver{clusterID, clusterName, subject}
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

func (resolver *clusterResolver) getRoleBindings(ctx context.Context, q *v1.Query) ([]*storage.K8SRoleBinding, error) {
	if err := readK8sRoleBindings(ctx); err != nil {
		return nil, err
	}

	bindings, err := resolver.root.K8sRoleBindingStore.SearchRawRoleBindings(ctx, resolver.getConjunctionQuery(q))
	if err != nil {
		return nil, err
	}
	return bindings, nil
}

func (resolver *clusterResolver) getSubjects(ctx context.Context, q *v1.Query) ([]*storage.Subject, error) {
	if err := readK8sSubjects(ctx); err != nil {
		return nil, err
	}

	bindings, err := resolver.getRoleBindings(ctx, q)
	if err != nil {
		return nil, err
	}
	subjects := k8srbac.GetAllSubjects(bindings, storage.SubjectKind_USER, storage.SubjectKind_GROUP)
	return subjects, nil
}

func (resolver *clusterResolver) getQuery() *v1.Query {
	return search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetId()).ProtoQuery()
}

func (resolver *clusterResolver) getConjunctionQuery(q *v1.Query) *v1.Query {
	q1 := resolver.getQuery()
	if q == search.EmptyQuery() {
		return q1
	}
	return search.NewConjunctionQuery(q, q1)
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

func (resolver *complianceControlResolver) getClusterIDs(ctx context.Context) ([]string, error) {
	clusters, err := resolver.root.ClusterDataStore.GetClusters(ctx)
	if err != nil {
		return nil, err
	}
	var clusterIDs []string
	for _, cluster := range clusters {
		clusterIDs = append(clusterIDs, cluster.GetId())
	}
	return clusterIDs, nil
}

// ControlStatus can be pass/fail, or N/A is neither passing or failing
type ControlStatus int32

const (
	fail ControlStatus = iota
	pass
	na
)

func (c ControlStatus) String() string {
	return []string{"FAIL", "PASS", "N/A"}[c]
}

func getControlStatusFromAggregationResult(result *v1.ComplianceAggregation_Result) string {
	return getControlStatus(result.GetNumFailing(), result.GetNumPassing())
}

func getControlStatus(failing, passing int32) string {
	var cs ControlStatus
	if passing == 0 && failing == 0 {
		cs = na
	} else if failing != 0 {
		cs = fail
	} else {
		cs = pass
	}
	return cs.String()
}

func getComplianceControlNodeCountFromAggregationResults(results []*v1.ComplianceAggregation_Result) *complianceControlNodeCountResolver {
	ret := &complianceControlNodeCountResolver{}
	for _, r := range results {
		if r.GetNumFailing() != 0 {
			ret.failingCount++
		} else if r.GetNumPassing() != 0 {
			ret.passingCount++
		} else {
			ret.unknownCount++
		}
	}
	return ret
}
