package reconcile

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	v1alpha1 "github.com/stackrox/rox/config-controller/api/v1alpha1"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/roxctl/common/environment"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type reconcileResult struct {
	created   []string
	applied   []string
	deleted   []string
	errored   []string
	dryCreate []string
	dryUpdate []string
	dryDelete []string
}

type reconciler struct {
	env         environment.Environment
	policySvc   v1.PolicyServiceClient
	notifierSvc v1.NotifierServiceClient
	clusterSvc  v1.ClustersServiceClient
	configScope string
	dryRun      bool
}

func (r *reconciler) reconcile(ctx context.Context, specs []v1alpha1.SecurityPolicySpec) (*reconcileResult, error) {
	caches, err := r.fetchReferences(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "fetching reference data from Central")
	}

	desiredByName := make(map[string]*storage.Policy, len(specs))
	for _, spec := range specs {
		proto, err := spec.ToProtobuf(caches)
		if err != nil {
			return nil, errors.Wrapf(err, "converting policy %q to protobuf", spec.PolicyName)
		}
		proto.Source = storage.PolicySource_DECLARATIVE
		proto.ConfigScope = r.configScope
		desiredByName[proto.GetName()] = proto
	}

	existingByName, err := r.fetchManagedPolicies(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "fetching managed policies from Central")
	}

	result := &reconcileResult{}

	if r.dryRun {
		r.computeDryRun(desiredByName, existingByName, result)
		return result, nil
	}

	r.applyPolicies(ctx, desiredByName, existingByName, result)
	r.deleteOrphans(ctx, desiredByName, existingByName, result)

	return result, nil
}

func (r *reconciler) fetchReferences(ctx context.Context) (map[v1alpha1.CacheType]map[string]string, error) {
	notifierMap := make(map[string]string)
	clusterMap := make(map[string]string)

	notifiers, err := r.notifierSvc.GetNotifiers(ctx, &v1.GetNotifiersRequest{})
	if err != nil {
		return nil, errors.Wrap(err, "listing notifiers")
	}
	for _, n := range notifiers.GetNotifiers() {
		notifierMap[n.GetName()] = n.GetId()
	}

	clusters, err := r.clusterSvc.GetClusters(ctx, &v1.GetClustersRequest{})
	if err != nil {
		return nil, errors.Wrap(err, "listing clusters")
	}
	for _, c := range clusters.GetClusters() {
		clusterMap[c.GetName()] = c.GetId()
	}

	return map[v1alpha1.CacheType]map[string]string{
		v1alpha1.Notifier: notifierMap,
		v1alpha1.Cluster:  clusterMap,
	}, nil
}

func (r *reconciler) fetchManagedPolicies(ctx context.Context) (map[string]*storage.Policy, error) {
	query := fmt.Sprintf("Config Scope:%s", r.configScope)
	listResp, err := r.policySvc.ListPolicies(ctx, &v1.RawQuery{Query: query})
	if err != nil {
		return nil, errors.Wrap(err, "listing policies")
	}

	existingByName := make(map[string]*storage.Policy)
	for _, lp := range listResp.GetPolicies() {
		if lp.GetSource() != storage.PolicySource_DECLARATIVE {
			continue
		}
		policy, err := r.policySvc.GetPolicy(ctx, &v1.ResourceByID{Id: lp.GetId()})
		if err != nil {
			return nil, errors.Wrapf(err, "fetching policy %q", lp.GetName())
		}
		existingByName[policy.GetName()] = policy
	}
	return existingByName, nil
}

func (r *reconciler) applyPolicies(ctx context.Context, desired, existing map[string]*storage.Policy, result *reconcileResult) {
	for name, policy := range desired {
		if existingPolicy, ok := existing[name]; ok {
			policy.Id = existingPolicy.GetId()
			if _, err := r.policySvc.PutPolicy(ctx, policy); err != nil {
				r.env.Logger().ErrfLn("Failed to update policy %q: %v", name, err)
				result.errored = append(result.errored, name)
				continue
			}
			result.applied = append(result.applied, name)
		} else {
			req := &v1.PostPolicyRequest{
				Policy:                 policy,
				EnableStrictValidation: true,
			}
			created, err := r.policySvc.PostPolicy(ctx, req)
			if err != nil {
				if isAlreadyExists(err) {
					if r.tryFallbackUpdate(ctx, policy, result) {
						continue
					}
				}
				r.env.Logger().ErrfLn("Failed to create policy %q: %v", name, err)
				result.errored = append(result.errored, name)
				continue
			}
			_ = created
			result.created = append(result.created, name)
		}
	}
}

// tryFallbackUpdate handles the crash-recovery case: PostPolicy failed with name conflict
// because the policy was already created in a previous run that didn't complete.
func (r *reconciler) tryFallbackUpdate(ctx context.Context, policy *storage.Policy, result *reconcileResult) bool {
	existing, err := r.findPolicyByName(ctx, policy.GetName())
	if err != nil || existing == nil {
		return false
	}
	if existing.GetSource() != storage.PolicySource_DECLARATIVE || existing.GetConfigScope() != r.configScope {
		return false
	}
	policy.Id = existing.GetId()
	if _, err := r.policySvc.PutPolicy(ctx, policy); err != nil {
		return false
	}
	result.applied = append(result.applied, policy.GetName())
	return true
}

func (r *reconciler) findPolicyByName(ctx context.Context, name string) (*storage.Policy, error) {
	listResp, err := r.policySvc.ListPolicies(ctx, &v1.RawQuery{
		Query: fmt.Sprintf("Policy:%s", name),
	})
	if err != nil {
		return nil, err
	}
	for _, lp := range listResp.GetPolicies() {
		if lp.GetName() == name {
			return r.policySvc.GetPolicy(ctx, &v1.ResourceByID{Id: lp.GetId()})
		}
	}
	return nil, nil
}

func (r *reconciler) deleteOrphans(ctx context.Context, desired, existing map[string]*storage.Policy, result *reconcileResult) {
	for name, policy := range existing {
		if _, ok := desired[name]; ok {
			continue
		}
		if _, err := r.policySvc.DeletePolicy(ctx, &v1.ResourceByID{Id: policy.GetId()}); err != nil {
			r.env.Logger().ErrfLn("Failed to delete orphaned policy %q: %v", name, err)
			result.errored = append(result.errored, name)
			continue
		}
		result.deleted = append(result.deleted, name)
	}
}

func (r *reconciler) computeDryRun(desired, existing map[string]*storage.Policy, result *reconcileResult) {
	for name := range desired {
		if _, ok := existing[name]; ok {
			result.dryUpdate = append(result.dryUpdate, name)
		} else {
			result.dryCreate = append(result.dryCreate, name)
		}
	}
	for name := range existing {
		if _, ok := desired[name]; !ok {
			result.dryDelete = append(result.dryDelete, name)
		}
	}
}

func isAlreadyExists(err error) bool {
	if s, ok := status.FromError(err); ok {
		return s.Code() == codes.AlreadyExists
	}
	return false
}
