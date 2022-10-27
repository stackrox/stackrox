package k8srbac

import (
	"sort"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sliceutils"
)

// PolicyRuleField represents a field of a PolicyRule.
type PolicyRuleField interface {
	Get(rule *storage.PolicyRule) []string
	Set(values []string, rule *storage.PolicyRule)

	Merge(to, from *storage.PolicyRule)
	Equals(first, second *storage.PolicyRule) bool
	Grants(first, second *storage.PolicyRule) bool
}

// Accessor accesses a field value in a rule.
type Accessor func(rule *storage.PolicyRule) []string

// Setter sets a field value in a rule.
type Setter func(value []string, rule *storage.PolicyRule)

// NewPolicyRuleField returns a new instance of a PolicyRuleField.
func NewPolicyRuleField(accessor Accessor, setter Setter) PolicyRuleField {
	return &policyRuleField{
		accessor: accessor,
		setter:   setter,
	}
}

type policyRuleField struct {
	accessor Accessor
	setter   Setter
}

// Get returns the value of the field in the input rule.
func (f *policyRuleField) Get(rule *storage.PolicyRule) []string {
	return f.accessor(rule)
}

// Set sets the value of the field in the rule.
func (f *policyRuleField) Set(values []string, rule *storage.PolicyRule) {
	f.setter(values, rule)
}

// Merge merges the values of the field in the two rules.
func (f *policyRuleField) Merge(to, from *storage.PolicyRule) {
	for _, fromValue := range f.accessor(from) {
		if !f.hasValue(to, fromValue) {
			toSet := append(f.accessor(to), fromValue)
			sort.SliceStable(toSet, func(i, j int) bool {
				return toSet[i] < toSet[j]
			})
			f.setter(toSet, to)
		}
	}
}

// Equals returns if the value of the field in the two rules is equal.
func (f *policyRuleField) Equals(first, second *storage.PolicyRule) bool {
	if len(f.accessor(first)) != len(f.accessor(second)) {
		return false
	}
	for _, firstValue := range f.accessor(first) {
		if !f.hasValue(second, firstValue) {
			return false
		}
	}
	return true
}

// Grants returns if the value in the first rule grants the values in the second rule. For a base field, that means
// that first's field has all of the values present in the second's.
func (f *policyRuleField) Grants(first, second *storage.PolicyRule) bool {
	// Can't grant if the second has more permissions than the first.
	if len(f.accessor(second)) > len(f.accessor(first)) {
		return false
	}
	for _, shadowedValue := range f.accessor(second) {
		if !f.hasValue(first, shadowedValue) {
			return false
		}
	}
	return true
}

func (f *policyRuleField) hasValue(rule *storage.PolicyRule, value string) bool {
	return sliceutils.Find(f.accessor(rule), value) >= 0
}
