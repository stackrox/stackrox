package renderer

import (
	"fmt"
	"strings"
	"testing"

	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/images/defaults/testutils"
	"github.com/stackrox/stackrox/pkg/k8sutil"
	"github.com/stretchr/testify/assert"
)

func TestRenderTLSSecretsOnly(t *testing.T) {
	config := Config{
		SecretsByteMap: map[string][]byte{
			"ca.pem":              []byte("CA"),
			"ca-key.pem":          []byte("CAKey"),
			"cert.pem":            []byte("CentralCert"),
			"key.pem":             []byte("CentralKey"),
			"scanner-cert.pem":    []byte("ScannerCert"),
			"scanner-key.pem":     []byte("ScannerKey"),
			"scanner-db-cert.pem": []byte("ScannerDBCert"),
			"scanner-db-key.pem":  []byte("ScannerDBKey"),
			"jwt-key.pem":         []byte("JWTKey"),
		},
		K8sConfig: &K8sConfig{
			DeploymentFormat: v1.DeploymentFormat_KUBECTL,
		},
	}

	for _, renderMode := range []mode{centralTLSOnly, scannerTLSOnly} {
		t.Run(fmt.Sprintf("mode=%s", renderMode), func(t *testing.T) {
			contents, err := renderAndExtractSingleFileContents(config, renderMode, testutils.MakeImageFlavorForTest(t))
			assert.NoError(t, err)

			objs, err := k8sutil.UnstructuredFromYAMLMulti(string(contents))
			assert.NoError(t, err)

			assert.NotEmpty(t, objs)
		})
	}
}

func TestRenderScannerOnly(t *testing.T) {
	flavor := testutils.MakeImageFlavorForTest(t)
	config := Config{
		SecretsByteMap: map[string][]byte{
			"ca.pem":              []byte("CA"),
			"ca-key.pem":          []byte("CAKey"),
			"cert.pem":            []byte("CentralCert"),
			"key.pem":             []byte("CentralKey"),
			"scanner-cert.pem":    []byte("ScannerCert"),
			"scanner-key.pem":     []byte("ScannerKey"),
			"scanner-db-cert.pem": []byte("ScannerDBCert"),
			"scanner-db-key.pem":  []byte("ScannerDBKey"),
			"jwt-key.pem":         []byte("JWTKey"),
		},
		K8sConfig: &K8sConfig{
			CommonConfig: CommonConfig{
				MainImage:      flavor.MainImage(),
				ScannerImage:   flavor.ScannerImage(),
				ScannerDBImage: flavor.ScannerDBImage(),
			},
			DeploymentFormat: v1.DeploymentFormat_KUBECTL,
		},
	}

	files, err := render(config, scannerOnly, flavor)
	assert.NoError(t, err)

	for _, f := range files {
		assert.Falsef(t, strings.HasPrefix(f.Name, "central/"), "unexpected file %s in scanner only bundle", f.Name)
	}
}
