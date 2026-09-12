package main

import (
	"testing"

	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCertsSecret(t *testing.T) {
	tests := map[string]struct {
		input     string
		namespace string
		name      string
		wantErr   bool
	}{
		"name only defaults to stackrox namespace": {
			input:     "central-tls",
			namespace: "stackrox",
			name:      "central-tls",
		},
		"explicit namespace": {
			input:     "my-ns/my-secret",
			namespace: "my-ns",
			name:      "my-secret",
		},
		"stackrox namespace explicit": {
			input:     "stackrox/central-tls",
			namespace: "stackrox",
			name:      "central-tls",
		},
		"missing namespace before slash": {
			input:   "/secret",
			wantErr: true,
		},
		"missing name after slash": {
			input:   "namespace/",
			wantErr: true,
		},
		"empty string": {
			input:   "",
			wantErr: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ns, n, err := parseCertsSecret(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.namespace, ns)
			assert.Equal(t, tc.name, n)
		})
	}
}

func TestCertsSecretFileMap(t *testing.T) {
	assert.Equal(t, mtls.CACertFileName, certsSecretFileMap["ca.pem"].fileName)
	assert.True(t, certsSecretFileMap["ca.pem"].required)
	assert.Equal(t, mtls.CAKeyFileName, certsSecretFileMap["ca-key.pem"].fileName)
	assert.False(t, certsSecretFileMap["ca-key.pem"].required)
	assert.Equal(t, mtls.ServiceCertFileName, certsSecretFileMap["cert.pem"].fileName)
	assert.True(t, certsSecretFileMap["cert.pem"].required)
	assert.Equal(t, mtls.ServiceKeyFileName, certsSecretFileMap["key.pem"].fileName)
	assert.True(t, certsSecretFileMap["key.pem"].required)
}
