package datastore

import (
	"context"
	"errors"

	errorsPkg "github.com/pkg/errors"
	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/policy/index"
	"github.com/stackrox/rox/central/policy/search"
	"github.com/stackrox/rox/central/policy/store"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	policiesPkg "github.com/stackrox/rox/pkg/policies"
	"github.com/stackrox/rox/pkg/sac"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	policySAC = sac.ForResource(resources.Policy)
)

type datastoreImpl struct {
	storage     store.Store
	indexer     index.Indexer
	searcher    search.Searcher
	policyMutex sync.Mutex

	clusterDatastore  clusterDS.DataStore
	notifierDatastore notifierDS.DataStore
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	if ok, err := policySAC.ReadAllowed(ctx); err != nil || !ok {
		return nil, err
	}
	return ds.searcher.Search(ctx, q)
}

// SearchPolicies
func (ds *datastoreImpl) SearchPolicies(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	return ds.searcher.SearchPolicies(ctx, q)
}

// SearchRawPolicies
func (ds *datastoreImpl) SearchRawPolicies(ctx context.Context, q *v1.Query) ([]*storage.Policy, error) {
	return ds.searcher.SearchRawPolicies(ctx, q)
}

func (ds *datastoreImpl) GetPolicy(ctx context.Context, id string) (*storage.Policy, bool, error) {
	if ok, err := policySAC.ReadAllowed(ctx); err != nil || !ok {
		return nil, false, err
	}

	policy, exists, err := ds.storage.GetPolicy(id)
	if err != nil || !exists {
		return nil, false, err
	}

	return policy, true, nil
}

func (ds *datastoreImpl) GetPolicies(ctx context.Context, ids []string) ([]*storage.Policy, []int, []error, error) {
	if ok, err := policySAC.ReadAllowed(ctx); err != nil || !ok {
		return nil, nil, nil, err
	}

	policies, missingIndices, policyErrors, err := ds.storage.GetPolicies(ids...)
	if err != nil {
		return nil, nil, nil, err
	}

	return policies, missingIndices, policyErrors, nil
}

func (ds *datastoreImpl) GetAllPolicies(ctx context.Context) ([]*storage.Policy, error) {
	if ok, err := policySAC.ReadAllowed(ctx); err != nil || !ok {
		return nil, err
	}

	policies, err := ds.storage.GetAllPolicies()
	if err != nil {
		return nil, err
	}

	return policies, err
}

// GetPolicyByName returns policy with given name.
func (ds *datastoreImpl) GetPolicyByName(ctx context.Context, name string) (policy *storage.Policy, exists bool, err error) {
	if ok, err := policySAC.ReadAllowed(ctx); err != nil || !ok {
		return nil, false, err
	}

	policies, err := ds.GetAllPolicies(ctx)
	if err != nil {
		return nil, false, err
	}

	for _, p := range policies {
		if p.GetName() == name {
			return p, true, nil
		}
	}
	return nil, false, nil
}

// AddPolicy inserts a policy into the storage and the indexer
func (ds *datastoreImpl) AddPolicy(ctx context.Context, policy *storage.Policy) (string, error) {
	if ok, err := policySAC.WriteAllowed(ctx); err != nil {
		return "", err
	} else if !ok {
		return "", errors.New("permission denied")
	}

	store.FillSortHelperFields(policy)

	// No need to lock here because nobody can update the policy
	// until this function returns and they receive the id.
	id, err := ds.storage.AddPolicy(policy)
	if err != nil {
		return id, err
	}
	return id, ds.indexer.AddPolicy(policy)
}

// UpdatePolicy updates a policy from the storage and the indexer
func (ds *datastoreImpl) UpdatePolicy(ctx context.Context, policy *storage.Policy) error {
	if ok, err := policySAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	store.FillSortHelperFields(policy)

	ds.policyMutex.Lock()
	defer ds.policyMutex.Unlock()
	if err := ds.storage.UpdatePolicy(policy); err != nil {
		return err
	}
	return ds.indexer.AddPolicy(policy)
}

