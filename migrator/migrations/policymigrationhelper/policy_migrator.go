package policymigrationhelper

import (
	"embed"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/pkg/jsonutil"
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
	// PolicySections is the new policy sections
	PolicySections []*storage.PolicySection
	// MitreVectors is the new MITRE ATT&CK section
	MitreVectors []*storage.Policy_MitreAttackVectors
	// ExclusionsToAdd is a list of exclusions to insert (or append) to policy
	ExclusionsToAdd []*storage.Exclusion
	// ExclusionsToRemove is a list of exclusions to remove from policy
	ExclusionsToRemove []*storage.Exclusion
	// Name is the new name for the policy
	Name *string
	// Remediation is the new remediation string
	Remediation *string
	// Rationale is the new rationale string
	Rationale *string
	// Description is the new description string
	Description *string
	// Disable is true if the policy should be disabled, false if it should be enabled and nil for no change
	Disable *bool
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

	// If policy mitre is to be updated, just clear the old one for the new
	if u.MitreVectors != nil {
		policy.MitreAttackVectors = u.MitreVectors
	}

	// Check if policy should be enabled or disabled
	if u.Disable != nil {
		policy.Disabled = *u.Disable
	}

	// Update string fields as needed
	if u.Name != nil {
		policy.Name = *u.Name
	}
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

// NameComparator compares both policies' names and returns true if they are equal
func NameComparator(first, second *storage.Policy) bool {
	return strings.TrimSpace(first.GetName()) == strings.TrimSpace(second.GetName())
}

// RemediationComparator compares the Remediation section of both policies and returns true if they are equal
func RemediationComparator(first, second *storage.Policy) bool {
	return strings.TrimSpace(first.GetRemediation()) == strings.TrimSpace(second.GetRemediation())
}

// RationaleComparator compares the Rationale section of both policies and returns true if they are equal
func RationaleComparator(first, second *storage.Policy) bool {
	return strings.TrimSpace(first.GetRationale()) == strings.TrimSpace(second.GetRationale())
}

// DescriptionComparator compares the Description of both policies and returns true if they are equal
func DescriptionComparator(first, second *storage.Policy) bool {
	return strings.TrimSpace(first.GetDescription()) == strings.TrimSpace(second.GetDescription())
}

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

	// Policy section
	if !reflect.DeepEqual(beforePolicy.GetPolicySections(), afterPolicy.GetPolicySections()) {
		updates.PolicySections = afterPolicy.PolicySections
	}
	beforePolicy.PolicySections = nil
	afterPolicy.PolicySections = nil

	// MITRE section
	if !reflect.DeepEqual(beforePolicy.GetMitreAttackVectors(), afterPolicy.GetMitreAttackVectors()) {
		updates.MitreVectors = afterPolicy.MitreAttackVectors
	}
	beforePolicy.MitreAttackVectors = nil
	afterPolicy.MitreAttackVectors = nil

	// Name
	if beforePolicy.GetName() != afterPolicy.GetName() {
		updates.Name = strPtr(afterPolicy.Name)
	}
	beforePolicy.Name = ""
	afterPolicy.Name = ""

	// Description
	if beforePolicy.GetDescription() != afterPolicy.GetDescription() {
		updates.Description = strPtr(afterPolicy.Description)
	}
	beforePolicy.Description = ""
	afterPolicy.Description = ""

	// Rationale
	if beforePolicy.GetRationale() != afterPolicy.GetRationale() {
		updates.Rationale = strPtr(afterPolicy.Rationale)
	}
	beforePolicy.Rationale = ""
	afterPolicy.Rationale = ""

	// Remediation
	if beforePolicy.GetRemediation() != afterPolicy.GetRemediation() {
		updates.Remediation = strPtr(afterPolicy.Remediation)
	}
	beforePolicy.Remediation = ""
	afterPolicy.Remediation = ""

	// Enable/Disable
	if beforePolicy.GetDisabled() != afterPolicy.GetDisabled() {
		updates.Disable = boolPtr(afterPolicy.GetDisabled())
	}
	beforePolicy.Disabled = false
	afterPolicy.Disabled = false

	// TODO: Add others as needed

	if !reflect.DeepEqual(beforePolicy, afterPolicy) {
		return PolicyUpdates{}, errors.New("policies have diff after nil-ing out fields we checked, please update this function " +
			"to be able to diff more fields")
	}
	return updates, nil
}

func getExclusionsUpdates(beforePolicy *storage.Policy, afterPolicy *storage.Policy, updates *PolicyUpdates) {
	matchedAfterExclusionsIdxs := set.NewSet[int]()
	for _, beforeExclusion := range beforePolicy.GetExclusions() {
		var found bool
		for afterExclusionIdx, afterExclusion := range afterPolicy.GetExclusions() {
			if reflect.DeepEqual(beforeExclusion, afterExclusion) {
				if !matchedAfterExclusionsIdxs.Contains(afterExclusionIdx) { // to account for duplicates
					found = true
					matchedAfterExclusionsIdxs.Add(afterExclusionIdx)
					break
				}

			}
		}
		if !found {
			updates.ExclusionsToRemove = append(updates.ExclusionsToRemove, beforeExclusion)
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

func checkIfPoliciesMatch(fieldsToCompare []FieldComparator, first *storage.Policy, second *storage.Policy) bool {
	for _, field := range fieldsToCompare {
		if !field(first, second) {
			return false
		}
	}
	return true
}

func strPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}
