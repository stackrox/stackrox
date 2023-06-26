//go:build release

package mtls

// OverrideCertificateParser does not do anything in release builds
func OverrideCertificateParser(_ CertificateParser) {
	log.Warn("Override certificate parser must not be called in production code")
}
