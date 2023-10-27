package policymigrationhelper

import (
	"context"
	"embed"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	pglog "github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/pkg/sac"
)

// MigratePoliciesWithDiffsAndStore migrates policies with the given diffs.
// The policyDiffFS should be an embedded FS that satisfies the following conditions:
// 1. It must contain a top-level directory called "policies_before_and_after".
// 2. That directory must contain two subdirectories: "before" and "after".
// 3. For each policy being migrated, there must be one copy in the "before" directory and one in the "after" directory.
// 4. The file names for a policy should match the PolicyFileName in the corresponding PolicyDiff passed in the third argument.
// This function then automatically computes the diff for each policy, and executes the migration.
// This method requires the caller to provide funcs that check if a policy exists,
// fetches policy from the store and will upsert a policy to the store
// Deprecated: Use MigratePoliciesWithDiffsAndStoreV2 instead
func MigratePoliciesWithDiffsAndStore(policyDiffFS embed.FS,
	policyDiffs []PolicyDiff,
	policyExists func(context.Context, string) (bool, error),
	getPolicy func(context.Context, string) (*storage.Policy, bool, error),
	upsertPolicy func(context.Context, *storage.Policy) error,
) error {
	policiesToMigrate := make(map[string]PolicyChanges, len(policyDiffs))
	preMigrationPolicies := make(map[string]*storage.Policy, len(policyDiffs))
	for _, diff := range policyDiffs {
		beforePolicy, err := ReadPolicyFromFile(policyDiffFS, filepath.Join(beforeDirName, diff.PolicyFileName))
		if err != nil {
			return err
		}
		afterPolicy, err := ReadPolicyFromFile(policyDiffFS, filepath.Join(afterDirName, diff.PolicyFileName))
		if err != nil {
			return err
		}
		if beforePolicy.GetId() == "" || beforePolicy.GetId() != afterPolicy.GetId() {
			return errors.Errorf("policies in file %s don't both have the same, non-empty, id", diff.PolicyFileName)
		}
		updates, err := diffPolicies(beforePolicy, afterPolicy)
		if err != nil {
			return err
		}
		policiesToMigrate[beforePolicy.GetId()] = PolicyChanges{FieldsToCompare: diff.FieldsToCompare, ToChange: updates}
		preMigrationPolicies[beforePolicy.GetId()] = beforePolicy
	}
	return MigratePoliciesWithStore(policiesToMigrate, preMigrationPolicies, policyExists, getPolicy, upsertPolicy)
}

// MigratePoliciesWithStore will migrate all policies in the db as specified by policiesToMigrate assuming the policies in the db
// matches the policies within comparisonPolicies. This method requires the caller to provide funcs that check if a policy exists,
// fetches policy from the store and will upsert a policy to the store
// Deprecated: Use MigratePoliciesWithStoreV2 instead
func MigratePoliciesWithStore(policiesToMigrate map[string]PolicyChanges,
	comparisonPolicies map[string]*storage.Policy,
	policyExists func(context.Context, string) (bool, error),
	getPolicy func(context.Context, string) (*storage.Policy, bool, error),
	upsertPolicy func(context.Context, *storage.Policy) error,
) error {

	ctx := sac.WithAllAccess(context.Background())

	for policyID, updateDetails := range policiesToMigrate {
		if exists, err := policyExists(ctx, policyID); err != nil {
			pglog.WriteToStderrf("err getting policy with id %s. Will not update.", policyID)
			continue
		} else if !exists {
			pglog.WriteToStderrf("unable to find policy with id %s. Will not update.", policyID)
			continue
		}

		policy, _, _ := getPolicy(ctx, policyID)
		comparePolicy := comparisonPolicies[policyID]
		if !checkIfPoliciesMatch(updateDetails.FieldsToCompare, comparePolicy, policy) {
			pglog.WriteToStderrf("policy ID %s has already been altered. Will not update.", policyID)
			continue
		}

		// Update policy as needed
		updateDetails.ToChange.applyToPolicy(policy)
		err := upsertPolicy(ctx, policy)
		if err != nil {
			return err
		}

	}

	return nil
}

