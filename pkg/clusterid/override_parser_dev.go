//go:build !release

package clusterid

import (
	"github.com/stackrox/rox/pkg/sync"
)

var mu sync.Mutex

// Override the internal parser. This should only be used for testing.
func (p *parserWrapper) Override(parser Parser) {
	mu.Lock()
	defer mu.Unlock()
	p.parser = parser
}
