package postgres

import (
	"context"
	"embed"
	"path/filepath"
	"reflect"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	pglog "github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/pkg/jsonutil"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
)

// PolicyDiff is an alternative to PolicyChanges that automatically constructs migrations based on diffs of policies.
type PolicyDiff struct {
	FieldsToCompare []FieldComparator
	PolicyFileName  string
}

// PolicyChanges lists the fields that must match before a policy is updated and what it should be updated to
type PolicyChanges struct {
	FieldsToCompare []FieldComparator
	// ToChange is the set of changes that must be made to this policy
	ToChange PolicyUpdates
}

// PolicyUpdates lists the possible fields of a policy that can be updated. Any nil fields will not be updated
// In order to change an item in an array (e.g. exclusions), remove the existing one and add the updated one back in.
type PolicyUpdates struct {
	ExclusionsToAdd []*storage.Exclusion
}

// FieldComparator should compare policies and return true if they match for a defined field
type FieldComparator func(first, second *storage.Policy) bool

// ReadPolicyFromFile reads policies from file given the path and the collection of files.
func ReadPolicyFromFile(fs embed.FS, filePath string) (*storage.Policy, error) {
	contents, err := fs.ReadFile(filePath)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to read file %s", filePath)
	}
	var policy storage.Policy
	err = jsonutil.JSONBytesToProto(contents, &policy)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to unmarshal policy json at path %s", filePath)
	}
	return &policy, nil
}

func diffPolicies(beforePolicy, afterPolicy *storage.Policy) (PolicyUpdates, error) {
	if beforePolicy == nil {
		return PolicyUpdates{}, errors.New("policy to be migrated must not be nil")
	}
	if afterPolicy == nil {
		return PolicyUpdates{}, errors.New("policy to be migrated does not have valid target, but found nil target")
	}

	// Clone policies because we mutate them.
	beforePolicy = beforePolicy.Clone()
	afterPolicy = afterPolicy.Clone()

	var updates PolicyUpdates

	// Diff exclusions and clear out if they are similar
	getExclusionsUpdates(beforePolicy, afterPolicy, &updates)
	beforePolicy.Exclusions = nil
	afterPolicy.Exclusions = nil

	if !reflect.DeepEqual(beforePolicy, afterPolicy) {
		return PolicyUpdates{}, errors.New("policies have diff after nil-ing out fields we checked, please update this function " +
			"to be able to diff more fields")
	}
	return updates, nil
}

func getExclusionsUpdates(beforePolicy *storage.Policy, afterPolicy *storage.Policy, updates *PolicyUpdates) {
	matchedAfterExclusionsIdxs := set.NewSet[int]()
	for _, beforeExclusion := range beforePolicy.GetExclusions() {
		for afterExclusionIdx, afterExclusion := range afterPolicy.GetExclusions() {
			if reflect.DeepEqual(beforeExclusion, afterExclusion) {
				matchedAfterExclusionsIdxs.Add(afterExclusionIdx)
				break
			}
		}
	}
	for i, exclusion := range afterPolicy.GetExclusions() {
		if !matchedAfterExclusionsIdxs.Contains(i) {
			updates.ExclusionsToAdd = append(updates.ExclusionsToAdd, exclusion)
		}
	}
}

const (
	policyDiffParentDirName = "policies_before_and_after"
	beforeDirName           = policyDiffParentDirName + "/before"
	afterDirName            = policyDiffParentDirName + "/after"
)

func (u *PolicyUpdates) applyToPolicy(policy *storage.Policy) {
	if u == nil {
		return
	}
	// Add new exclusions as needed
	if u.ExclusionsToAdd != nil {
		policy.Exclusions = append(policy.Exclusions, u.ExclusionsToAdd...)
	}
}

// MigratePoliciesWithDiffs migrates policies with the given diffs.
// The policyDiffFS should be an embedded FS that satisfies the following conditions:
// 1. It must contain a top-level directory called "policies_before_and_after".
// 2. That directory must contain two subdirectories: "before" and "after".
// 3. For each policy being migrated, there must be one copy in the "before" directory and one in the "after" directory.
// 4. The file names for a policy should match the PolicyFileName in the corresponding PolicyDiff passed in the third argument.
// This function then automatically computes the diff for each policy, and executes the migration.
func MigratePoliciesWithDiffs(db postgres.DB, policyDiffFS embed.FS, policyDiffs []PolicyDiff) error {
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
	return MigratePolicies(db, policiesToMigrate, preMigrationPolicies)
}

// MigratePolicies will migrate all policies in the db as specified by policiesToMigrate assuming the policies in the db
// matches the policies within comparisonPolicies.
func MigratePolicies(db postgres.DB, policiesToMigrate map[string]PolicyChanges, comparisonPolicies map[string]*storage.Policy) error {

	ctx := sac.WithAllAccess(context.Background())
	policyStore := New(db)

	for policyID, updateDetails := range policiesToMigrate {
		policy, _, _ := policyStore.Get(ctx, policyID)
		comparePolicy := comparisonPolicies[policyID]
		if exists, err := policyStore.Exists(ctx, policyID); err != nil {
			pglog.WriteToStderrf("err getting policy with id %s. Will not update.", policyID)
			continue
		} else if !exists {
			pglog.WriteToStderrf("unable to find policy with id %s. Will not update.", policyID)
			continue
		}
		if !checkIfPoliciesMatch(updateDetails.FieldsToCompare, comparePolicy, policy) {
			pglog.WriteToStderrf("policy ID %s has already been altered. Will not update.", policyID)
			continue
		}

		// Update policy as needed
		updateDetails.ToChange.applyToPolicy(policy)
		err := policyStore.Upsert(ctx, policy)
		if err != nil {
			return err
		}

	}

	return nil

}

func checkIfPoliciesMatch(fieldsToCompare []FieldComparator, first *storage.Policy, second *storage.Policy) bool {
	for _, field := range fieldsToCompare {
		if !field(first, second) {
			return false
		}
	}
	return true
}
