package tlsprofile

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chartutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var intermediateProfile = &TLSProfile{
	MinVersion:     "TLSv1.2",
	CipherSuites:   "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
	OpenSSLCiphers: "ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384",
}

func TestEnricher_InjectsEnvVars(t *testing.T) {
	e := NewEnricher(intermediateProfile)

	vals := chartutil.Values{}
	result, err := e.Enrich(context.Background(), &unstructured.Unstructured{}, vals)
	require.NoError(t, err)

	envVars, err := result.Table("customize.envVars")
	require.NoError(t, err)

	assert.Equal(t, "TLSv1.2", envVars["ROX_TLS_MIN_VERSION"])
	assert.Contains(t, envVars["ROX_TLS_CIPHER_SUITES"], "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384")
	assert.Contains(t, envVars["ROX_OPENSSL_TLS_CIPHER_SUITES"], "ECDHE-ECDSA-AES256-GCM-SHA384")
}

func TestEnricher_UserValuesOverrideInjected(t *testing.T) {
	e := NewEnricher(intermediateProfile)

	vals := chartutil.Values{
		"customize": map[string]interface{}{
			"envVars": map[string]interface{}{
				"ROX_TLS_MIN_VERSION": "TLSv1.3",
			},
		},
	}
	result, err := e.Enrich(context.Background(), &unstructured.Unstructured{}, vals)
	require.NoError(t, err)

	envVars, err := result.Table("customize.envVars")
	require.NoError(t, err)

	assert.Equal(t, "TLSv1.3", envVars["ROX_TLS_MIN_VERSION"],
		"user-specified value should take precedence over injected value")
	assert.Contains(t, envVars["ROX_TLS_CIPHER_SUITES"], "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
		"non-overridden values should still be injected")
}

func TestEnricher_NilProfileNoInjection(t *testing.T) {
	e := NewEnricher(nil)

	vals := chartutil.Values{"existing": "value"}
	result, err := e.Enrich(context.Background(), &unstructured.Unstructured{}, vals)
	require.NoError(t, err)

	_, err = result.Table("customize.envVars")
	assert.Error(t, err, "no envVars should be injected in legacy mode")
	assert.Equal(t, "value", result["existing"])
}
