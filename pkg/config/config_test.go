package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultingReadConfig(t *testing.T) {
	conf, err := readConfigsImpl(
		"does-not-exist-and-will-be-defaulted/central-config.yaml",
		"does-not-exist-and-will-be-defaulted/central-external-db.yaml")
	assert.NoError(t, err)
	assert.NoError(t, conf.validate())
	assert.True(t, *conf.Maintenance.Compaction.Enabled)
	assert.Equal(t, defaultDBSource, conf.CentralDB.Source)
}

func TestReadConfig(t *testing.T) {
	const centralConfig = `
maintenance:
  safeMode: false
  compaction:
    enabled: false
    bucketFillFraction: .5
    freeFractionThreshold: 0.75`

	testCases := []struct {
		title             string
		centralConfigPath string
		centralConfig     string
		extDBConfigPath   string
		extDBConfig       string
		compactionEnabled bool
		expectedError     string
	}{
		{
			title:         "valid config",
			centralConfig: centralConfig,
			extDBConfig: `
centralDB:
  source: >
    host=fakehost
    port=5432
    user=fakeuser`,
			compactionEnabled: false,
		},
		{
			title:         "malformed config",
			centralConfig: centralConfig,
			extDBConfig: `
centralDB:
  source
    host=fakehost
    port=5432
    user=fakeuser`,
			expectedError: "cannot unmarshal string into Go struct",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, dir+"/central.yaml", tc.centralConfig)
			writeFile(t, dir+"/external-db.yaml", tc.extDBConfig)

			conf, err := readConfigsImpl(dir+"/central.yaml", dir+"/external-db.yaml")
			if tc.expectedError == "" {
				assert.NoError(t, err)
				assert.NoError(t, conf.validate())
				assert.Equal(t, *conf.Maintenance.Compaction.Enabled, tc.compactionEnabled)
				assert.Contains(t, conf.CentralDB.Source, "fake")
			} else {
				assert.ErrorContains(t, err, tc.expectedError)
			}
		})
	}
}

func writeFile(t *testing.T, filename, contents string) {
	err := os.WriteFile(filename, []byte(contents), 0644)
	require.NoError(t, err)
}
