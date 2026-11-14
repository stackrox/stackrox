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

func TestValidReadConfig(t *testing.T) {
	const centralConfig = `
maintenance:
  safeMode: false
  compaction:
    enabled: false
    bucketFillFraction: .5
    freeFractionThreshold: 0.75`

	const extDBConfig = `
centralDB:
  source: >
    host=fakehost
    port=5432
    user=fakeuser`

	conf, err := readConfigsT(t, centralConfig, extDBConfig)

	assert.NoError(t, err)
	assert.NoError(t, conf.validate())
	assert.False(t, *conf.Maintenance.Compaction.Enabled)
	assert.Contains(t, conf.CentralDB.Source, "fake")
}

func TestInvalidReadConfig(t *testing.T) {
	const centralConfig = `
maintenance:
  safeMode: false
  compaction:
    enabled: false
    bucketFillFraction: .5
    freeFractionThreshold: 0.75`

	const extDBConfig = `
centralDB:
  source
    host=fakehost
    port=5432
    user=fakeuser`

	conf, err := readConfigsT(t, centralConfig, extDBConfig)

	assert.ErrorContains(t, err, "cannot unmarshal string into Go struct")
	assert.Nil(t, conf)
}

func readConfigsT(t *testing.T, centralConfig, extDBConfig string) (*Config, error) {
	dir := t.TempDir()
	writeFile(t, dir+"/central.yaml", centralConfig)
	writeFile(t, dir+"/external-db.yaml", extDBConfig)

	return readConfigsImpl(dir+"/central.yaml", dir+"/external-db.yaml")
}

func writeFile(t *testing.T, filename, contents string) {
	err := os.WriteFile(filename, []byte(contents), 0644)
	require.NoError(t, err)
}
