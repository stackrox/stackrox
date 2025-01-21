package phonehome

import (
	"github.com/gobwas/glob"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/sync"
)

type Pattern string

func (p *Pattern) compile() error {
	if p == nil {
		return nil
	}
	g, err := glob.Compile(string(*p))
	if err != nil {
		return errors.WithMessagef(err, "failed to compile %q", string(*p))
	}
	globCache.Add(*p, g)
	return nil
}

func (p *Pattern) Match(s string) bool {
	return globCache.Get(*p).Match(s)
}

func (p Pattern) Ptr() *Pattern {
	return &p
}

var cacheMux = sync.RWMutex{}

type globCacheType map[Pattern]glob.Glob

func (c globCacheType) Get(pattern Pattern) glob.Glob {
	cacheMux.RLock()
	defer cacheMux.RUnlock()
	return c[pattern]
}

func (c globCacheType) Add(pattern Pattern, value glob.Glob) {
	cacheMux.Lock()
	defer cacheMux.Unlock()
	c[pattern] = value
}

var globCache = make(globCacheType)
