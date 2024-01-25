package resources

import (
	goerrors "errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessInput(t *testing.T) {
	cases := map[string]struct {
		inputFolderPath1      string
		treatWarningsAsErrors bool
		stopOnFirstError      bool
		expectedWarn          string
		expectedErr           string
	}{
		"Not existing input folder paths should result in error": {
			inputFolderPath1: "/tmp/xxx",
			expectedWarn:     "",
			expectedErr:      "the path \"/tmp/xxx\" does not exist",
		},
		"Inputs with no resources should not result in general NP-Guard error": {
			inputFolderPath1:      "testdata/empty-yamls",
			treatWarningsAsErrors: false,
			expectedWarn:          "unable to decode",
			expectedErr:           "",
		},
		"Inputs with no resources should result in general NP-Guard error when run with --fail": {
			inputFolderPath1:      "testdata/empty-yamls",
			treatWarningsAsErrors: true,
			expectedWarn:          "",
			expectedErr:           "Object 'Kind' is missing in",
		},
		"Treating warnings as errors should result in error of type 'npg.ErrWarnings'": {
			inputFolderPath1:      "testdata/contains-invalid-doc",
			treatWarningsAsErrors: true,
			expectedWarn:          "",
			expectedErr:           "Object 'Kind' is missing in",
		},
		"Warnings on invalid input docs without using strict flag should not be treated as errors": {
			inputFolderPath1:      "testdata/contains-invalid-doc",
			treatWarningsAsErrors: false,
			expectedWarn:          "Object 'Kind' is missing in",
			expectedErr:           "",
		},
		"Stop on first error with malformed yaml inputs without strict setting should not stop with general NP-Guard error": {
			inputFolderPath1:      "testdata/dirty", // yaml document malformed
			stopOnFirstError:      true,
			treatWarningsAsErrors: false,
			expectedWarn:          "error parsing",
			expectedErr:           "",
		},
		"Stop on first error with malformed yaml inputs with strict setting should stop with general NP-Guard error as error": {
			inputFolderPath1:      "testdata/dirty", // yaml document malformed
			stopOnFirstError:      true,
			treatWarningsAsErrors: true,
			expectedWarn:          "",
			expectedErr:           "error parsing testdata/dirty",
		},
		"Testing Diff between two dirs should run successfully without errors": {
			inputFolderPath1: "testdata/valid-minimal",
			expectedWarn:     "",
			expectedErr:      "",
		},
	}

	for name, tt := range cases {
		tt := tt
		t.Run(name, func(t *testing.T) {
			_, warns, errs := GetK8sInfos(tt.inputFolderPath1, tt.stopOnFirstError, tt.treatWarningsAsErrors)
			// Joining errors for easier assertions
			mergedWarns := goerrors.Join(warns...)
			mergedErrors := goerrors.Join(errs...)

			if tt.expectedWarn != "" {
				require.Error(t, mergedWarns)
				assert.ErrorContains(t, mergedWarns, tt.expectedWarn)
			} else {
				assert.NoError(t, mergedWarns, "Received unexpected warning")
			}
			if tt.expectedErr != "" {
				require.Error(t, mergedErrors)
				assert.ErrorContains(t, mergedErrors, tt.expectedErr)
			} else {
				assert.NoError(t, mergedErrors)
			}
		})
	}
}
