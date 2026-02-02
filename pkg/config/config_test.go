package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
	testCases := []struct {
		title             string
		centralConfigPath string
		extDBConfigPath   string
		compactionEnabled bool
		isValid           bool
	}{
		{
			title:             "valid config",
			centralConfigPath: "testdata/valid_case/central-config.yaml",
			extDBConfigPath:   "testdata/valid_case/central-external-db.yaml",
			compactionEnabled: false,
			isValid:           true,
		},
		{
			title:             "malformed config",
			centralConfigPath: "testdata/malformed_case/central-config.yaml",
			extDBConfigPath:   "testdata/malformed_case/central-external-db.yaml",
			isValid:           false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			conf, err := readConfigsImpl(tc.centralConfigPath, tc.extDBConfigPath)
			if tc.isValid {
				assert.NoError(t, err)
				assert.NoError(t, conf.validate())
				assert.Equal(t, *conf.Maintenance.Compaction.Enabled, tc.compactionEnabled)
				assert.Contains(t, conf.CentralDB.Source, "fake")
			} else {
				assert.Error(t, err)
			}
		})
	}
}
