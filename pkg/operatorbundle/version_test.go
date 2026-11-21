package operatorbundle

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func bundle(version string) Bundle {
	return Bundle{Version: version, CSVName: "pkg.v" + version}
}

func TestSelectLatestPatch(t *testing.T) {
	tests := map[string]struct {
		installed   string
		candidates  []Bundle
		wantVersion string // "" => expect no candidate
		wantErr     bool
	}{
		"picks highest patch within same major.minor": {
			installed:   "1.4.0",
			candidates:  []Bundle{bundle("1.4.1"), bundle("1.4.3"), bundle("1.4.2")},
			wantVersion: "1.4.3",
		},
		"ignores higher minor": {
			installed:   "1.4.0",
			candidates:  []Bundle{bundle("1.5.0"), bundle("1.4.1")},
			wantVersion: "1.4.1",
		},
		"ignores higher major": {
			installed:   "1.4.0",
			candidates:  []Bundle{bundle("2.0.0"), bundle("1.4.1")},
			wantVersion: "1.4.1",
		},
		"ignores equal version": {
			installed:   "1.4.2",
			candidates:  []Bundle{bundle("1.4.2")},
			wantVersion: "",
		},
		"ignores older patch": {
			installed:   "1.4.5",
			candidates:  []Bundle{bundle("1.4.2"), bundle("1.4.4")},
			wantVersion: "",
		},
		"no candidates in same major.minor": {
			installed:   "1.4.0",
			candidates:  []Bundle{bundle("1.5.1"), bundle("2.0.0")},
			wantVersion: "",
		},
		"skips unparseable candidate but keeps valid one": {
			installed:   "1.4.0",
			candidates:  []Bundle{bundle("not-a-version"), bundle("1.4.7")},
			wantVersion: "1.4.7",
		},
		"falls back to version_original when version empty": {
			installed:   "1.4.0",
			candidates:  []Bundle{{VersionOriginal: "1.4.9"}},
			wantVersion: "1.4.9",
		},
		"unparseable installed version errors": {
			installed:  "garbage",
			candidates: []Bundle{bundle("1.4.1")},
			wantErr:    true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, reason, err := SelectLatestPatch(tc.installed, tc.candidates)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tc.wantVersion == "" {
				assert.Nil(t, got)
				assert.NotEmpty(t, reason)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tc.wantVersion, bundleVersionString(got))
		})
	}
}
