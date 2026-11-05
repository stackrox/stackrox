package postgresv1

import (
	"bytes"
	"testing"

	"github.com/pkg/errors"
	grpcerrors "github.com/stackrox/rox/pkg/grpc/errors"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/version"
	versiontest "github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"gopkg.in/yaml.v3"
)

func TestCheckMigrationVersion(t *testing.T) {
	setMainVersion(t, "5.1.0")
	// Get the current minimum supported version to use in tests
	minSupportedVersion, err := migrations.MinimumSupportedDBVersionSeqNum()
	require.NoError(t, err)

	tests := []struct {
		name        string
		version     migrations.MigrationVersion
		wantErr     bool
		errContains string
	}{
		{
			name: "success - version at minimum",
			version: migrations.MigrationVersion{
				MainVersion: "4.10.0",
				SeqNum:      minSupportedVersion,
			},
			wantErr: false,
		},
		{
			name: "success - version above minimum",
			version: migrations.MigrationVersion{
				MainVersion: "4.11.0",
				SeqNum:      migrations.CurrentDBVersionSeqNum() - 1,
			},
			wantErr: false,
		},
		{
			name: "success - version well above minimum",
			version: migrations.MigrationVersion{
				MainVersion: "5.0.0",
				SeqNum:      migrations.CurrentDBVersionSeqNum(),
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

func TestCheckMigrationVersionNewerBackup(t *testing.T) {
	setMainVersion(t, "5.1.0")
	current := migrations.CurrentDBVersionSeqNum()
	for name, tc := range map[string]struct {
		source   string
		sequence int
		wantErr  bool
	}{
		"same version and sequence":           {source: "5.1.0", sequence: current},
		"older version with current sequence": {source: "5.0.0", sequence: current},
		"newer sequence":                      {source: "5.1.0", sequence: current + 1, wantErr: true},
		"newer stream with current sequence":  {source: "5.2.0", sequence: current, wantErr: true},
		"newer patch with current sequence":   {source: "5.1.1", sequence: current, wantErr: true},
	} {
		t.Run(name, func(t *testing.T) {
			err := checkVersion(t, migrations.MigrationVersion{MainVersion: tc.source, SeqNum: tc.sequence})
			if tc.wantErr {
				require.ErrorContains(t, err, "newer than the current version")
				assert.Equal(t, codes.FailedPrecondition, grpcerrors.ErrToGrpcStatus(err).Code())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCheckMigrationVersionProductCompatibility(t *testing.T) {
	setMainVersion(t, "5.4.0")
	minimum, err := migrations.MinimumSupportedForVersion(version.GetMainVersion())
	require.NoError(t, err)
	require.Equal(t, "5.1", minimum.Version)

	for name, tc := range map[string]struct {
		source  string
		blocked bool
	}{
		"N-4 sharing minimum sequence":     {source: "5.0.0", blocked: true},
		"N-3 sharing minimum sequence":     {source: "5.1.0"},
		"N-3 patch and prerelease ignored": {source: "5.1.2-rc.1"},
		"missing source":                   {blocked: true},
		"malformed source":                 {source: "invalid", blocked: true},
	} {
		t.Run(name, func(t *testing.T) {
			err := checkVersion(t, migrations.MigrationVersion{MainVersion: tc.source, SeqNum: minimum.Sequence})
			if tc.blocked {
				require.Error(t, err)
				assert.Equal(t, codes.FailedPrecondition, grpcerrors.ErrToGrpcStatus(err).Code())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCheckMigrationVersionInvalidTarget(t *testing.T) {
	for name, target := range map[string]string{
		"invalid version":  "garbage",
		"missing metadata": "5.5.0",
	} {
		t.Run(name, func(t *testing.T) {
			setMainVersion(t, target)
			err := checkVersion(t, migrations.MigrationVersion{MainVersion: "5.1.0", SeqNum: 227})
			require.ErrorContains(t, err, "determining minimum supported database version for restore")
			assert.Equal(t, codes.Internal, grpcerrors.ErrToGrpcStatus(err).Code())
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

func checkVersion(t *testing.T, ver migrations.MigrationVersion) error {
	t.Helper()
	data, err := yaml.Marshal(ver)
	require.NoError(t, err)
	return checkMigrationVersion(nil, bytes.NewReader(data), int64(len(data)))
}

func setMainVersion(t *testing.T, target string) {
	t.Helper()
	old := version.GetMainVersion()
	t.Cleanup(func() { versiontest.SetMainVersion(t, old) })
	versiontest.SetMainVersion(t, target)
}