// RemovePolicy removes a policy from the storage and the indexer
func (ds *datastoreImpl) RemovePolicy(ctx context.Context, id string) error {
	if ok, err := policySAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	ds.policyMutex.Lock()
	defer ds.policyMutex.Unlock()
	if err := ds.storage.RemovePolicy(id); err != nil {
		return err
	}
	return ds.indexer.DeletePolicy(id)
}

func (ds *datastoreImpl) RenamePolicyCategory(ctx context.Context, request *v1.RenamePolicyCategoryRequest) error {
	if ok, err := policySAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	return ds.storage.RenamePolicyCategory(request)
}

func (ds *datastoreImpl) DeletePolicyCategory(ctx context.Context, request *v1.DeletePolicyCategoryRequest) error {
	if ok, err := policySAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return errors.New("permission denied")
	}

	return ds.storage.DeletePolicyCategory(request)
}

func (ds *datastoreImpl) ImportPolicies(ctx context.Context, importPolicies []*storage.Policy, overwrite bool) ([]*v1.ImportPolicyResponse, bool, error) {
	if ok, err := policySAC.WriteAllowed(ctx); err != nil {
		return nil, false, err
	} else if !ok {
		return nil, false, sac.ErrPermissionDenied
	}

	// Remove all cluster scopes and notifiers that can't be applied to this installation
	changedIndices, err := ds.removeForeignClusterScopesAndNotifiers(ctx, importPolicies...)
	if err != nil {
		return nil, false, errorsPkg.Wrap(err, "removing cluster scopes and notifiers")
	}

	store.FillSortHelperFields(importPolicies...)

	// Store the policies and report any errors
	ds.policyMutex.Lock()
	defer ds.policyMutex.Unlock()

	allPolicies, err := ds.GetAllPolicies(ctx)
	if err != nil {
		return nil, false, errorsPkg.Wrap(err, "getting all policies")
	}
	policiesByName := make(map[string]*storage.Policy, len(allPolicies))
	for _, policy := range allPolicies {
		policiesByName[policy.GetName()] = policy
	}
	allSucceeded := true
	responses := make([]*v1.ImportPolicyResponse, len(importPolicies))
	for i, policy := range importPolicies {
		response := ds.importPolicy(policy, overwrite, policiesByName)
		if !response.Succeeded {
			allSucceeded = false
		}
		if changedIndices.Contains(i) {
			response.Errors = append(response.Errors, &v1.ImportPolicyError{
				Message: "Cluster scopes, cluster whitelists, and notification options have been removed from this policy.",
				Type:    policiesPkg.ErrImportClustersOrNotifiersRemoved,
			})
		}

		responses[i] = response
	}

	return responses, allSucceeded, nil
}

func (ds *datastoreImpl) importPolicy(policy *storage.Policy, overwrite bool, policiesByName map[string]*storage.Policy) *v1.ImportPolicyResponse {
	result := &v1.ImportPolicyResponse{
		Policy: policy,
	}
	var err error
	if overwrite {
		err = ds.importOverwrite(policy, policiesByName)
	} else {
		_, err = ds.storage.AddPolicy(policy)
	}
	if err != nil {
		result.Errors = getImportErrorsFromError(err)
		return result
	}
	err = ds.indexer.AddPolicy(policy)
	if err != nil {
		result.Errors = getImportErrorsFromError(err)
		return result
	}
	result.Succeeded = true
	return result
}

func (ds *datastoreImpl) importOverwrite(policy *storage.Policy, policiesByName map[string]*storage.Policy) error {
	if policy.GetId() == "" {
		// TODO: Reconsider this, IDs to this point have always been generated by the Store.
		policy.Id = uuid.NewV4().String()
	}
	if otherPolicy, ok := policiesByName[policy.GetName()]; ok && otherPolicy.GetId() != policy.GetId() {
		err := ds.storage.RemovePolicy(otherPolicy.GetId())
		if err != nil {
			return err
		}
		err = ds.indexer.DeletePolicy(otherPolicy.GetId())
		if err != nil {
			return err
		}
	}
	// This should never create a name violation because we just removed any name conflicts
	return ds.storage.UpdatePolicy(policy)
}

