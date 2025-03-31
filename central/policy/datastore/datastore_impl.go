package datastore

import (
	"context"
	"errors"
	"fmt"

	errorsPkg "github.com/pkg/errors"
	clusterDS "github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/metrics"
	notifierDS "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/policy/search"
	"github.com/stackrox/rox/central/policy/store"
	categoriesDataStore "github.com/stackrox/rox/central/policycategory/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	policiesPkg "github.com/stackrox/rox/pkg/policies"
	"github.com/stackrox/rox/pkg/policyutils"
	pgPkg "github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	log                       = logging.LoggerForModule()
	workflowAdministrationSAC = sac.ForResource(resources.WorkflowAdministration)

	workflowAdministrationCtx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.WorkflowAdministration)))
)

// PolicyStoreErrorList is used to encapsulate multiple errors returned from policy store methods
type PolicyStoreErrorList struct {
	Errors []error
}

func (p *PolicyStoreErrorList) Error() string {
	return errorhelpers.NewErrorListWithErrors("policy store encountered errors", p.Errors).String()
}

// IDConflictError can be returned by AddPolicies when a policy exists with the same ID as a new policy
type IDConflictError struct {
	ErrString          string
	ExistingPolicyName string
}

func (i *IDConflictError) Error() string {
	return i.ErrString
}

// NameConflictError can be returned by AddPolicies when a policy exists with the same name as a new policy
type NameConflictError struct {
	ErrString          string
	ExistingPolicyName string
}

func (i *NameConflictError) Error() string {
	return i.ErrString
}

type datastoreImpl struct {
	storage     store.Store
	searcher    search.Searcher
	policyMutex sync.Mutex

	clusterDatastore    clusterDS.DataStore
	notifierDatastore   notifierDS.DataStore
	categoriesDatastore categoriesDataStore.DataStore
}

func (ds *datastoreImpl) Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error) {
	if ok, err := workflowAdministrationSAC.ReadAllowed(ctx); err != nil || !ok {
		return nil, err
	}
	return ds.searcher.Search(ctx, q)
}

