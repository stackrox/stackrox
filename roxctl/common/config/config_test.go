package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestReadConfig(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	store := configStore{path: cfgPath}

	// Reading from a non-existing file should simply return an empty config and no error.
	cfg, err := store.Read()
	assert.NoError(t, err)
	assert.Equal(t, &RoxctlConfig{CentralConfigs: map[CentralURL]*CentralConfig{}}, cfg)

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

	// Write some invalid JSON in the config file.
	require.NoError(t, os.WriteFile(cfgPath, []byte(`{"some":"value"`), 0644))
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

func TestDetermineConfigPath(t *testing.T) {
	configDir := t.TempDir()
	runtimeDir := t.TempDir()
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)
	homeDir = filepath.Join(homeDir, ".roxctl")
	t.Setenv(env.ConfigDirEnv.EnvVar(), configDir)
	t.Setenv("XDG_RUNTIME_DIR", runtimeDir)

	// ROX_CONFIG_DIR should be used instead of XDG_RUNTIME_DIR.
	dir, err := determineConfigDir()
	assert.NoError(t, err)
	assert.Equal(t, configDir, dir)

	// If only XDF_RUNTIME_DIR is set, it should be used.
	t.Setenv(env.ConfigDirEnv.EnvVar(), "")
	dir, err = determineConfigDir()
	assert.NoError(t, err)
	assert.Equal(t, runtimeDir, dir)

	// If no environment variable is set, the .roxctl dir within the homedir should be used.
	t.Setenv("XDG_RUNTIME_DIR", "")
	dir, err = determineConfigDir()
	assert.NoError(t, err)
	assert.Equal(t, homeDir, dir)
}

func TestDetermineConfigDirPermissionDenied(t *testing.T) {
	// This test verifies that when directory creation fails (e.g., permission denied),
	// a clear filesystem error is returned (not wrapped as a NoCredentials error).

	testDir := t.TempDir()

	// Create a file where HOME/.roxctl would be, which will cause mkdir to fail
	homeFile := filepath.Join(testDir, "homefile")
	require.NoError(t, os.WriteFile(homeFile, []byte("not a directory"), 0600))

	// Set HOME to the file path - this will cause MkdirAll to fail when trying to create .roxctl
	t.Setenv("HOME", homeFile)
	t.Setenv(env.ConfigDirEnv.EnvVar(), "")
	t.Setenv("XDG_RUNTIME_DIR", "")

	expectedConfigPath := filepath.Join(homeFile, ".roxctl")

	_, err := determineConfigDir()
	require.Error(t, err)

	// Verify the error is NOT wrapped as a NoCredentials error
	assert.False(t, errors.Is(err, errox.NoCredentials),
		"filesystem error should not be wrapped as NoCredentials")

	// Verify the error mentions the config path that failed
	errMsg := err.Error()
	assert.Contains(t, errMsg, expectedConfigPath, "error should mention the config path that failed")

	// Verify the error does NOT contain authentication guidance text
	assert.NotContains(t, errMsg, "No authentication credentials are available",
		"filesystem error should not include auth guidance")
	assert.NotContains(t, errMsg, "--password",
		"filesystem error should not suggest auth methods")
	assert.NotContains(t, errMsg, "ROX_API_TOKEN",
		"filesystem error should not suggest auth methods")
}

func TestEnsureRoxctlConfigFilePathExistsPermissionDenied(t *testing.T) {
	// This test verifies that ensureRoxctlConfigFilePathExists returns
	// a clear filesystem error when directory creation fails (not wrapped as NoCredentials).

	testDir := t.TempDir()

	// Create a file that blocks directory creation
	blockingFile := filepath.Join(testDir, "blocking")
	require.NoError(t, os.WriteFile(blockingFile, []byte("not a directory"), 0600))

	// Try to create config file path under the blocking file
	configDirPath := filepath.Join(blockingFile, "subdir")

	_, err := ensureRoxctlConfigFilePathExists(configDirPath)
	require.Error(t, err)

	// Verify the error is NOT wrapped as a NoCredentials error
	assert.False(t, errors.Is(err, errox.NoCredentials),
		"filesystem error should not be wrapped as NoCredentials")

	// Verify the error mentions the config path that failed
	errMsg := err.Error()
	assert.Contains(t, errMsg, configDirPath, "error should mention the config path that failed")

	// Verify the error does NOT contain authentication guidance text
	assert.NotContains(t, errMsg, "No authentication credentials are available",
		"filesystem error should not include auth guidance")
	assert.NotContains(t, errMsg, "--password",
		"filesystem error should not suggest auth methods")
	assert.NotContains(t, errMsg, "ROX_API_TOKEN",
		"filesystem error should not suggest auth methods")
}
