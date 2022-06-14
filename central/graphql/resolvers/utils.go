package resolvers

import (
	"context"
	"reflect"
	"sort"
	"strings"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	complianceStandards "github.com/stackrox/stackrox/central/compliance/standards"
	"github.com/stackrox/stackrox/central/graphql/resolvers/inputtypes"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/k8srbac"
	"github.com/stackrox/stackrox/pkg/pointers"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/set"
)

// idField map holds id search field label for corresponding search category
var (
	idField = map[v1.SearchCategory]search.FieldLabel{
		v1.SearchCategory_CLUSTERS:    search.ClusterID,
		v1.SearchCategory_NAMESPACES:  search.NamespaceID,
		v1.SearchCategory_DEPLOYMENTS: search.DeploymentID,
		v1.SearchCategory_POLICIES:    search.PolicyID,
	}
)

// IDQuery is a wrapper around a graphql.ID
type IDQuery struct {
	ID *graphql.ID
}

// stringListEntryResolver represents a set of values keyed by a string
type stringListEntryResolver struct {
	key    string
	values set.StringSet
}

// scopedPermissionsResolver represents the scoped permissions of a subject/service account
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

// wrapPermissions wraps the input into a scopedPermissionsResolver
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

	q, err := resolver.getClusterConjunctionQuery(q)
	if err != nil {
		return nil, err
	}

	bindings, err := resolver.root.K8sRoleBindingStore.SearchRawRoleBindings(ctx, q)
	if err != nil {
		return nil, err
	}
	return bindings, nil
}

func (resolver *clusterResolver) getSubjects(ctx context.Context, q *v1.Query) ([]*storage.Subject, error) {
	if err := readK8sSubjects(ctx); err != nil {
		return nil, err
	}

	baseQ := search.NewQueryBuilder().
		AddExactMatches(search.ClusterID, resolver.data.GetId()).
		AddExactMatches(search.SubjectKind, storage.SubjectKind_USER.String(), storage.SubjectKind_GROUP.String()).
		ProtoQuery()

	bindings, err := resolver.getRoleBindings(ctx, search.ConjunctionQuery(baseQ, q))
	if err != nil {
		return nil, err
	}
	subjects := k8srbac.GetAllSubjects(bindings, storage.SubjectKind_USER, storage.SubjectKind_GROUP)
	return subjects, nil
}

