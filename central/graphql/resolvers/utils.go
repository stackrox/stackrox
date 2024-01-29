package resolvers

import (
	"context"
	"sort"
	"strings"

	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	complianceStandards "github.com/stackrox/rox/central/compliance/standards"
	"github.com/stackrox/rox/central/graphql/resolvers/inputtypes"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authz/interceptor"
	"github.com/stackrox/rox/pkg/k8srbac"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
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
func (resolver *stringListEntryResolver) Key(_ context.Context) string {
	return resolver.key
}

// Values represents the set of values of the string list entry
func (resolver *stringListEntryResolver) Values(_ context.Context) []string {
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
func (resolver *scopedPermissionsResolver) Scope(_ context.Context) string {
	return resolver.scope
}

// Permissions represents the verbs and the resources to which those verbs are granted
func (resolver *scopedPermissionsResolver) Permissions(_ context.Context) []*stringListEntryResolver {
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

func paginate[T any](pv *v1.QueryPagination, slice []T, err error) ([]T, error) {
	if err != nil || pv == nil {
		return slice, err
	}
	if len(slice) == 0 {
		return slice, nil
	}

	offset := int(pv.GetOffset())
	limit := int(pv.GetLimit())
	if offset < 0 {
		offset = 0
	}
	if limit < 0 {
		limit = 0
	}

	remnants := len(slice) - offset
	if remnants <= 0 {
		return nil, nil
	}

	var end int
	if limit == 0 || remnants < limit {
		end = offset + remnants
	} else {
		end = offset + limit
	}
	return slice[offset:end], nil
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

// logErrorOnQueryContainingField logs error if the query contains the given field label.
func logErrorOnQueryContainingField(query *v1.Query, label search.FieldLabel, resolver string) {
	search.ApplyFnToAllBaseQueries(query, func(bq *v1.BaseQuery) {
		mfQ, ok := bq.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
		if ok && mfQ.MatchFieldQuery.GetField() == label.String() {
			log.Errorf("Unexpected field (%s) found in query to resolver (%s). Response maybe unexpected.", label.String(), resolver)
		}
	})
}

// FilterFieldFromRawQuery removes the given field from RawQuery
func FilterFieldFromRawQuery(rq RawQuery, label search.FieldLabel) RawQuery {
	return RawQuery{
		Query: pointers.String(search.FilterFields(rq.String(), func(field string) bool {
			return label.String() != field
		})),
		ScopeQuery: rq.ScopeQuery,
	}
}

// processWithAuditLog runs handler and logs to the audit log pipeline (assuming there is a notifier setup for audit logging).
// It logs details of the request and if there was an error. processWithAuditLog will return the response and error directly
// from handler. You may need to cast it back to your desired type.
// This is required because currently audit logs are only automatically added for GRPC calls and not GraphQL.
// However, mutating calls should also log. This is a workaround for this limitation.
func (resolver *Resolver) processWithAuditLog(ctx context.Context, req interface{}, method string, handler func() (interface{}, error)) (interface{}, error) {
	resp, err := handler()
	if resolver.AuditLogger != nil {
		go resolver.AuditLogger.SendAuditMessage(ctx, req, method, interceptor.AuthStatus{}, err)
	}
	return resp, err
}

func getImageIDFromQuery(q *v1.Query) string {
	if q == nil {
		return ""
	}
	var imageID string
	search.ApplyFnToAllBaseQueries(q, func(bq *v1.BaseQuery) {
		matchFieldQuery, ok := bq.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
		if !ok {
			return
		}
		if strings.EqualFold(matchFieldQuery.MatchFieldQuery.GetField(), search.ImageSHA.String()) {
			imageID = matchFieldQuery.MatchFieldQuery.Value
			imageID = strings.TrimRight(imageID, `"`)
			imageID = strings.TrimLeft(imageID, `"`)
		}
	})
	return imageID
}
