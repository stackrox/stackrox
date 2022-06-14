package k8srbac

import (
	"github.com/stackrox/stackrox/generated/storage"
)

const wildcard = "*"

// Wildcardable is a fields whos 'all values' state can be represented by a wildcard: *.
type Wildcardable interface {
	PolicyRuleField

	IsWildcarded(rule *storage.PolicyRule) bool
}

// NewWildcardable returns a new instance of a Wildcardable set of values.
func NewWildcardable(underlying PolicyRuleField) Wildcardable {
	return &wildcardable{
		underlying: underlying,
	}
}

type wildcardable struct {
	underlying PolicyRuleField
}

// Get gets the value using the underlying field.
func (w *wildcardable) Get(rule *storage.PolicyRule) []string {
	return w.underlying.Get(rule)
}

// Set sets the value using the underlying field.
func (w *wildcardable) Set(values []string, rule *storage.PolicyRule) {
	w.underlying.Set(values, rule)
}

// Merge merges the two by checking for wildcards before using the underlying if that fails.
func (w *wildcardable) Merge(to, from *storage.PolicyRule) {
	// If either has a wildcard, just set that value and return.
	if w.IsWildcarded(to) || w.IsWildcarded(from) {
		w.Set([]string{wildcard}, to)
		return
	}
	// Use underlying to merge.
	w.underlying.Merge(to, from)
}

// Equals uses the Equals from the underlying field.
func (w *wildcardable) Equals(first, second *storage.PolicyRule) bool {
	return w.underlying.Equals(first, second)
}

// Grants checks if a wildcard gives all permissions, if not, it checks that the underlying grants the second.
func (w *wildcardable) Grants(first, second *storage.PolicyRule) bool {
	if w.IsWildcarded(first) {
		return true
	}
	return w.underlying.Grants(first, second)
}

// IsWildcarded returns if the field is wildcarded in the input rule.
func (w *wildcardable) IsWildcarded(rule *storage.PolicyRule) bool {
	for _, v := range w.Get(rule) {
		if v == wildcard {
			return true
		}
	}
	return false
}
