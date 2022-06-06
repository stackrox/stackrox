package generate

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/buildinfo"
	buildTestutils "github.com/stackrox/rox/pkg/buildinfo/testutils"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/renderer"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/printer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chartutil"
)

const (
	// Generated test data file.
	backupBundleWithDer = "testdata/stackrox_with_der.zip"
	backupBundleWithPem = "testdata/stackrox_with_pem.zip"

	// Pre-populated sha256 checksum of keys and certificates.
	// To repopulate these values, run `shasum -a 256 <file-name>`
	checksumCaKey  = "ee4ce36941347600a9e520a9f7a12fda569c1c20a8435457e846b61cdc1704fe"
	checksumCaCert = "ec4b9a04bed129018aafa9a791063d64d6ec45c2e7985a77bb17ed9dbbe1ec68"
	checksumJwtKey = "57e27883493f7375671d30aa679789db3dcfcb2221c4dce0a17d83fc64da5c36"
)

func TestRestoreKeysAndCerts(t *testing.T) {
	tmpDir := t.TempDir()

	testutils.SetExampleVersion(t)
	buildTestutils.SetBuildTimestamp(t, time.Now())

	flavorName := defaults.ImageFlavorNameDevelopmentBuild
	if buildinfo.ReleaseBuild {
		flavorName = defaults.ImageFlavorNameStackRoxIORelease
	}
	config := renderer.Config{
		Version:     version.GetMainVersion(),
		ClusterType: storage.ClusterType_KUBERNETES_CLUSTER,
		K8sConfig: &renderer.K8sConfig{
			AppName:          "someApp",
			ImageFlavorName:  flavorName,
			DeploymentFormat: v1.DeploymentFormat_HELM,
			OfflineMode:      false,
		},
	}

	testCases := []struct {
		description  string
		backupBundle string
		testDir      string
		equal        bool
	}{
		{
			description:  "Backup bundle with jwt key in PEM format",
			backupBundle: backupBundleWithPem,
			testDir:      "pem",
			equal:        true,
		},
		{
			description:  "Backup bundle with jwt key in DER format",
			backupBundle: backupBundleWithDer,
			testDir:      "der",
			equal:        true,
		},
		{
			description:  "No backup bundle sanity check",
			backupBundle: "",
			testDir:      "no",
			equal:        false,
		},
	}

	io, _, _, _ := common.TestIO()
	logger := common.NewLogger(io, printer.DefaultColorPrinter())

	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			// Note: This test is not for parallel run.
			config.OutputDir = filepath.Join(tmpDir, testCase.testDir)
			config.BackupBundle = testCase.backupBundle
			require.NoError(t, OutputZip(logger, config))

			// Load values-private.yaml file
			values, err := chartutil.ReadValuesFile(filepath.Join(config.OutputDir, "values-private.yaml"))
			require.NoError(t, err)

			// Verify correctness by comparing sha256 checksum.
			verify := func(path string, checksum string) {
				content, err := values.PathValue(path)
				assert.NoError(t, err)
				require.NotNilf(t, content, "value for %s is missing", path)
				assert.Equal(t, getSha256Sum(content.(string)) == checksum, testCase.equal)
			}
			verify("ca.cert", checksumCaCert)
			verify("ca.key", checksumCaKey)
			verify("central.jwtSigner.key", checksumJwtKey)
		})
	}
}

func getSha256Sum(input string) string {
	shaHash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(shaHash[:])
}
