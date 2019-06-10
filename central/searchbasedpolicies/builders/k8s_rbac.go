package builders

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	roleDataStore "github.com/stackrox/rox/central/rbac/k8srole/datastore"
	bindingDataStore "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/searchbasedpolicies"
	serviceAccountDataStore "github.com/stackrox/rox/central/serviceaccount/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/k8srbac"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
)

var (
	rbacReadingCtx = sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
		sac.ResourceScopeKeys(resources.Cluster, resources.K8sRole, resources.K8sRoleBinding, resources.ServiceAccount)))
)

// K8sRBACQueryBuilder builds queries for K8s RBAC permission level.
type K8sRBACQueryBuilder struct {
	Clusters        clusterDataStore.DataStore
	K8sRoles        roleDataStore.DataStore
	K8sBindings     bindingDataStore.DataStore
	ServiceAccounts serviceAccountDataStore.DataStore
}

// Name implements the PolicyQueryBuilder interface.
func (p K8sRBACQueryBuilder) Name() string {
	return "query builder for k8s rbac permissions"
}

// Query implements the PolicyQueryBuilder interface.
func (p K8sRBACQueryBuilder) Query(fields *storage.PolicyFields, _ map[search.FieldLabel]*v1.SearchField) (q *v1.Query, v searchbasedpolicies.ViolationPrinter, err error) {
	// Check that a permission level is set in the policy.
	if fields.GetPermissionPolicy().GetPermissionLevel() == storage.PermissionLevel_UNSET {
		return
	}
	maxPermissionAllowed := fields.GetPermissionPolicy().GetPermissionLevel()

	// Generate a query for deployments using any of those service accounts.
	q, err = p.allClustersQuery(rbacReadingCtx, maxPermissionAllowed)
	if err != nil {
		err = errors.Wrap(err, p.Name())
		return
	}

	v = func(ctx context.Context, result search.Result) searchbasedpolicies.Violations {
		violations := searchbasedpolicies.Violations{
			AlertViolations: []*storage.Alert_Violation{
				{
					Message: fmt.Sprintf("Deployment uses a service account with permissions greater than %s", maxPermissionAllowed),
				},
			},
		}
		return violations
	}
	return
}

// Create a query that matches the deployments with privileges above the threshold in each cluster and combine then in a
// disjunction.
func (p K8sRBACQueryBuilder) allClustersQuery(ctx context.Context, maxPermissionAllowed storage.PermissionLevel) (q *v1.Query, err error) {
	// Get all clusters.
	clusters, err := p.Clusters.GetClusters(ctx)
	if err != nil {
		return nil, err
	}

	// Generate a query for each cluster.
	clusterQueries := make([]*v1.Query, 0, len(clusters))
	for _, cluster := range clusters {
		clusterQuery, err := p.clusterQuery(ctx, cluster.GetId(), maxPermissionAllowed)
		if err != nil {
			return nil, err
		}
		if clusterQuery == nil {
			continue // skip the cluster since no deployments in it can violate the policy.
		}
		clusterQueries = append(clusterQueries, clusterQuery)
	}
	if len(clusterQueries) == 0 {
		return nil, nil
	} else if len(clusterQueries) == 1 {
		return clusterQueries[0], nil
	}
	// Combine all of the queries in a disjunction, since a deployment only needs to match a single cluster's query.
	return search.NewDisjunctionQuery(clusterQueries...), nil
}

// Create a query that matches the deployments with privileges above the threshold in a specific cluster.
func (p K8sRBACQueryBuilder) clusterQuery(ctx context.Context, clusterID string, maxPermissionAllowed storage.PermissionLevel) (q *v1.Query, err error) {
	isInCluster := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery()

	// Fetch roles, bindings, and service accounts to form the query for the cluster.
	roles, err := p.K8sRoles.SearchRawRoles(ctx, isInCluster)
	if err != nil {
		return nil, err
	}
	bindings, err := p.K8sBindings.SearchRawRoleBindings(ctx, isInCluster)
	if err != nil {
		return nil, err
	}
	serviceAccounts, err := p.ServiceAccounts.SearchRawServiceAccounts(ctx, isInCluster)
	if err != nil {
		return nil, err
	}

	// Find all of the service accounts that have permissions above the bucket level specified.
	bucketEval := newBucketEvaluator(roles, bindings)
	var subjectsThatViolateBucket []*storage.Subject
	for _, subject := range collectServiceAccounts(serviceAccounts) {
		if bucketEval.getBucket(subject) > maxPermissionAllowed {
			subjectsThatViolateBucket = append(subjectsThatViolateBucket, subject)
		}
	}
	if len(subjectsThatViolateBucket) == 0 {
		return // Deployments are UNABLE to violate the policy.
	}

	// Generate a query for deployments using any of those service accounts.
	return clusterSubjectsToQuery(clusterID, subjectsThatViolateBucket), nil
}

