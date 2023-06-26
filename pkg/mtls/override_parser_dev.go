//go:build !release

package mtls

// OverrideCertificateParser the CertificateParser parser. This should only be used for testing.
func OverrideCertificateParser(parser CertificateParser) {
	mu.Lock()
	defer mu.Unlock()
	instance = parser
}
