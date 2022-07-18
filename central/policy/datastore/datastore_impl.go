package datastore

import (
	"context"
	"errors"
	"fmt"

	errorsPkg "github.com/pkg/errors"
	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/policy/index"
	"github.com/stackrox/rox/central/policy/search"
	"github.com/stackrox/rox/central/policy/store"
	"github.com/stackrox/rox/central/policy/store/boltdb"
	"github.com/stackrox/rox/central/policy/utils"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	policiesPkg "github.com/stackrox/rox/pkg/policies"
	"github.com/stackrox/rox/pkg/sac"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	log       = logging.LoggerForModule()
	policySAC = sac.ForResource(resources.Policy)

	policyCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Policy)))
)

type datastoreImpl struct {
	storage     store.Store
	indexer     index.Indexer
	searcher    search.Searcher
	policyMutex sync.Mutex

	clusterDatastore  clusterDS.DataStore
	notifierDatastore notifierDS.DataStore
}

func (ds *datastoreImpl) buildIndex() error {
	if features.PostgresDatastore.Enabled() {
		return nil
	}
	policies, err := ds.storage.GetAll(policyCtx)
	if err != nil {
		return err
	}
	return ds.indexer.AddPolicies(policies)
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	if ok, err := policySAC.ReadAllowed(ctx); err != nil || !ok {
		return nil, err
	}
	return ds.searcher.Search(ctx, q)
}

// Count returns the number of search results from the query
func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	if ok, err := policySAC.ReadAllowed(ctx); err != nil || !ok {
		return 0, err
	}
	return ds.searcher.Count(ctx, q)
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

	policy, exists, err := ds.storage.Get(ctx, id)
	if err != nil || !exists {
		return nil, false, err
	}
	return policy, true, nil
}

func (ds *datastoreImpl) GetPolicies(ctx context.Context, ids []string) ([]*storage.Policy, []int, error) {
	if ok, err := policySAC.ReadAllowed(ctx); err != nil || !ok {
		return nil, nil, err
	}

	policies, missingIndices, err := ds.storage.GetMany(ctx, ids)
	if err != nil {
		return nil, nil, err
	}
	return policies, missingIndices, nil
}

func (ds *datastoreImpl) GetAllPolicies(ctx context.Context) ([]*storage.Policy, error) {
	if ok, err := policySAC.ReadAllowed(ctx); err != nil || !ok {
		return nil, err
	}

	policies, err := ds.storage.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	return policies, err
}

// GetPolicyByName returns policy with given name.
func (ds *datastoreImpl) GetPolicyByName(ctx context.Context, name string) (*storage.Policy, bool, error) {
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
		return "", sac.ErrResourceAccessDenied
	}

	if policy.Id == "" {
		policy.Id = uuid.NewV4().String()
	}

	ds.policyMutex.Lock()
	defer ds.policyMutex.Unlock()

	allPolicies, err := ds.GetAllPolicies(ctx)
	if err != nil {
		return "", errorsPkg.Wrap(err, "getting all policies")
	}
	policyNameIDMap := make(map[string]string, len(allPolicies))
	for _, policy := range allPolicies {
		policyNameIDMap[policy.GetName()] = policy.GetId()
	}

	if ds.policyNameIsNotUnique(policyNameIDMap, policy.GetName()) {
		return "", fmt.Errorf("Could not add policy due to name validation, policy with name %s already exists", policy.GetName())
	}
	utils.FillSortHelperFields(policy)
	// Any policy added after startup must be marked custom policy.
	markPoliciesAsCustom(policy)

	// No need to lock here because nobody can update the policy
	// until this function returns and they receive the id.
	err = ds.storage.Upsert(ctx, policy)

	if err != nil {
		return policy.Id, err
	}

	return policy.Id, ds.indexer.AddPolicy(policy)
}

// UpdatePolicy updates a policy from the storage and the indexer
func (ds *datastoreImpl) UpdatePolicy(ctx context.Context, policy *storage.Policy) error {
	if ok, err := policySAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	if policy.Id == "" {
		return errors.New("policy id not specified")
	}

	utils.FillSortHelperFields(policy)

	ds.policyMutex.Lock()
	defer ds.policyMutex.Unlock()
	if err := ds.storage.Upsert(ctx, policy); err != nil {
		return err
	}
	return ds.indexer.AddPolicy(policy)
}

// RemovePolicy removes a policy from the storage and the indexer
func (ds *datastoreImpl) RemovePolicy(ctx context.Context, id string) error {
	if ok, err := policySAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	ds.policyMutex.Lock()
	defer ds.policyMutex.Unlock()

	return ds.removePolicyNoLock(ctx, id)
}

func (ds *datastoreImpl) removePolicyNoLock(ctx context.Context, id string) error {
	if err := ds.storage.Delete(ctx, id); err != nil {
		return err
	}
	return ds.indexer.DeletePolicy(id)
}

func (ds *datastoreImpl) RenamePolicyCategory(ctx context.Context, request *v1.RenamePolicyCategoryRequest) error {
	if features.PostgresDatastore.Enabled() {
		return nil
	}
	if ok, err := policySAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}
	ds.policyMutex.Lock()
	defer ds.policyMutex.Unlock()

	return ds.storage.RenamePolicyCategory(request)
}

func (ds *datastoreImpl) DeletePolicyCategory(ctx context.Context, request *v1.DeletePolicyCategoryRequest) error {
	if features.PostgresDatastore.Enabled() {
		return nil
	}

	if ok, err := policySAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	return ds.storage.DeletePolicyCategory(request)
}

