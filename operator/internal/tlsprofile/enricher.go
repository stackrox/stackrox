package tlsprofile

import (
	"context"

	"github.com/stackrox/rox/operator/internal/values/translation"
	"github.com/stackrox/rox/pkg/k8sutil"
	"helm.sh/helm/v3/pkg/chartutil"
)

type enricher struct {
	profile *TLSProfile
}

var _ translation.Enricher = &enricher{}

// NewEnricher returns an Enricher that injects TLS profile environment
// variables into the Helm values. When profile is nil (legacy mode),
// no environment variables are injected and services use their compiled-in
// defaults.
func NewEnricher(profile *TLSProfile) translation.Enricher {
	return &enricher{profile: profile}
}

func (e *enricher) Enrich(_ context.Context, _ k8sutil.Object, vals chartutil.Values) (chartutil.Values, error) {
	profile := e.profile
	if profile == nil {
		return vals, nil
	}

	tlsVals := chartutil.Values{
		"customize": map[string]interface{}{
			"envVars": map[string]interface{}{
				"ROX_TLS_MIN_VERSION":           profile.MinVersion,
				"ROX_TLS_CIPHER_SUITES":         profile.CipherSuites,
				"ROX_OPENSSL_TLS_CIPHER_SUITES": profile.OpenSSLCiphers,
			},
		},
	}

	// CoalesceTables gives precedence to vals (the first argument), so
	// user-specified customize.envVars override Operator-injected values.
	return chartutil.CoalesceTables(vals, tlsVals), nil
}