// MigratePoliciesWithDiffsAndStoreV2 migrates policies with the given diffs.
// The policyDiffFS should be an embedded FS that satisfies the following conditions:
// 1. It must contain a top-level directory called "policies_before_and_after".
// 2. That directory must contain two subdirectories: "before" and "after".
// 3. For each policy being migrated, there must be one copy in the "before" directory and one in the "after" directory.
// 4. The file names for a policy should match the PolicyFileName in the corresponding PolicyDiff passed in the third argument.
// This function then automatically computes the diff for each policy, and executes the migration.
// This method requires the caller to provide funcs that check if a policy exists,
// fetches policy from the store and will upsert a policy to the store
func MigratePoliciesWithDiffsAndStoreV2(policyDiffFS embed.FS,
	policyDiffs []PolicyDiff,
	getPolicy func(context.Context, string) (*storage.Policy, bool, error),
	upsertPolicy func(context.Context, *storage.Policy) error,
) error {
	policiesToMigrate := make(map[string]PolicyChanges, len(policyDiffs))
	preMigrationPolicies := make(map[string]*storage.Policy, len(policyDiffs))
	for _, diff := range policyDiffs {
		beforePolicy, err := ReadPolicyFromFile(policyDiffFS, filepath.Join(beforeDirName, diff.PolicyFileName))
		if err != nil {
			return err
		}
		afterPolicy, err := ReadPolicyFromFile(policyDiffFS, filepath.Join(afterDirName, diff.PolicyFileName))
		if err != nil {
			return err
		}
		if beforePolicy.GetId() == "" || beforePolicy.GetId() != afterPolicy.GetId() {
			return errors.Errorf("policies in file %s don't both have the same, non-empty, id", diff.PolicyFileName)
		}
		updates, err := diffPolicies(beforePolicy, afterPolicy)
		if err != nil {
			return err
		}
		policiesToMigrate[beforePolicy.GetId()] = PolicyChanges{FieldsToCompare: diff.FieldsToCompare, ToChange: updates}
		preMigrationPolicies[beforePolicy.GetId()] = beforePolicy
	}
	return MigratePoliciesWithStoreV2(policiesToMigrate, preMigrationPolicies, getPolicy, upsertPolicy)
}

// MigratePoliciesWithStoreV2 will migrate all policies in the db as specified by policiesToMigrate assuming the policies in the db
// matches the policies within comparisonPolicies. This method requires the caller to provide funcs that check if a policy exists,
// fetches policy from the store and will upsert a policy to the store
func MigratePoliciesWithStoreV2(policiesToMigrate map[string]PolicyChanges,
	comparisonPolicies map[string]*storage.Policy,
	getPolicy func(context.Context, string) (*storage.Policy, bool, error),
	upsertPolicy func(context.Context, *storage.Policy) error,
) error {
	ctx := sac.WithAllAccess(context.Background())

	for policyID, updateDetails := range policiesToMigrate {
		policy, exists, err := getPolicy(ctx, policyID)
		if err != nil {
			pglog.WriteToStderrf("err getting policy with id %s. Will not update.", policyID)
			continue
		} else if !exists {
			pglog.WriteToStderrf("unable to find policy with id %s. Will not update.", policyID)
			continue
		}

		comparePolicy := comparisonPolicies[policyID]
		if !checkIfPoliciesMatch(updateDetails.FieldsToCompare, comparePolicy, policy) {
			pglog.WriteToStderrf("policy ID %s has already been altered. Will not update.", policyID)
			continue
		}

		// Update policy as needed
		updateDetails.ToChange.applyToPolicy(policy)
		err = upsertPolicy(ctx, policy)
		if err != nil {
			return err
		}

	}
	return nil
}
