package k8srbac

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
)

// PolicyRuleFieldSet creates functions for comparing and merging a set of fields.
type PolicyRuleFieldSet interface {
	Merge(to, from *storage.PolicyRule) bool
	Equals(first, second *storage.PolicyRule) bool
	Grants(first, second *storage.PolicyRule) bool

	Granters(rule *storage.PolicyRule) []*storage.PolicyRule
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

// Granters returns a list of policy rules that will grant the permissions in the given policy rule (including the same rule).
// It creates all possible permutations of the Granters for each field. For example:
// IF
// p.Fields = [Wildcardable(ApiGroups), Wildcardable(Verbs)] and rule = { ApiGroups: ["", "custom"], Verbs: ["Get", "Put"] }
// THEN
// p.Granters(rule) = [
//     { ApiGroups: ["", "custom"], Verbs: ["Get", "Put"] }
//     { ApiGroups: ["*"], Verbs: ["Get", "Put"] }
//     { ApiGroups: ["", "custom"], Verbs: ["*"] }
//     { ApiGroups: ["*"], Verbs: ["*"] }
// ]
// Because each field in rule can be matched by all values being present, or a wildcard being present.
func (k *policyRuleFieldSet) Granters(rule *storage.PolicyRule) []*storage.PolicyRule {
	var options []*storage.PolicyRule
	for _, field := range k.fields {
		fieldOptions := field.Granters(rule)
		if len(fieldOptions) == 0 {
			return nil
		}
		if len(options) == 0 {
			for _, fieldOption := range fieldOptions {
				newRule := &storage.PolicyRule{}
				field.Set(fieldOption, newRule)
				options = append(options, newRule)
			}
		} else {
			newOptions := make([]*storage.PolicyRule, 0, len(options)*len(fieldOptions))
			for _, fieldOption := range fieldOptions {
				for _, rule := range options {
					newRule := proto.Clone(rule).(*storage.PolicyRule)
					field.Set(fieldOption, newRule)
					newOptions = append(newOptions, newRule)
				}
			}
			options = newOptions
		}
	}
	return options
}
