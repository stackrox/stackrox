package renderer

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/roxctl/defaults"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
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

	for _, flagValue := range []bool{false, true} {
		for _, renderMode := range []mode{centralTLSOnly, scannerTLSOnly} {
			t.Run(fmt.Sprintf("newExperience=%t,mode=%s", flagValue, renderMode), func(t *testing.T) {
				env := envisolator.NewEnvIsolator(t)
				defer env.RestoreAll()

				env.Setenv(features.CentralInstallationExperience.EnvVar(), strconv.FormatBool(flagValue))
				contents, err := renderAndExtractSingleFileContents(config, renderMode)
				assert.NoError(t, err)

				objs, err := k8sutil.UnstructuredFromYAMLMulti(string(contents))
				assert.NoError(t, err)

				assert.NotEmpty(t, objs)
			})
		}
	}
}

func TestRenderScannerOnly(t *testing.T) {
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
				MainImage:      defaults.MainImage(),
				ScannerImage:   defaults.ScannerImage(),
				ScannerDBImage: defaults.ScannerDBImage(),
			},
			DeploymentFormat: v1.DeploymentFormat_KUBECTL,
		},
	}

	for _, flagValue := range []bool{false, true} {
		t.Run(fmt.Sprintf("newExperience=%t", flagValue), func(t *testing.T) {
			env := envisolator.NewEnvIsolator(t)
			defer env.RestoreAll()

			env.Setenv(features.CentralInstallationExperience.EnvVar(), strconv.FormatBool(flagValue))
			files, err := render(config, scannerOnly, nil)
			assert.NoError(t, err)

			for _, f := range files {
				assert.Falsef(t, strings.HasPrefix(f.Name, "central/"), "unexpected file %s in scanner only bundle", f.Name)
			}
		})
	}
}