// SubjectCount returns the count of Subjects which have any permission on this namespace or the cluster it belongs to
func (resolver *namespaceResolver) getSubjects(ctx context.Context, baseQuery *v1.Query) ([]*storage.Subject, error) {
	if err := readK8sSubjects(ctx); err != nil {
		return nil, err
	}
	if err := readK8sRoleBindings(ctx); err != nil {
		return nil, err
	}

	q := search.ConjunctionQuery(
		search.NewQueryBuilder().AddExactMatches(search.ClusterID, resolver.data.GetMetadata().GetClusterId()).ProtoQuery(),
		search.DisjunctionQuery(
			search.NewQueryBuilder().AddExactMatches(search.Namespace, resolver.data.GetMetadata().GetName()).ProtoQuery(),
			search.NewQueryBuilder().AddBools(search.ClusterRole, true).ProtoQuery()),
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

func getControlStatusFromAggregationResult(result *storage.ComplianceAggregation_Result) string {
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

func getComplianceControlNodeCountFromAggregationResults(results []*storage.ComplianceAggregation_Result) *complianceControlNodeCountResolver {
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

// K8sCVEInfoResolver holds CVE and fixable count for a cluster
type K8sCVEInfoResolver struct {
	cveIDs        []string
	fixableCveIDs []string
}

// CveIDs returns IDs of CVEs that affect this cluster
func (resolver *K8sCVEInfoResolver) CveIDs() []string {
	if resolver == nil {
		return []string{}
	}
	return resolver.cveIDs
}

// FixableCveIDs returns IDs of fixable CVEs that affect this cluster
func (resolver *K8sCVEInfoResolver) FixableCveIDs() []string {
	if resolver == nil {
		return []string{}
	}
	return resolver.fixableCveIDs
}

// ClusterWithK8sCVEInfoResolver holds a cluster with its K8s info
type ClusterWithK8sCVEInfoResolver struct {
	cluster    *clusterResolver
	k8sCVEInfo *K8sCVEInfoResolver
}

// Cluster returns cluster on a ClusterWithK8sCVEInfoResolver
func (resolver *ClusterWithK8sCVEInfoResolver) Cluster() *clusterResolver {
	if resolver == nil {
		return nil
	}
	return resolver.cluster
}

// K8sCVEInfo returns k8sCVEInfo on ClusterWithK8sCVEInfoResolver
func (resolver *ClusterWithK8sCVEInfoResolver) K8sCVEInfo() *K8sCVEInfoResolver {
	if resolver == nil {
		return nil
	}
	return resolver.k8sCVEInfo
}

type paginationWrapper struct {
	pv *v1.QueryPagination
}

func (pw paginationWrapper) paginate(datSlice interface{}, err error) (interface{}, error) {
	if err != nil || pw.pv == nil {
		return datSlice, err
	}

	datType := reflect.TypeOf(datSlice)
	if datType.Kind() != reflect.Slice {
		return datSlice, errors.New("not a slice")
	}

	datValue := reflect.ValueOf(datSlice)
	if datValue.Len() == 0 {
		return datSlice, nil
	}

	offset := int(pw.pv.GetOffset())
	limit := int(pw.pv.GetLimit())
	if offset < 0 {
		offset = 0
	}
	if limit < 0 {
		limit = 0
	}

	remnants := datValue.Len() - offset
	if remnants <= 0 {
		return reflect.Zero(datType).Interface(), nil
	}

	var end int
	if limit == 0 || remnants < limit {
		end = offset + remnants
	} else {
		end = offset + limit
	}
	return datValue.Slice(offset, end).Interface(), nil
}

func getImageIDFromIfImageShaQuery(ctx context.Context, resolver *Resolver, args RawQuery) (string, error) {
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return "", err
	}

	query, filtered := search.FilterQuery(query, func(bq *v1.BaseQuery) bool {
		matchFieldQuery, ok := bq.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
		if ok {
			if strings.EqualFold(matchFieldQuery.MatchFieldQuery.GetField(), search.ImageSHA.String()) {
				return true
			}
		}
		return false
	})

	if !filtered || query == search.EmptyQuery() {
		return "", nil
	}

	res, err := resolver.ImageDataStore.Search(ctx, query)
	if err != nil {
		return "", err
	}
	if len(res) != 1 {
		return "", errors.Errorf(
			"received %d images in query response when 1 image was expected. Please check the query",
			len(res))
	}

	return res[0].ID, nil
}

// V1RawQueryAsResolverQuery parse v1.RawQuery into inputtypes.RawQuery and inputtypes.Pagination queries used by graphQL resolvers.
func V1RawQueryAsResolverQuery(rQ *v1.RawQuery) (RawQuery, PaginatedQuery) {
	if rQ.GetPagination() == nil {
		return RawQuery{pointers.String(rQ.GetQuery()), nil}, PaginatedQuery{Query: pointers.String(rQ.GetQuery())}
	}

	return RawQuery{pointers.String(rQ.GetQuery()), nil}, PaginatedQuery{
		Query: pointers.String(rQ.GetQuery()),
		Pagination: &inputtypes.Pagination{
			Limit:  pointers.Int32(rQ.GetPagination().GetLimit()),
			Offset: pointers.Int32(rQ.GetPagination().GetOffset()),
			SortOption: &inputtypes.SortOption{
				Field:    pointers.String(rQ.GetPagination().GetSortOption().GetField()),
				Reversed: pointers.Bool(rQ.GetPagination().GetSortOption().GetReversed()),
			},
		},
	}
}