// Count returns the number of search results from the query
func (ds *datastoreImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	if ok, err := workflowAdministrationSAC.ReadAllowed(ctx); err != nil || !ok {
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
	policies, err := ds.searcher.SearchRawPolicies(ctx, q)
	if err != nil {
		return nil, err
	}
	for _, p := range policies {
		categories, err := ds.categoriesDatastore.GetPolicyCategoriesForPolicy(ctx, p.GetId())
		if err != nil {
			log.Errorf("Failed to find categories associated with policy %s: %q. Error: %v", p.GetId(), p.GetName(), err)
			continue
		}
		for _, c := range categories {
			p.Categories = append(p.Categories, c.GetName())
		}
	}
	return policies, nil
}

func (ds *datastoreImpl) GetPolicy(ctx context.Context, id string) (*storage.Policy, bool, error) {
	if ok, err := workflowAdministrationSAC.ReadAllowed(ctx); err != nil || !ok {
		return nil, false, err
	}

	policy, exists, err := ds.storage.Get(ctx, id)
	if err != nil || !exists {
		return nil, false, err
	}

	err = ds.fillCategoryNames(ctx, policy)
	if err != nil {
		return nil, true, err
	}

	return policy, true, nil
}

func (ds *datastoreImpl) fillCategoryNames(ctx context.Context, policies ...*storage.Policy) error {
	for _, p := range policies {
		categories, err := ds.categoriesDatastore.GetPolicyCategoriesForPolicy(ctx, p.GetId())
		if err != nil {
			return err
		}
		for _, c := range categories {
			p.Categories = append(p.Categories, c.GetName())
		}
	}
	return nil
}
func (ds *datastoreImpl) GetPolicies(ctx context.Context, ids []string) ([]*storage.Policy, []int, error) {
	if ok, err := workflowAdministrationSAC.ReadAllowed(ctx); err != nil || !ok {
		return nil, nil, err
	}

	policies, missingIndices, err := ds.storage.GetMany(ctx, ids)
	if err != nil {
		return nil, nil, err
	}

	err = ds.fillCategoryNames(ctx, policies...)
	if err != nil {
		return nil, nil, err
	}
	return policies, missingIndices, nil
}

func (ds *datastoreImpl) GetAllPolicies(ctx context.Context) ([]*storage.Policy, error) {
	if ok, err := workflowAdministrationSAC.ReadAllowed(ctx); err != nil || !ok {
		return nil, err
	}

	var policies []*storage.Policy
	err := ds.storage.Walk(ctx, func(policy *storage.Policy) error {
		policies = append(policies, policy)
		err := ds.fillCategoryNames(ctx, policy)
		if err != nil {
			return errorsPkg.Wrap(err, "failed to fill category names")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return policies, err
}

// GetPolicyByName returns policy with given name.
func (ds *datastoreImpl) GetPolicyByName(ctx context.Context, name string) (*storage.Policy, bool, error) {
	if ok, err := workflowAdministrationSAC.ReadAllowed(ctx); err != nil || !ok {
		return nil, false, err
	}

	policies, err := ds.GetAllPolicies(ctx)
	if err != nil {
		return nil, false, err
	}

	for _, p := range policies {
		if p.GetName() == name {
			err = ds.fillCategoryNames(ctx, p)
			if err != nil {
				return nil, true, err
			}
			return p, true, nil
		}
	}
	return nil, false, nil
}

// AddPolicy inserts a policy into the storage.
func (ds *datastoreImpl) AddPolicy(ctx context.Context, policy *storage.Policy) (string, error) {
	if ok, err := workflowAdministrationSAC.WriteAllowed(ctx); err != nil {
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

	ctx, tx, err := ds.storage.Begin(ctx)
	if err != nil {
		return "", err
	}

	policyNameToPolicyMap := make(map[string]*storage.Policy, len(allPolicies))
	for _, policy := range allPolicies {
		policyNameToPolicyMap[policy.GetName()] = policy
	}

	if findPolicyWithSameName(policyNameToPolicyMap, policy.GetName()) != nil {
		return "", fmt.Errorf("Could not add policy due to name validation, policy with name %s already exists", policy.GetName())
	}
	policyutils.FillSortHelperFields(policy)
	// Any policy added after startup must be marked custom policy.
	markPoliciesAsCustom(policy)

	// Stash away the category names, since they need to be erased on storage. But the policy insert must happen first,
	// to get an ID, to satisfy foreign key constraints when policy category edges are added.
	policyCategories := policy.GetCategories()
	// Make sure to reset the policy categories field on a clone before upserting; otherwise the given reference
	// will be changed and information lost when the reference is being kept in-memory (like in policy sets).
	clonedPolicy := policy.CloneVT()
	clonedPolicy.Categories = []string{}
	err = ds.storage.Upsert(ctx, clonedPolicy)
	if err != nil {
		return "", ds.wrapWithRollback(ctx, tx, err)
	}

	err = ds.categoriesDatastore.SetPolicyCategoriesForPolicy(ctx, clonedPolicy.GetId(), policyCategories)
	if err != nil {
		return "", ds.wrapWithRollback(ctx, tx, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return "", ds.wrapWithRollback(ctx, tx, err)
	}

	if clonedPolicy.Source == storage.PolicySource_DECLARATIVE {
		metrics.IncrementTotalExternalPoliciesGauge()
	}

	return clonedPolicy.Id, nil
}

// UpdatePolicy updates a policy from the storage.
func (ds *datastoreImpl) UpdatePolicy(ctx context.Context, policy *storage.Policy) error {
	if ok, err := workflowAdministrationSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	if policy.Id == "" {
		return errors.New("policy id not specified")
	}

	policyutils.FillSortHelperFields(policy)

	ds.policyMutex.Lock()
	defer ds.policyMutex.Unlock()

	ctx, tx, err := ds.storage.Begin(ctx)
	if err != nil {
		return err
	}

	// Check if categories need to be created/new policy category edges need to be created/
	// existing policy category edges need to be removed?
	if err := ds.categoriesDatastore.SetPolicyCategoriesForPolicy(ctx, policy.GetId(), policy.GetCategories()); err != nil {
		return ds.wrapWithRollback(ctx, tx, err)
	}
	// Make sure to reset the policy categories field on a clone before upserting; otherwise the given reference
	// will be changed and information lost when the reference is being kept in-memory (like in policy sets).
	clonedPolicy := policy.CloneVT()
	clonedPolicy.Categories = []string{}

	if err = ds.storage.Upsert(ctx, clonedPolicy); err != nil {
		return ds.wrapWithRollback(ctx, tx, err)
	}
	return tx.Commit(ctx)
}

// RemovePolicy removes a policy from the storage.
func (ds *datastoreImpl) RemovePolicy(ctx context.Context, policy *storage.Policy) error {
	if ok, err := workflowAdministrationSAC.WriteAllowed(ctx); err != nil {
		return err
	} else if !ok {
		return sac.ErrResourceAccessDenied
	}

	ds.policyMutex.Lock()
	defer ds.policyMutex.Unlock()

	err := ds.removePolicyNoLock(ctx, policy.GetId())

	if err != nil && policy.Source == storage.PolicySource_DECLARATIVE {
		metrics.DecrementTotalExternalPoliciesGauge()
	}

	return err
}

func (ds *datastoreImpl) removePolicyNoLock(ctx context.Context, id string) error {
	return ds.storage.Delete(ctx, id)
}

func (ds *datastoreImpl) ImportPolicies(ctx context.Context, importPolicies []*storage.Policy, overwrite bool) ([]*v1.ImportPolicyResponse, bool, error) {
	if ok, err := workflowAdministrationSAC.WriteAllowed(ctx); err != nil {
		return nil, false, err
	} else if !ok {
		return nil, false, sac.ErrResourceAccessDenied
	}

	// Remove all cluster scopes and notifiers that can't be applied to this installation
	changedIndices, err := ds.removeForeignClusterScopesAndNotifiers(ctx, importPolicies...)
	if err != nil {
		return nil, false, errorsPkg.Wrap(err, "removing cluster scopes and notifiers")
	}

	policyutils.FillSortHelperFields(importPolicies...)
	// All imported policies must be marked custom policy even if they were exported default policies.
	markPoliciesAsCustom(importPolicies...)

	// Store the policies and report any errors
	ds.policyMutex.Lock()
	defer ds.policyMutex.Unlock()

	allPolicies, err := ds.GetAllPolicies(ctx)
	if err != nil {
		return nil, false, errorsPkg.Wrap(err, "getting all policies")
	}
	policyNameToPolicyMap := make(map[string]*storage.Policy, len(allPolicies))
	for _, policy := range allPolicies {
		policyNameToPolicyMap[policy.GetName()] = policy
	}

	allSucceeded := true
	responses := make([]*v1.ImportPolicyResponse, len(importPolicies))
	for i, policy := range importPolicies {
		response := ds.importPolicy(ctx, policy, overwrite, policyNameToPolicyMap)
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

func (ds *datastoreImpl) importPolicy(ctx context.Context, policy *storage.Policy, overwrite bool, policyNameToPolicyMap map[string]*storage.Policy) *v1.ImportPolicyResponse {
	if policy.GetId() == "" {
		// generate id here since upsert no longer generates id
		policy.Id = uuid.NewV4().String()
	}

	result := &v1.ImportPolicyResponse{
		Policy: policy.CloneVT(),
	}

	importErrors, err := ds.validateUniqueNameAndID(ctx, policy, result, overwrite, policyNameToPolicyMap)
	if err != nil {
		result.Errors = getImportErrorsFromError(err)
		return result
	}
	if len(importErrors) > 0 {
		result.Errors = importErrors
		return result
	}

	if overwrite {
		err = ds.importOverwrite(ctx, policy, policyNameToPolicyMap)
		if err != nil {
			result.Errors = getImportErrorsFromError(err)
			return result
		}
	} else {
		policyCategories := policy.GetCategories()
		policy.Categories = []string{}
		err = ds.storage.Upsert(ctx, policy)
		if err != nil {
			result.Errors = getImportErrorsFromError(err)
			return result
		}

		err = ds.categoriesDatastore.SetPolicyCategoriesForPolicy(ctx, policy.GetId(), policyCategories)
		if err != nil {
			result.Errors = getImportErrorsFromError(err)
			return result
		}
	}
	result.Succeeded = true
	return result
}

func (ds *datastoreImpl) validateUniqueNameAndID(ctx context.Context, policy *storage.Policy, result *v1.ImportPolicyResponse,
	overwrite bool, policyNameToPolicyMap map[string]*storage.Policy) ([]*v1.ImportPolicyError, error) {
	var importErrors []*v1.ImportPolicyError

	if policy.GetId() != "" {
		existingPolicy, exists, err := ds.storage.Get(ctx, policy.GetId())
		if err != nil {
			result.Errors = getImportErrorsFromError(err)
			return importErrors, err
		}
		if exists {
			// New policy cannot have the same id as default policies (aka system policies) and it cannot be overwritten, so always return an error
			if existingPolicy.GetIsDefault() {
				importErrors = append(importErrors,
					duplicateNameImportErrf(policiesPkg.ErrImportDuplicateSystemPolicyID, existingPolicy.GetName(), "system policy with id %q already exists, unable to import policy", policy.GetId()))
			} else if !overwrite {
				// If existing policy is a custom one, then it's only an error if the user hasn't explicitly enabled overwrite
				importErrors = append(importErrors,
					duplicateNameImportErrf(policiesPkg.ErrImportDuplicateID, existingPolicy.GetName(), "policy with id %q already exists, unable to import policy", policy.GetId()))
			}
		}
	}

	if existingPolicy := findPolicyWithSameName(policyNameToPolicyMap, policy.GetName()); existingPolicy != nil { // Only validate name collisions against default policies if this is an overwrite operation
		// Similarly, new policy cannot have the same name as default policies
		if existingPolicy.GetIsDefault() {
			importErrors = append(importErrors,
				duplicateNameImportErrf(policiesPkg.ErrImportDuplicateSystemPolicyName, policy.GetName(), "system policy with id %q already exists, unable to import policy", policy.GetId()))
		} else if !overwrite { // And if not system policy, then it's only an error if it is not an overwrite operation
			importErrors = append(importErrors,
				duplicateNameImportErrf(policiesPkg.ErrImportDuplicateName, policy.GetName(), "policy with id %q already exists, unable to import policy", policy.GetId()))
		}
	}
	return importErrors, nil
}

func duplicateNameImportErrf(errType string, duplicateName string, errMsgTemplate string, args ...interface{}) *v1.ImportPolicyError {
	return &v1.ImportPolicyError{
		Message: fmt.Sprintf(errMsgTemplate, args...),
		Type:    errType,
		Metadata: &v1.ImportPolicyError_DuplicateName{
			DuplicateName: duplicateName,
		},
	}
}

func findPolicyWithSameName(policyNameToPolicyMap map[string]*storage.Policy, name string) *storage.Policy {
	for n, existingPolicy := range policyNameToPolicyMap {
		if n == name {
			return existingPolicy
		}
	}
	return nil
}

func (ds *datastoreImpl) importOverwrite(ctx context.Context, policy *storage.Policy, policyNameToPolicyMap map[string]*storage.Policy) error {
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

	if otherPolicy, ok := policyNameToPolicyMap[policy.GetName()]; ok && otherPolicy.GetId() != policy.GetId() {
		if err := ds.removePolicyNoLock(ctx, otherPolicy.GetId()); err != nil {
			return errorsPkg.Wrapf(err, "removing policy %s", otherPolicy.GetId())
		}
	}

	// This should never create a name violation because we just removed any ID/name conflicts
	policyCategories := policy.GetCategories()
	policy.Categories = []string{}
	err := ds.storage.Upsert(ctx, policy)
	if err != nil {
		return err
	}
	err = ds.categoriesDatastore.SetPolicyCategoriesForPolicy(ctx, policy.GetId(), policyCategories)
	if err != nil {
		return err
	}

	return nil
}

func getImportErrorsFromError(err error) []*v1.ImportPolicyError {
	var policyError *PolicyStoreErrorList
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

func handlePolicyStoreErrorList(policyError *PolicyStoreErrorList) []*v1.ImportPolicyError {
	var errList []*v1.ImportPolicyError
	for _, err := range policyError.Errors {
		var nameErr *NameConflictError
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

		var idError *IDConflictError
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
func (ds *datastoreImpl) removeForeignClusterScopesAndNotifiers(ctx context.Context, importPolicies ...*storage.Policy) (set.Set[int], error) {
	// pre-load all clusters.  There should be a manageable number.
	clusterList, err := ds.clusterDatastore.GetClusters(ctx)
	if err != nil {
		return set.NewIntSet(), err
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
					return set.NewIntSet(), err
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

func (ds *datastoreImpl) wrapWithRollback(ctx context.Context, tx *pgPkg.Tx, err error) error {
	if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
		err = errorsPkg.Wrap(err, "There was an issue rolling back changes in the database")
	}
	return err
}
