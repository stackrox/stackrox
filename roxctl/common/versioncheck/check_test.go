package versioncheck

import (
	"bytes"
	"testing"

	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
)

func TestCheckAndWarn(t *testing.T) {
	cases := map[string]struct {
		localVersion   string
		centralVersion string
		expectWarning  bool
		warnContains   string
	}{
		"same version": {
			localVersion:   "4.8.0",
			centralVersion: "4.8.0",
			expectWarning:  false,
		},
		"same XY different patch": {
			localVersion:   "4.8.0",
			centralVersion: "4.8.1",
			expectWarning:  false,
		},
		"compatible behind within range": {
			localVersion:   "4.8.0",
			centralVersion: "4.6.0",
			expectWarning:  false,
		},
		"compatible ahead within range": {
			localVersion:   "4.8.0",
			centralVersion: "4.10.0",
			expectWarning:  false,
		},
		"incompatible behind": {
			localVersion:   "4.8.0",
			centralVersion: "4.3.0",
			expectWarning:  true,
			warnContains:   "too new",
		},
		"incompatible ahead": {
			localVersion:   "4.8.0",
			centralVersion: "4.15.0",
			expectWarning:  true,
			warnContains:   "too old",
		},
		"invalid central version": {
			localVersion:   "4.8.0",
			centralVersion: "invalid",
			expectWarning:  false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			testutils.SetMainVersion(t, tc.localVersion)

			var buf bytes.Buffer
			result := checkAndWarn(tc.centralVersion, &buf)

			assert.Equal(t, tc.expectWarning, result, "return value mismatch")
			if tc.warnContains != "" {
				assert.Contains(t, buf.String(), tc.warnContains)
				assert.Contains(t, buf.String(), "Compatible Centrals:")
			} else {
				assert.Empty(t, buf.String())
			}
		})
	}
}
