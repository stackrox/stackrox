package k8srbac

import (
	"github.com/stackrox/rox/generated/storage"
)

// Blankable is a fields whos 'all values' state can be represented by not having any values.
type Blankable interface {
	PolicyRuleField

	IsBlanked(rule *storage.PolicyRule) bool
}

// NewBlankable returns a new instance of a Blankable set of values.
func NewBlankable(underlying PolicyRuleField) Blankable {
	return &blankable{
		underlying: underlying,
	}
}

type blankable struct {
	underlying PolicyRuleField
}

// Get gets the value using the underlying field.
func (w *blankable) Get(rule *storage.PolicyRule) []string {
	return w.underlying.Get(rule)
}

// Set sets the value using the underlying field.
func (w *blankable) Set(values []string, rule *storage.PolicyRule) {
	w.underlying.Set(values, rule)
}

// Merge merges the two by checking for blanks before using the underlying if that fails.
func (w *blankable) Merge(to, from *storage.PolicyRule) {
	// If either has a blank, just set it blank since that covers everything.
	if w.IsBlanked(to) || w.IsBlanked(from) {
		w.Set(nil, to)
		return
	}
	// Use underlying to merge.
	w.underlying.Merge(to, from)
}

// Equals uses the Equals from the underlying field.
func (w *blankable) Equals(first, second *storage.PolicyRule) bool {
	return w.underlying.Equals(first, second)
}

// Grants checks if a blank gives all permissions, if not, it checks that the underlying grants the second.
func (w *blankable) Grants(first, second *storage.PolicyRule) bool {
	if w.IsBlanked(first) {
		return true
	} else if w.IsBlanked(second) { // A filled field cannot grant a blank field.
		return false
	}
	return w.underlying.Grants(first, second)
}

// IsBlanked returns if the field is blanked in the input rule.
func (w *blankable) IsBlanked(rule *storage.PolicyRule) bool {
	return len(w.Get(rule)) == 0
}
