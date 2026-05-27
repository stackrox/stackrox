package glob

import (
	"fmt"

	"github.com/stackrox/rox/pkg/errorhelpers"
)

type GlobValidator struct {
	nfas []*nfa
}

func NewGlobValidator(patterns ...string) (*GlobValidator, error) {
	nfas := make([]*nfa, 0, len(patterns))

	for _, p := range patterns {
		n, err := buildNFA(p)
		if err != nil {
			return nil, err
		}
		nfas = append(nfas, n)
	}
	return &GlobValidator{
		nfas: nfas,
	}, nil
}

func (g *GlobValidator) Overlaps(pattern string) (bool, error) {
	pnfa, err := buildNFA(pattern)
	if err != nil {
		return false, err
	}

	for _, n := range g.nfas {
		if n.intersects(pnfa) {
			return true, nil
		}
	}

	return false, nil
}

// ValidateExceptions checks that all exception patterns overlap with at least
// one capturing pattern. Returns an error containing all invalid exceptions, or
// nil if all exceptions are valid.
func (g *GlobValidator) ValidateExceptions(exceptions ...string) error {
	errList := errorhelpers.NewErrorList("exception validation")

	for _, exc := range exceptions {
		overlaps, err := g.Overlaps(exc)
		if err != nil {
			return fmt.Errorf("failed to validate exception %q: %w", exc, err)
		}
		if !overlaps {
			errList.AddStringf("exception %q does not overlap with any capturing pattern", exc)
		}
	}

	return errList.ToError()
}

// PatternsOverlap reports whether two glob patterns could match the same path.
func PatternsOverlap(pattern1, pattern2 string) (bool, error) {
	n1, err := buildNFA(pattern1)
	if err != nil {
		return false, err
	}
	n2, err := buildNFA(pattern2)
	if err != nil {
		return false, err
	}

	return n1.intersects(n2), nil
}
