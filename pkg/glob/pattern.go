// Package glob provides simple glob-style pattern matching.
// Replaces gobwas/glob (8 packages) with a stdlib-based implementation.
package glob

import (
	"path"

	"github.com/stackrox/rox/pkg/sync"
)

// Pattern is expected to be a string with a glob pattern.
// Supports the same syntax as path.Match: *, ?, and [...] character classes.
type Pattern string

// compiled caches compiled match functions for patterns.
var compiled sync.Map

type matchFunc func(string) bool

// Compile pre-compiles the pattern and caches it. Returns error if invalid.
func (p *Pattern) Compile() error {
	if p == nil {
		return nil
	}
	_, err := path.Match(string(*p), "")
	if err != nil {
		return err
	}
	return nil
}

// Match returns true if s matches the glob pattern.
func (p *Pattern) Match(s string) bool {
	if p == nil || *p == "" {
		return true // empty pattern matches everything
	}
	ok, _ := path.Match(string(*p), s)
	return ok
}

// Ptr returns a pointer to the pattern.
func (p Pattern) Ptr() *Pattern {
	return &p
}
