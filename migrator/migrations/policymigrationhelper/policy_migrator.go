package policymigrationhelper

import (
	"reflect"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/migrator/log"
	bolt "go.etcd.io/bbolt"
)

// PolicyChanges lists the fields that must match before a policy is updated and what it should be updated to
type PolicyChanges struct {
	FieldsToCompare []FieldComparator
	// ToChange is the set of changes that must be made to this policy
	ToChange PolicyUpdates
}

// PolicyUpdates lists the possible fields of a policy that can be updated. Any nil fields will not be updated
// In order to change an item in an array (e.g. exclusions), remove the existing one and add the updated one back in.
type PolicyUpdates struct {
	// PolicySections is the new policy sections
	PolicySections []*storage.PolicySection
	// ExclusionsToAdd is a list of exclusions to insert (or append) to policy
	ExclusionsToAdd []*storage.Exclusion
	// ExclusionsToRemove is a list of exclusions to remove from policy
	ExclusionsToRemove []*storage.Exclusion
	// Remediation is the new remediation string
	Remediation *string
	// Rationale is the new rationale string
	Rationale *string
	// Description is the new description string
	Description *string
}

func (u *PolicyUpdates) applyToPolicy(policy *storage.Policy) {
	if u == nil {
		return
	}

	if u.ExclusionsToRemove != nil {
		for _, toRemove := range u.ExclusionsToRemove {
			if !removeExclusion(policy, toRemove) {
				log.WriteToStderrf("policy ID %s has already been altered because exclusion was already removed. Will not update.", policy.Id)
				continue
			}
		}
	}

	// Add new exclusions as needed
	if u.ExclusionsToAdd != nil {
		policy.Exclusions = append(policy.Exclusions, u.ExclusionsToAdd...)
	}

	// If policy section is to be updated, just clear the old one for the new
	if u.PolicySections != nil {
		policy.PolicySections = u.PolicySections
	}

	// Update string fields as needed
	if u.Rationale != nil {
		policy.Rationale = *u.Rationale
	}
	if u.Remediation != nil {
		policy.Remediation = *u.Remediation
	}
	if u.Description != nil {
		policy.Description = *u.Description
	}
}

func removeExclusion(policy *storage.Policy, exclusionToRemove *storage.Exclusion) bool {
	exclusions := policy.GetExclusions()
	for i, exclusion := range exclusions {
		if reflect.DeepEqual(exclusion, exclusionToRemove) {
			policy.Exclusions = append(exclusions[:i], exclusions[i+1:]...)
			return true
		}
	}
	return false
}

// FieldComparator should compare policies and return true if they match for a defined field
type FieldComparator func(first, second *storage.Policy) bool

// PolicySectionComparator compares the policySections of both policies and returns true if they are equal
func PolicySectionComparator(first, second *storage.Policy) bool {
	return reflect.DeepEqual(first.GetPolicySections(), second.GetPolicySections())
}

// ExclusionComparator compares the Exclusions of both policies and returns true if they are equal
func ExclusionComparator(first, second *storage.Policy) bool {
	return reflect.DeepEqual(first.GetExclusions(), second.GetExclusions())
}

// RemediationComparator compares the Remediation section of both policies and returns true if they are equal
func RemediationComparator(first, second *storage.Policy) bool {
	return first.GetRemediation() == second.GetRemediation()
}

// RationaleComparator compares the Rationale section of both policies and returns true if they are equal
func RationaleComparator(first, second *storage.Policy) bool {
	return first.GetRationale() == second.GetRationale()
}

// DescriptionComparator compares the Description of both policies and returns true if they are equal
func DescriptionComparator(first, second *storage.Policy) bool {
	return first.GetDescription() == second.GetDescription()
}

var (
	policyBucketName = []byte("policies")
)

// MigratePolicies will migrate all policies in the db as specified by policiesToMigrate assuming the policies in the db
// matches the policies within comparisonPolicies.
func MigratePolicies(db *bolt.DB, policiesToMigrate map[string]PolicyChanges, comparisonPolicies map[string]*storage.Policy) error {
	if exists, err := bolthelpers.BucketExists(db, policyBucketName); err != nil {
		return errors.Wrapf(err, "getting bucket with name %q", policyBucketName)
	} else if !exists {
		return errors.Errorf("unable to find policy bucket with name %s", policyBucketName)
	}

	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(policyBucketName)

		// Migrate and update policies one by one. Abort the transaction, and hence
		// the migration, in case of any error.
		for policyID, updateDetails := range policiesToMigrate {
			v := bucket.Get([]byte(policyID))
			if v == nil {
				log.WriteToStderrf("no policy exists for ID %s in policy migration. Continuing", policyID)
				continue
			}

			var policy storage.Policy
			if err := proto.Unmarshal(v, &policy); err != nil {
				// Unable to recover, so abort transaction
				return errors.Wrapf(err, "unmarshaling migrated policy with id %q", policyID)
			}

			// Fetch the saved policy state to compare with
			comparePolicy, ok := comparisonPolicies[policyID]
			if !ok || comparePolicy == nil {
				return errors.Errorf("policy cannot be compared because comparison policy doesn't exist for %q", policyID)
			}

			// Validate all the required fields to ensure policy hasn't been updated
			if !checkIfPoliciesMatch(updateDetails.FieldsToCompare, comparePolicy, &policy) {
				log.WriteToStderrf("policy ID %s has already been altered. Will not update.", policyID)
				continue
			}

			// Update policy as needed
			updateDetails.ToChange.applyToPolicy(&policy)

			policyBytes, err := proto.Marshal(&policy)
			if err != nil {
				return errors.Wrapf(err, "marshaling migrated policy %q with id %q", policy.GetName(), policy.GetId())
			}
			if err := bucket.Put([]byte(policyID), policyBytes); err != nil {
				return errors.Wrapf(err, "writing migrated policy with id %q to the store", policy.GetId())
			}
		}

		return nil
	})
}

func checkIfPoliciesMatch(fieldsToCompare []FieldComparator, first *storage.Policy, second *storage.Policy) bool {
	for _, field := range fieldsToCompare {
		if !field(first, second) {
			return false
		}
	}
	return true
}
