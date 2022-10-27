package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadConfig(t *testing.T) {
	testCases := []struct {
		title             string
		centralConfigPath string
		extDBConfigPath   string
		compactionEnabled bool
		isValid           bool
		isDefault         bool
	}{
		{
			title:             "valid config",
			centralConfigPath: "testdata/valid_case/central-config.yaml",
			extDBConfigPath:   "testdata/valid_case/central-external-db.yaml",
			compactionEnabled: false,
			isValid:           true,
			isDefault:         false,
		},
		{
			title:             "default config",
			centralConfigPath: "testdata/default_case/central-config.yaml",
			extDBConfigPath:   "testdata/default_case/central-external-db.yaml",
			compactionEnabled: true,
			isValid:           true,
			isDefault:         true,
		},
		{
			title:             "malformed config",
			centralConfigPath: "testdata/malformed_case/central-config.yaml",
			extDBConfigPath:   "testdata/malformed_case/central-external-db.yaml",
			isValid:           false,
			isDefault:         false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.title, func(t *testing.T) {
			configPath = tc.centralConfigPath
			dbConfigPath = tc.extDBConfigPath
			conf, err := readConfigs()
			if tc.isValid {
				assert.NoError(t, err)
				require.NoError(t, conf.validate())
				require.Equal(t, *conf.Maintenance.Compaction.Enabled, tc.compactionEnabled)
				if tc.isDefault {
					require.Equal(t, defaultDBSource, conf.CentralDB.Source)
				} else {
					assert.Contains(t, conf.CentralDB.Source, "fake")
				}
			} else {
				assert.Error(t, err)
			}
		})
	}
}
