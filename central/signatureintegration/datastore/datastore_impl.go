package datastore

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	policyDatastore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/central/signatureintegration/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	integrationSAC = sac.ForResource(resources.Integration)
)

const (
	// invisiblePolicyPlaceHolder will be used as a placeholder when returning policy names that have references to
	// signature integration for policies that are invisible to the user due to missing access scopes.
	invisiblePolicyPlaceHolder = "<hidden>"
)

type datastoreImpl struct {
	storage store.SignatureIntegrationStore

	policyStore policyDatastore.DataStore

	lock sync.RWMutex
}

func (d *datastoreImpl) GetSignatureIntegration(ctx context.Context, id string) (*storage.SignatureIntegration, bool, error) {
	if ok, err := integrationSAC.ReadAllowed(ctx); !ok || err != nil {
		return nil, false, err
	}

	return d.storage.Get(ctx, id)
}

func (d *datastoreImpl) GetAllSignatureIntegrations(ctx context.Context) ([]*storage.SignatureIntegration, error) {
	if ok, err := integrationSAC.ReadAllowed(ctx); !ok || err != nil {
		return nil, err
	}

	var integrations []*storage.SignatureIntegration
	walkFn := func() error {
		integrations = integrations[:0]
		return d.storage.Walk(ctx, func(integration *storage.SignatureIntegration) error {
			integrations = append(integrations, integration)
			return nil
		})
	}
	if err := pgutils.RetryIfPostgres(walkFn); err != nil {
		return nil, err
	}
	return integrations, nil
}

func (d *datastoreImpl) CountSignatureIntegrations(ctx context.Context) (int, error) {
	if ok, err := integrationSAC.ReadAllowed(ctx); !ok || err != nil {
		return 0, err
	}

	return d.storage.Count(ctx)
}

func (d *datastoreImpl) AddSignatureIntegration(ctx context.Context, integration *storage.SignatureIntegration) (*storage.SignatureIntegration, error) {
	if err := sac.VerifyAuthzOK(integrationSAC.WriteAllowed(ctx)); err != nil {
		return nil, err
	}
	if integration.GetId() != "" {
		return nil, errox.InvalidArgs.Newf("id should be empty but %q provided", integration.GetId())
	}
	integration.Id = GenerateSignatureIntegrationID()
	if err := ValidateSignatureIntegration(integration); err != nil {
		return nil, errox.InvalidArgs.CausedBy(err)
	}

	// Protect against TOCTOU race condition.
	d.lock.Lock()
	defer d.lock.Unlock()

	if err := d.verifyIntegrationIDDoesNotExist(ctx, integration.GetId()); err != nil {
		if errors.Is(err, errox.AlreadyExists) {
			return nil, errors.Wrap(err, "collision in generated signature integration id, try again")
		}
		return nil, err
	}

	err := d.storage.Upsert(ctx, integration)
	if err != nil {
		return nil, err
	}
	return integration, nil
}

func (d *datastoreImpl) UpdateSignatureIntegration(ctx context.Context, integration *storage.SignatureIntegration) (bool, error) {
	if err := sac.VerifyAuthzOK(integrationSAC.WriteAllowed(ctx)); err != nil {
		return false, err
	}
	if err := ValidateSignatureIntegration(integration); err != nil {
		return false, errox.InvalidArgs.CausedBy(err)
	}

	// Protect against TOCTOU race condition.
	d.lock.Lock()
	defer d.lock.Unlock()

	hasUpdatedPublicKeys, err := d.verifyIntegrationIDAndUpdates(ctx, integration)
	if err != nil {
		return false, err
	}

	return hasUpdatedPublicKeys, d.storage.Upsert(ctx, integration)
}

func (d *datastoreImpl) RemoveSignatureIntegration(ctx context.Context, id string) error {
	if err := sac.VerifyAuthzOK(integrationSAC.WriteAllowed(ctx)); err != nil {
		return err
	}

	d.lock.Lock()
	defer d.lock.Unlock()

	if err := d.verifyIntegrationIDExists(ctx, id); err != nil {
		return err
	}

	// We want to avoid deleting a signature integration which is referenced by any policy. If that is the case,
	// stop the deletion of the policy and return an appropriate error message to the user.
	if err := d.verifyIntegrationIDIsNotInPolicy(ctx, id); err != nil {
		return err
	}

	return d.storage.Delete(ctx, id)
}

func (d *datastoreImpl) verifyIntegrationIDExists(ctx context.Context, id string) error {
	_, err := d.getSignatureIntegrationByID(ctx, id)
	if err != nil {
		return err
	}
	return nil
}