// Convert list of subjects to a query.
///////////////////////////////////////
func clusterSubjectsToQuery(clusterID string, subjects []*storage.Subject) *v1.Query {
	return search.NewConjunctionQuery(
		search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery(),
		subjectsToQuery(subjects),
	)
}

func subjectsToQuery(subjects []*storage.Subject) *v1.Query {
	if len(subjects) == 1 {
		return singleSubjectToQuery(subjects[0])
	}
	return multiSubjectsToQuery(subjects)
}

func singleSubjectToQuery(subject *storage.Subject) *v1.Query {
	return search.NewConjunctionQuery(
		search.NewQueryBuilder().AddExactMatches(search.ServiceAccountName, subject.GetName()).ProtoQuery(),
		search.NewQueryBuilder().AddExactMatches(search.Namespace, subject.GetNamespace()).ProtoQuery(),
	)
}

func multiSubjectsToQuery(subjects []*storage.Subject) *v1.Query {
	queries := make([]*v1.Query, 0, len(subjects))
	for _, subject := range subjects {
		queries = append(queries, singleSubjectToQuery(subject))
	}
	return search.NewDisjunctionQuery(queries...)
}

// Convert a list of service accounts into a list of subjects.
//////////////////////////////////////////////////////////////
func collectServiceAccounts(serviceAccounts []*storage.ServiceAccount) []*storage.Subject {
	allServiceAccounts := k8srbac.NewSubjectSet()
	// Add explicitly labeled from bindings.
	for _, sa := range serviceAccounts {
		allServiceAccounts.Add(&storage.Subject{
			Kind:      storage.SubjectKind_SERVICE_ACCOUNT,
			Name:      sa.GetName(),
			Namespace: sa.GetNamespace(),
		})
	}
	return allServiceAccounts.ToSlice()
}

// Evaluate the permission bucket for a subject.
////////////////////////////////////////////////
type bucketEvaluator struct {
	clusterEvaluator    k8srbac.Evaluator
	namespaceEvaluators map[string]k8srbac.Evaluator
}

func newBucketEvaluator(roles []*storage.K8SRole, bindings []*storage.K8SRoleBinding) *bucketEvaluator {
	return &bucketEvaluator{
		clusterEvaluator:    k8srbac.MakeClusterEvaluator(roles, bindings),
		namespaceEvaluators: k8srbac.MakeNamespaceEvaluators(roles, bindings),
	}
}

func (be *bucketEvaluator) getBucket(subject *storage.Subject) storage.PermissionLevel {
	// Check for admin or elevated permissions cluster wide.
	clusterPermissions := be.clusterEvaluator.ForSubject(subject)
	if clusterPermissions.Grants(k8srbac.EffectiveAdmin) {
		return storage.PermissionLevel_CLUSTER_ADMIN
	}
	if k8srbac.CanWriteAResource(clusterPermissions) || k8srbac.CanReadAResource(clusterPermissions) {
		return storage.PermissionLevel_ELEVATED_CLUSTER_WIDE
	}

	// Check for elevated or default permissions within a namespace.
	var maxPermissions storage.PermissionLevel
	for _, namespaceEvaluator := range be.namespaceEvaluators {
		if namespaceEvaluator == nil {
			continue
		}
		namespacePermissions := namespaceEvaluator.ForSubject(subject)
		if k8srbac.CanWriteAResource(namespacePermissions) || namespacePermissions.Grants(k8srbac.ListAnything) {
			return storage.PermissionLevel_ELEVATED_IN_NAMESPACE
		} else if k8srbac.CanReadAResource(namespacePermissions) {
			maxPermissions = storage.PermissionLevel_DEFAULT
		}
	}
	if maxPermissions == storage.PermissionLevel_UNSET {
		return storage.PermissionLevel_NONE
	}
	return maxPermissions
}
