package mtls

import (
	"testing"
	"time"

	"github.com/cloudflare/cfssl/csr"
	"github.com/cloudflare/cfssl/initca"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_CA_IssueCertForSubject(t *testing.T) {
	tests := map[string]struct {
		opts        []IssueCertOption
		minNotAfter time.Duration
		maxNotAfter time.Duration
	}{
		"regular cert": {
			opts:        nil,
			minNotAfter: 364 * 24 * time.Hour,
			maxNotAfter: 366 * 24 * time.Hour,
		},
		"ephemeral cert hourly expiration": {
			opts:        []IssueCertOption{WithValidityExpiringInHours()},
			minNotAfter: 2 * time.Hour,
			maxNotAfter: 4 * time.Hour,
		},
		"ephemeral cert daily expiration": {
			opts:        []IssueCertOption{WithValidityExpiringInDays()},
			minNotAfter: (2*24 - 1) * time.Hour,
			maxNotAfter: (2*24 + 1) * time.Hour,
		},
	}

	cert, _, key, err := initca.New(&csr.CertificateRequest{
		CN: "Fake CA",
	})
	require.NoError(t, err)

	ca, err := LoadCAForSigning(cert, key)
	require.NoError(t, err)

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := ca.IssueCertForSubject(CentralSubject, tt.opts...)
			require.NoError(t, err)

			notAfter := got.X509Cert.NotAfter
			assert.True(t, notAfter.After(time.Now().Add(tt.minNotAfter)), "expected notAfter=%q to be after %q from now", notAfter, tt.minNotAfter)
			assert.True(t, notAfter.Before(time.Now().Add(tt.maxNotAfter)), "expected notAfter=%q to be before %q from now", notAfter, tt.maxNotAfter)
		})
	}
}
