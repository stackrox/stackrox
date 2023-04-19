package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestReadConfig(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	store := configStore{path: cfgPath}

	// Reading from a non-existing file should simply return an empty config and no error.
	cfg, err := store.Read()
	assert.NoError(t, err)
	assert.Empty(t, cfg)

	// Write a config to the file.
	sampleCfg := RoxctlConfig{CentralConfigs: map[string]*CentralConfig{
		"central": {AccessConfig: &CentralAccessConfig{
			AccessToken:  "some-access-token",
			RefreshToken: "some-refresh-token",
		}},
	}}
	rawCfg, err := yaml.Marshal(&sampleCfg)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(cfgPath, rawCfg, 0644))

	// Reading from the config again should yield the same configuration.
	cfg, err = store.Read()
	assert.NoError(t, err)
	assert.Equal(t, sampleCfg, *cfg)

	// Write some non-YAML in the config file.
	require.NoError(t, os.WriteFile(cfgPath, []byte(`XXX`), 0644))

	cfg, err = store.Read()
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestWriteConfig(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	store := configStore{path: cfgPath}

	sampleCfg := RoxctlConfig{CentralConfigs: map[string]*CentralConfig{
		"central": {AccessConfig: &CentralAccessConfig{
			AccessToken:  "some-access-token",
			RefreshToken: "some-refresh-token",
		}},
	}}
	err := store.Write(&sampleCfg)
	assert.NoError(t, err)

	cfg, err := store.Read()
	assert.NoError(t, err)
	assert.Equal(t, sampleCfg, *cfg)
}