func (d *datastoreImpl) getSignatureIntegrationByID(ctx context.Context, id string) (*storage.SignatureIntegration, error) {
	integration, found, err := d.storage.Get(ctx, id)
	if err != nil {
		return nil, err
	} else if !found {
		return nil, errox.NotFound.Newf("signature integration id=%s doesn't exist", id)
	}
	return integration, nil
}

func (d *datastoreImpl) verifyIntegrationIDAndUpdates(ctx context.Context,
	updatedIntegration *storage.SignatureIntegration) (bool, error) {
	existingIntegration, err := d.getSignatureIntegrationByID(ctx, updatedIntegration.GetId())
	if err != nil {
		return false, err
	}
	return !getPublicKeyPEMSet(existingIntegration).Equal(getPublicKeyPEMSet(updatedIntegration)), nil
}

func (d *datastoreImpl) verifyIntegrationIDDoesNotExist(ctx context.Context, id string) error {
	_, found, err := d.storage.Get(ctx, id)
	if err != nil {
		return err
	} else if found {
		return errox.AlreadyExists.Newf("signature integration id=%s already exists", id)
	}
	return nil
}

func (d *datastoreImpl) verifyIntegrationIDIsNotInPolicy(ctx context.Context, id string) error {
	workflowAdministrationCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedResourceLevelScopes(sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.WorkflowAdministration)))

	policies, err := d.policyStore.GetAllPolicies(workflowAdministrationCtx)
	if err != nil {
		return errors.Wrap(err, "retrieving all policies")
	}

	var policiesContainingID []string
	for _, p := range policies {
		if checkIfPolicyContainsID(id, p) {
			policiesContainingID = append(policiesContainingID, p.GetName())
		}
	}

	if len(policiesContainingID) != 0 {
		integration, err := d.getSignatureIntegrationByID(ctx, id)
		if err != nil {
			return err
		}

		// Fetch policies with the user context. Return any error that's not sac.ErrResourceAccessDenied.
		policiesVisibleToUser, err := d.policyStore.GetAllPolicies(ctx)
		if err != nil && !errors.Is(err, sac.ErrResourceAccessDenied) {
			return errors.Wrap(err, "retrieving all policies visible to user")
		}

		listOfPolicies := strings.Join(removePoliciesInvisibleToUser(policiesVisibleToUser, policiesContainingID), ",")

		return errox.ReferencedByAnotherObject.Newf("cannot delete signature integration %q since there are "+
			"existing policies that reference it: [%s]", integration.GetName(), listOfPolicies)
	}

	return nil
}

func checkIfPolicyContainsID(id string, policy *storage.Policy) bool {
	for _, section := range policy.GetPolicySections() {
		for _, group := range section.GetPolicyGroups() {
			// Only check values of the "Image Signature Verified By" field.
			if group.GetFieldName() == search.ImageSignatureVerifiedBy.String() {
				for _, v := range group.GetValues() {
					if v.GetValue() == id {
						return true
					}
				}
			}
		}
	}
	return false
}

// removePoliciesInvisibleToUser will intersect the policy names of policies which are visible to a user
// and policies which have references of signature integrations.
// It will return a list of policy names that are visible to the user, replacing policies
// that have a reference but are not visibile to the user with "<hidden>".
func removePoliciesInvisibleToUser(policiesVisibleToUser []*storage.Policy,
	policiesWithReferences []string) []string {
	// Get names of all policies that are accessible to the user.
	policyNamesVisibleToUser := set.NewStringSet()
	for _, p := range policiesVisibleToUser {
		policyNamesVisibleToUser.Add(p.GetName())
	}

	// Ensure we only return policy names that are visible to the user.
	var policyNames []string
	for _, p := range policiesWithReferences {
		if policyNamesVisibleToUser.Contains(p) {
			policyNames = append(policyNames, p)
		}
	}

	// If we had to skip any amount of policies, add "<hidden>" as a placeholder for non-visible policies referencing
	// integration to the user.
	if len(policiesWithReferences) != len(policyNames) {
		policyNames = append(policyNames, invisiblePolicyPlaceHolder)
	}

	return policyNames
}

func getPublicKeyPEMSet(integration *storage.SignatureIntegration) set.StringSet {
	publicKeySet := set.NewStringSet()
	for _, key := range integration.GetCosign().GetPublicKeys() {
		publicKeySet.Add(key.GetPublicKeyPemEnc())
	}
	return publicKeySet
}
