//go:build release

package clusterid

// Override does not do anything in release builds
func (p *parserWrapper) Override(_ Parser) {
	log.Warn("Override called in production code")
}
