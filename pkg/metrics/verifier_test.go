package metrics

import (
	"crypto/tls"
	"os"
	"testing"

	"github.com/cloudflare/cfssl/helpers"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientCertVerifier(t *testing.T) {
	prometheusCertCN := env.SecureMetricsClientCertCN.Setting()
	cases := map[string]struct {
		certFilePath string
		subjectCN    string
		isError      bool
	}{
		"correct common name,cert signed": {
			certFilePath: "./testdata/client.crt",
			subjectCN:    prometheusCertCN,
			isError:      false,
		},
		"wrong common name,cert signed": {
			certFilePath: "./testdata/client.crt",
			subjectCN:    "fake-CN",
			isError:      true,
		},
		"correct common name,cert unsigned": {
			certFilePath: "./testdata/unsigned-client.crt",
			subjectCN:    prometheusCertCN,
			isError:      true,
		},
		"wrong common name,cert unsigned": {
			certFilePath: "./testdata/unsigned-client.crt",
			subjectCN:    "fake-CN",
			isError:      true,
		},
	}

	for name, c := range cases {
		c := c
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			certFile, err := os.ReadFile(c.certFilePath)
			require.NoError(t, err)
			certs, err := helpers.ParseCertificatesPEM(certFile)
			require.NoError(t, err)
			require.Len(t, certs, 1)

			caFile, err := os.ReadFile("./testdata/ca.pem")
			require.NoError(t, err)
			caPool, err := helpers.PEMToCertPool(caFile)
			require.NoError(t, err)

			tlsVerifier := &clientCertVerifier{
				subjectCN: c.subjectCN,
			}
			tlsConfig := &tls.Config{ClientCAs: caPool}
			err = tlsVerifier.VerifyPeerCertificate(certs[0], nil, tlsConfig)

			if c.isError {
				assert.Error(t, err)
				assert.ErrorIs(t, err, errox.NotAuthorized)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
