package datastore

import (
	"context"
	"strings"

	"github.com/pkg/errors"
	policyDatastore "github.com/stackrox/rox/central/policy/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/signatureintegration/store"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	signatureSAC = sac.ForResource(resources.SignatureIntegration)

	policySAC = sac.ForResource(resources.Policy)
)

type datastoreImpl struct {
	storage store.SignatureIntegrationStore

	policyStore policyDatastore.DataStore

	lock sync.RWMutex
}

func (d *datastoreImpl) GetSignatureIntegration(ctx context.Context, id string) (*storage.SignatureIntegration, bool, error) {
	if ok, err := signatureSAC.ReadAllowed(ctx); !ok || err != nil {
		return nil, false, err
	}

	return d.storage.Get(ctx, id)
}

func (d *datastoreImpl) GetAllSignatureIntegrations(ctx context.Context) ([]*storage.SignatureIntegration, error) {
	if ok, err := signatureSAC.ReadAllowed(ctx); !ok || err != nil {
		return nil, err
	}

	var integrations []*storage.SignatureIntegration
	err := d.storage.Walk(ctx, func(integration *storage.SignatureIntegration) error {
		integrations = append(integrations, integration)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return integrations, nil
}

func (d *datastoreImpl) AddSignatureIntegration(ctx context.Context, integration *storage.SignatureIntegration) (*storage.SignatureIntegration, error) {
	if err := sac.VerifyAuthzOK(signatureSAC.WriteAllowed(ctx)); err != nil {
		return nil, err
	}
	if integration.GetId() != "" {
		return nil, errox.InvalidArgs.Newf("id should be empty but %q provided", integration.GetId())
	}
	integration.Id = GenerateSignatureIntegrationID()
	if err := ValidateSignatureIntegration(integration); err != nil {
		return nil, errox.NewErrInvalidArgs(err.Error())
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
	if err := sac.VerifyAuthzOK(signatureSAC.WriteAllowed(ctx)); err != nil {
		return false, err
	}
	if err := ValidateSignatureIntegration(integration); err != nil {
		return false, errox.NewErrInvalidArgs(err.Error())
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
	if err := sac.VerifyAuthzOK(signatureSAC.WriteAllowed(ctx)); err != nil {
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
	policyCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Policy)))

	policies, err := d.policyStore.GetAllPolicies(policyCtx)
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

		// Fetch policies with the user context, we can safely ignore the error since one would be returned in case
		// the user does not have access to policies.
		policiesVisibleToUser, _ := d.policyStore.GetAllPolicies(ctx)

		listOfPolicies := strings.Join(policiesContainingID, ",")
		// If the length of fetched policies is less than the policies that reference signature integration IDs, we
		// assume that the caller does not have read access to policies and will not print policy names in the error
		// message.
		if len(policiesVisibleToUser) < len(policiesContainingID) {
			listOfPolicies = "<hidden>"
		}

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

func getPublicKeyPEMSet(integration *storage.SignatureIntegration) set.StringSet {
	publicKeySet := set.NewStringSet()
	for _, key := range integration.GetCosign().GetPublicKeys() {
		publicKeySet.Add(key.GetPublicKeyPemEnc())
	}
	return publicKeySet
}
