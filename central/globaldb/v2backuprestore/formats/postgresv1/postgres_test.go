package postgresv1

import (
	"bytes"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestCheckMigrationVersion(t *testing.T) {
	// Get the current minimum supported version to use in tests
	minSupportedVersion := migrations.MinimumSupportedDBVersionSeqNum()

	tests := []struct {
		name        string
		version     migrations.MigrationVersion
		wantErr     bool
		errContains string
	}{
		{
			name: "success - version at minimum",
			version: migrations.MigrationVersion{
				MainVersion: "4.0.0",
				SeqNum:      minSupportedVersion,
			},
			wantErr: false,
		},
		{
			name: "success - version above minimum",
			version: migrations.MigrationVersion{
				MainVersion: "4.5.0",
				SeqNum:      minSupportedVersion + 10,
			},
			wantErr: false,
		},
		{
			name: "success - version well above minimum",
			version: migrations.MigrationVersion{
				MainVersion: "5.0.0",
				SeqNum:      minSupportedVersion + 100,
			},
			wantErr: false,
		},
		{
			name: "error - version below minimum",
			version: migrations.MigrationVersion{
				MainVersion: "3.75.0",
				SeqNum:      minSupportedVersion - 1,
			},
			wantErr:     true,
			errContains: "no longer supported",
		},
		{
			name: "error - version well below minimum",
			version: migrations.MigrationVersion{
				MainVersion: "3.0.0",
				SeqNum:      100,
			},
			wantErr:     true,
			errContains: "no longer supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal the version to YAML
			data, err := yaml.Marshal(tt.version)
			require.NoError(t, err)

			// Create a reader with the marshaled data
			reader := bytes.NewReader(data)
			size := int64(len(data))

			// Call the function under test
			err = checkMigrationVersion(nil, reader, size)

			// Assert results
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCheckMigrationVersion_InvalidYAML(t *testing.T) {
	// Test with invalid YAML data
	invalidYAML := []byte("not: valid: yaml: data: [")
	reader := bytes.NewReader(invalidYAML)
	size := int64(len(invalidYAML))

	err := checkMigrationVersion(nil, reader, size)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "yaml")
}

func TestCheckMigrationVersion_ReadError(t *testing.T) {
	// Test with a reader that returns an error
	errReader := &errorReader{err: errors.New("read error")}
	size := int64(100)

	err := checkMigrationVersion(nil, errReader, size)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "read error")
}

// errorReader is a helper type that always returns an error when Read is called
type errorReader struct {
	err error
}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}
