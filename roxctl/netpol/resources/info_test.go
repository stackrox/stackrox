package resources

import (
	"testing"

	"github.com/stackrox/rox/roxctl/common/npg"
)

func TestProcessInput(t *testing.T) {
	cases := map[string]struct {
		inputFolderPath1      string
		treatWarningsAsErrors bool
		stopOnFirstError      bool

		expectedWarn []string
		expectedErr  []string
	}{
		"Not existing input folder paths should result in error": {
			inputFolderPath1: "/tmp/xxx",
			expectedWarn:     []string{},
			expectedErr:      []string{"the path \"/tmp/xxx\" does not exist"},
		},
		"Warnings should not produce error when treatWarningsAsErrors is false": {
			inputFolderPath1:      "testdata/empty-yamls",
			treatWarningsAsErrors: false,
			expectedWarn: []string{
				"unable to decode \"testdata/empty-yamls/empty.yaml\"",
				"unable to decode \"testdata/empty-yamls/empty2.yaml\"",
			},
			expectedErr: []string{},
		},
		"Presence of warnings with stopOnFirstError set to true should not produce error": {
			inputFolderPath1:      "testdata/dirty", // yaml document malformed
			stopOnFirstError:      true,
			treatWarningsAsErrors: false,
			expectedWarn: []string{
				"error parsing testdata/dirty/backend.yaml",
				"error parsing testdata/dirty/frontend.yaml",
			},
			expectedErr: []string{},
		},
		"Presence of warnings with stopOnFirstError true and treatWarningsAsErrors should stop on first warning and produce no marker errors": {
			inputFolderPath1:      "testdata/dirty", // yaml document malformed
			stopOnFirstError:      true,
			treatWarningsAsErrors: true,
			expectedWarn: []string{
				"error parsing testdata/dirty/backend.yaml",
			},
			expectedErr: []string{},
		},
		"Location with valid yamls should produce no warnings and no errors": {
			inputFolderPath1: "testdata/valid-minimal",
			expectedWarn:     []string{},
			expectedErr:      []string{},
		},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			_, warns, errs := GetK8sInfos(tt.inputFolderPath1, tt.stopOnFirstError, tt.treatWarningsAsErrors)
			npg.AssertErrorsContain(t, tt.expectedErr, errs, "errors")
			npg.AssertErrorsContain(t, tt.expectedWarn, warns, "warnings")
		})
	}
}