func getImportErrorsFromError(err error) []*v1.ImportPolicyError {
	var policyError *store.PolicyStoreErrorList
	if errors.As(err, &policyError) {
		return handlePolicyStoreErrorList(policyError)
	}

	return []*v1.ImportPolicyError{
		{
			Message: err.Error(),
			Type:    policiesPkg.ErrImportUnknown,
		},
	}
}

func handlePolicyStoreErrorList(policyError *store.PolicyStoreErrorList) []*v1.ImportPolicyError {
	var errList []*v1.ImportPolicyError
	for _, err := range policyError.Errors {
		var nameErr *store.NameConflictError
		if errors.As(err, &nameErr) {
			errList = append(errList, &v1.ImportPolicyError{
				Message: nameErr.ErrString,
				Type:    policiesPkg.ErrImportDuplicateName,
				Metadata: &v1.ImportPolicyError_DuplicateName{
					DuplicateName: nameErr.ExistingPolicyName,
				},
			})
			continue
		}

		var idError *store.IDConflictError
		if errors.As(err, &idError) {
			errList = append(errList, &v1.ImportPolicyError{
				Message: idError.ErrString,
				Type:    policiesPkg.ErrImportDuplicateID,
				Metadata: &v1.ImportPolicyError_DuplicateName{
					DuplicateName: idError.ExistingPolicyName,
				},
			})
			continue
		}

		errList = append(errList, &v1.ImportPolicyError{
			Message: err.Error(),
			Type:    policiesPkg.ErrImportUnknown,
		})
	}
	return errList
}

// SIDE EFFECTS, THE ORIGINAL OBJECTS WILL BE MODIFIED IN PLACE
func (ds *datastoreImpl) removeForeignClusterScopesAndNotifiers(ctx context.Context, importPolicies ...*storage.Policy) (set.IntSet, error) {
	// pre-load all clusters.  There should be a manageable number.
	clusterList, err := ds.clusterDatastore.GetClusters(ctx)
	if err != nil {
		return nil, err
	}
	clusters := set.NewStringSet()
	for _, cluster := range clusterList {
		clusters.Add(cluster.GetId())
	}

	notifierCache := make(map[string]bool)
	changedIndices := set.NewIntSet()
	for i, policy := range importPolicies {
		modified := false
		var scopes []*storage.Scope
		for _, scope := range policy.GetScope() {
			if scope.GetCluster() == "" {
				scopes = append(scopes, scope)
				continue
			}
			exists := clusters.Contains(scope.GetCluster())
			if exists {
				scopes = append(scopes, scope)
				continue
			}
			modified = true
		}
		policy.Scope = scopes

		var notifiers []string
		for _, notifier := range policy.GetNotifiers() {
			exists, cached := notifierCache[notifier]
			if !cached {
				_, exists, err = ds.notifierDatastore.GetNotifier(ctx, notifier)
				if err != nil {
					return nil, err
				}
				notifierCache[notifier] = exists
			}
			if exists {
				notifiers = append(notifiers, notifier)
				continue
			}
			modified = true
		}
		policy.Notifiers = notifiers

		var whitelists []*storage.Whitelist
		for _, whitelist := range policy.GetWhitelists() {
			whitelistCluster := whitelist.GetDeployment().GetScope().GetCluster()
			if whitelistCluster == "" {
				whitelists = append(whitelists, whitelist)
				continue
			}
			exists := clusters.Contains(whitelistCluster)
			if exists {
				whitelists = append(whitelists, whitelist)
				continue
			}
			modified = true
		}
		policy.Whitelists = whitelists

		if modified {
			changedIndices.Add(i)
		}
	}
	return changedIndices, nil
}
