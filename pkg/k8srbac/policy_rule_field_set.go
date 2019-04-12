package k8srbac

import (
	"github.com/stackrox/rox/generated/storage"
)

// PolicyRuleFieldSet creates functions for comparing and merging a set of fields.
type PolicyRuleFieldSet interface {
	Merge(to, from *storage.PolicyRule) bool
	Equals(first, second *storage.PolicyRule) bool
	Grants(first, second *storage.PolicyRule) bool
}

// NewPolicyRuleFieldSet returns a new instance of a PolicyRuleFieldSet.
func NewPolicyRuleFieldSet(fields ...PolicyRuleField) PolicyRuleFieldSet {
	return &policyRuleFieldSet{
		fields: fields,
	}
}

type policyRuleFieldSet struct {
	fields []PolicyRuleField
}

// Merge tries to merge from into to, and returns if it was successful.
func (k *policyRuleFieldSet) Merge(to, from *storage.PolicyRule) bool {
	// If the are equal, then consider them merged.
	if k.Equals(to, from) {
		return true
	}
	// To merge, n-1/n fields must be equal. Then the unequal field can be merged.
	for fIndex, mergeField := range k.fields {
		var matchFields []PolicyRuleField
		if fIndex == 0 { // All but first field.
			matchFields = k.fields[1:]
		} else if fIndex == len(k.fields)-1 { // all but last field.
			matchFields = k.fields[:len(k.fields)-1]
		} else { // all but some middle field.
			matchFields = make([]PolicyRuleField, fIndex)
			copy(matchFields, k.fields[:fIndex])
			matchFields = append(matchFields, k.fields[fIndex+1:]...)
		}
		if NewPolicyRuleFieldSet(matchFields...).Equals(to, from) {
			mergeField.Merge(to, from)
			return true
		}
	}
	return false
}

// Equals returns if all of the fields in the field set are equal for the two rules.
func (k *policyRuleFieldSet) Equals(first, second *storage.PolicyRule) bool {
	for _, field := range k.fields {
		if !field.Equals(first, second) {
			return false
		}
	}
	return true
}

// Grants returns if all of the fields in the field set grant the second rule with the first.
func (k *policyRuleFieldSet) Grants(first, second *storage.PolicyRule) bool {
	for _, field := range k.fields {
		if !field.Grants(first, second) {

			return false
		}
	}
	return true
}
