package glob

import (
	"sync"

	"github.com/gobwas/glob"
	"github.com/pkg/errors"
)

type Pattern string

var globCache = sync.Map{}

func (p *Pattern) Compile() error {
	if p == nil {
		return nil
	}
	_, err := p.compile()
	return err
}

func (p *Pattern) compile() (glob.Glob, error) {
	g, err := glob.Compile(string(*p))
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to compile %q", string(*p))
	}
	globCache.Store(*p, g)
	return g, nil
}

func (p *Pattern) Match(s string) bool {
	v, ok := globCache.Load(*p)
	if !ok {
		var err error
		if v, err = p.compile(); err != nil {
			return false
		}
	}
	return v.(glob.Glob).Match(s)
}

func (p Pattern) Ptr() *Pattern {
	return &p
}
