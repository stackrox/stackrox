package k8srbac

import (
	"strings"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/set"
)

const globChar = "*"

// Globable is a fields that represents a path that can match all subpaths with a glob character.
type Globable interface {
	PolicyRuleField

	Globs(first, second *storage.PolicyRule) bool
}

// NewGlobable returns a new instance of a Globable set of values.
func NewGlobable(underlying PolicyRuleField) Globable {
	return &globable{
		underlying: underlying,
	}
}

type globable struct {
	underlying PolicyRuleField
}

// Get gets the value using the underlying field.
func (g *globable) Get(rule *storage.PolicyRule) []string {
	return g.underlying.Get(rule)
}

// Set sets the value using the underlying field.
func (g *globable) Set(values []string, rule *storage.PolicyRule) {
	g.underlying.Set(values, rule)
}

// Merge merges the two by checking if either is a glob of the other first.
func (g *globable) Merge(to, from *storage.PolicyRule) {
	toVals := g.Get(to)
	fromVals := g.Get(from)
	newValues := set.NewStringSet()
	// All of the from's values that aren't shadowed by a to value.
	for _, fromVal := range fromVals {
		for _, toVal := range toVals {
			if firstGlobsSecond(toVal, fromVal) {
				break
			}
		}
		newValues.Add(fromVal)
	}
	// All of the to's values that aren't shadowed by a from value.
	for _, toVal := range toVals {
		for _, fromVal := range fromVals {
			if firstGlobsSecond(fromVal, toVal) {
				break
			}
		}
		newValues.Add(toVal)
	}
	// Set of non-shadowing values.
	g.underlying.Set(newValues.AsSlice(), to)
}

// Equals uses the Equals from the underlying field.
func (g *globable) Equals(first, second *storage.PolicyRule) bool {
	return g.underlying.Equals(first, second)
}

// Grants checks if a first's glob gives the second permission, if not, it checks that the underlying grants the second.
func (g *globable) Grants(first, second *storage.PolicyRule) bool {
	if g.Globs(first, second) { // if second is a glob of the first, then its granted.
		return true
	}
	return g.underlying.Grants(first, second)
}

// IsGlobOf returns if the first is a globed child of the second.
func (g *globable) Globs(first, second *storage.PolicyRule) bool {
	fVals := g.Get(first)
	sVals := g.Get(second)
	for _, sVal := range sVals {
		if !firstSetGlobsSecond(fVals, sVal) {
			return false // If we run through all of the second's values without finding something that globs.
		}
	}
	return true // all of first's values had globs in second.
}

func firstSetGlobsSecond(first []string, second string) bool {
	for _, fVal := range first {
		if firstGlobsSecond(fVal, second) {
			return true
		}
	}
	return false // If we run through all of the second's values without finding something that globs.
}

func firstGlobsSecond(first, second string) bool {
	if !strings.HasSuffix(first, globChar) { // second is globbing
		return false
	}
	if len(first) == 1 { // Pure wildcard glob
		return true
	}
	if strings.HasPrefix(second, strings.TrimSuffix(first, "*")) { // second is globbing over the first.
		return true
	}
	return false
}
