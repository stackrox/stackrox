//go:build !release

package mtls

import "github.com/stackrox/rox/pkg/sync"

var mu sync.Mutex

// Override the internal parser. This should only be used for testing.
func (c *certificateParserWrapper) Override(parser CertificateParser) {
	mu.Lock()
	defer mu.Unlock()
	c.parser = parser
}
