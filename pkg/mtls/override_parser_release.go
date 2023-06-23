//go:build release

package mtls

// Override does not do anything in release builds
func (c *certificateParserWrapper) Override(_ CertificateParser) {
	log.Warn("Override called in production code")
}
