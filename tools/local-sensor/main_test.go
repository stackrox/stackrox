package main

import (
	"os"
	"testing"

	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/sensor/debugger/certs"
	"github.com/stackrox/rox/sensor/debugger/k8s"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureRealConnectionCerts(t *testing.T) {
	cases := map[string]struct {
		certFile      string
		keyFile       string
		caFile        string
		cacheDir      string
		expectErr     bool
		expectCache   string
	}{
		"skips k8s secret fetch when all mTLS env vars are set": {
			certFile:    "/tmp/sensor-cert.pem",
			keyFile:     "/tmp/sensor-key.pem",
			caFile:      "/tmp/ca-cert.pem",
			expectCache: "tmp/.local-sensor-cache",
		},
		"preserves preconfigured ROX_CERTIFICATE_CACHE_DIR": {
			certFile:    "/tmp/sensor-cert.pem",
			keyFile:     "/tmp/sensor-key.pem",
			caFile:      "/tmp/ca-cert.pem",
			cacheDir:    "/custom/cache/dir",
			expectCache: "/custom/cache/dir",
		},
		"fetches from k8s when mTLS env vars are incomplete": {
			certFile:  "/tmp/sensor-cert.pem",
			keyFile:   "",
			caFile:    "/tmp/ca-cert.pem",
			expectErr: true,
		},
		"fetches from k8s when no mTLS env vars are set": {
			expectErr: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Setenv(mtls.CertFilePathEnvName, tc.certFile)
			t.Setenv(mtls.KeyFileEnvName, tc.keyFile)
			t.Setenv(mtls.CAFileEnvName, tc.caFile)
			t.Setenv("ROX_CERTIFICATE_CACHE_DIR", tc.cacheDir)

			cli := k8s.MakeFakeClient()
			err := ensureRealConnectionCerts(cli, localSensorConfig{
				Namespace: certs.DefaultNamespace,
			})
			if tc.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "failed to retrieve sensor's certificates")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expectCache, os.Getenv("ROX_CERTIFICATE_CACHE_DIR"))
		})
	}
}
