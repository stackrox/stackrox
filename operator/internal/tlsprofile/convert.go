package tlsprofile

import (
	"fmt"
	"strings"

	configv1 "github.com/openshift/api/config/v1"
	libgocrypto "github.com/openshift/library-go/pkg/crypto"
)

// versionToOpenSSL maps OpenShift API TLSProtocolVersion values to OpenSSL
// version strings. This format is understood by ROX_TLS_MIN_VERSION,
// PostgreSQL ssl_min_protocol_version, and OpenSSL-based runtimes alike.
var versionToOpenSSL = map[configv1.TLSProtocolVersion]string{
	configv1.VersionTLS10: "TLSv1.0",
	configv1.VersionTLS11: "TLSv1.1",
	configv1.VersionTLS12: "TLSv1.2",
	configv1.VersionTLS13: "TLSv1.3",
}

// tls13Ciphers is the set of TLS 1.3 cipher names in both OpenSSL and IANA
// conventions. They appear in the OpenShift TLS profile spec alongside TLS 1.2
// ciphers but must be excluded from ROX_TLS_CIPHER_SUITES and
// ROX_OPENSSL_TLS_CIPHER_SUITES because Go and OpenSSL handle them separately.
var tls13Ciphers = map[string]bool{
	// OpenSSL / OpenShift API format (as returned by configv1.TLSProfiles).
	"TLS_AES_128_GCM_SHA256":       true,
	"TLS_AES_256_GCM_SHA384":       true,
	"TLS_CHACHA20_POLY1305_SHA256": true,
}

func convertMinVersion(apiVersion configv1.TLSProtocolVersion) (string, error) {
	if v, ok := versionToOpenSSL[apiVersion]; ok {
		return v, nil
	}
	return "", fmt.Errorf("unsupported TLS version %q", apiVersion)
}

// convertCiphersToIANA converts OpenSSL cipher names (as provided by the
// OpenShift TLS profile API) to a comma-separated IANA string for
// ROX_TLS_CIPHER_SUITES. TLS 1.3 ciphers and ciphers unknown to library-go
// are skipped (they are handled separately by Go).
func convertCiphersToIANA(opensslCiphers []string) string {
	tls12Only := filterTLS13(opensslCiphers)
	iana := libgocrypto.OpenSSLToIANACipherSuites(tls12Only)
	return strings.Join(iana, ",")
}

// convertCiphersToOpenSSL produces a colon-separated OpenSSL cipher string
// for ROX_OPENSSL_TLS_CIPHER_SUITES. TLS 1.3 ciphers are excluded because
// OpenSSL and PostgreSQL configure them separately. The input is already in
// OpenSSL format from the API, so this is essentially a filter + join.
func convertCiphersToOpenSSL(opensslCiphers []string) string {
	return strings.Join(filterTLS13(opensslCiphers), ":")
}

func filterTLS13(ciphers []string) []string {
	result := make([]string, 0, len(ciphers))
	for _, c := range ciphers {
		if !tls13Ciphers[c] {
			result = append(result, c)
		}
	}
	return result
}