func (ds *datastoreImpl) ImportPolicies(ctx context.Context, importPolicies []*storage.Policy, overwrite bool) ([]*v1.ImportPolicyResponse, bool, error) {
	if ok, err := policySAC.WriteAllowed(ctx); err != nil {
		return nil, false, err
	} else if !ok {
		return nil, false, sac.ErrResourceAccessDenied
	}

	// Remove all cluster scopes and notifiers that can't be applied to this installation
	changedIndices, err := ds.removeForeignClusterScopesAndNotifiers(ctx, importPolicies...)
	if err != nil {
		return nil, false, errorsPkg.Wrap(err, "removing cluster scopes and notifiers")
	}

	utils.FillSortHelperFields(importPolicies...)
	// All imported policies must be marked custom policy even if they were exported default policies.
	markPoliciesAsCustom(importPolicies...)

	// Store the policies and report any errors
	ds.policyMutex.Lock()
	defer ds.policyMutex.Unlock()

	allPolicies, err := ds.GetAllPolicies(ctx)
	if err != nil {
		return nil, false, errorsPkg.Wrap(err, "getting all policies")
	}
	policyNameIDMap := make(map[string]string, len(allPolicies))
	for _, policy := range allPolicies {
		policyNameIDMap[policy.GetName()] = policy.GetId()
	}

	allSucceeded := true
	responses := make([]*v1.ImportPolicyResponse, len(importPolicies))
	for i, policy := range importPolicies {
		response := ds.importPolicy(ctx, policy, overwrite, policyNameIDMap)
		if !response.Succeeded {
			allSucceeded = false
		}
		if changedIndices.Contains(i) {
			response.Errors = append(response.Errors, &v1.ImportPolicyError{
				Message: "Cluster scopes, cluster exclusions, and notification options have been removed from this policy.",
				Type:    policiesPkg.ErrImportClustersOrNotifiersRemoved,
			})
		}

		responses[i] = response
	}

	return responses, allSucceeded, nil
}

func (ds *datastoreImpl) importPolicy(ctx context.Context, policy *storage.Policy, overwrite bool, policyNameIDMap map[string]string) *v1.ImportPolicyResponse {
	result := &v1.ImportPolicyResponse{
		Policy: policy,
	}
	var err error
	if overwrite {
		err = ds.importOverwrite(ctx, policy, policyNameIDMap)
	} else {
		var importErrors []*v1.ImportPolicyError

		if policy.GetId() != "" {
			existingPolicy, exists, err := ds.storage.Get(ctx, policy.GetId())
			if err != nil {
				result.Errors = getImportErrorsFromError(err)
				return result
			}
			if exists {
				importErrors = append(result.Errors, &v1.ImportPolicyError{
					Message: fmt.Sprintf("policy with id '%q' already exists, unable to import policy", policy.GetId()),
					Type:    policiesPkg.ErrImportDuplicateID,
					Metadata: &v1.ImportPolicyError_DuplicateName{
						DuplicateName: existingPolicy.GetName(),
					},
				})
			}
		} else {
			// generate id here since upsert no longer generates id
			policy.Id = uuid.NewV4().String()
		}

		if ds.policyNameIsNotUnique(policyNameIDMap, policy.GetName()) {
			importErrors = append(importErrors, &v1.ImportPolicyError{
				Message: fmt.Sprintf("policy with name '%s' already exists, unable to import policy", policy.GetName()),
				Type:    policiesPkg.ErrImportDuplicateName,
				Metadata: &v1.ImportPolicyError_DuplicateName{
					DuplicateName: policy.GetName(),
				},
			})
		}
		if len(importErrors) > 0 {
			result.Errors = importErrors
			return result
		}
		err = ds.storage.Upsert(ctx, policy)
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

func (ds *datastoreImpl) policyNameIsNotUnique(policyNameIDMap map[string]string, name string) bool {
	for n := range policyNameIDMap {
		if n == name {
			return true
		}
	}
	return false
}

func (ds *datastoreImpl) importOverwrite(ctx context.Context, policy *storage.Policy, policyNameIDMap map[string]string) error {
	if policy.GetId() != "" {
		_, exists, err := ds.storage.Get(ctx, policy.GetId())
		if err != nil {
			return errorsPkg.Wrapf(err, "getting policy %s", policy.GetId())
		}
		if exists {
			if err := ds.removePolicyNoLock(ctx, policy.GetId()); err != nil {
				return errorsPkg.Wrapf(err, "removing policy %s", policy.GetId())
			}
		}
	}

	if otherPolicyID, ok := policyNameIDMap[policy.GetName()]; ok && otherPolicyID != policy.GetId() {
		if err := ds.removePolicyNoLock(ctx, otherPolicyID); err != nil {
			return errorsPkg.Wrapf(err, "removing policy %s", otherPolicyID)
		}
	}

	// This should never create a name violation because we just removed any ID/name conflicts
	return ds.storage.Upsert(ctx, policy)
}

func getImportErrorsFromError(err error) []*v1.ImportPolicyError {
	var policyError *boltdb.PolicyStoreErrorList
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

func handlePolicyStoreErrorList(policyError *boltdb.PolicyStoreErrorList) []*v1.ImportPolicyError {
	var errList []*v1.ImportPolicyError
	for _, err := range policyError.Errors {
		var nameErr *boltdb.NameConflictError
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

		var idError *boltdb.IDConflictError
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

		var exclusions []*storage.Exclusion
		for _, exclusion := range policy.GetExclusions() {
			excludeCluster := exclusion.GetDeployment().GetScope().GetCluster()
			if excludeCluster == "" {
				exclusions = append(exclusions, exclusion)
				continue
			}
			exists := clusters.Contains(excludeCluster)
			if exists {
				exclusions = append(exclusions, exclusion)
				continue
			}
			modified = true
		}
		policy.Exclusions = exclusions

		if modified {
			changedIndices.Add(i)
		}
	}
	return changedIndices, nil
}
