package download

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFilenameFromHeader(t *testing.T) {
	tests := map[string]struct {
		headerValue    string
		expectedName   string
		expectError    bool
		expectedErrMsg string
	}{
		"valid with quotes": {
			headerValue:  "attachment; filename=\"report.csv\"",
			expectedName: "report.csv",
		},
		"valid without quotes": {
			headerValue:  "attachment; filename=report.txt",
			expectedName: "report.txt",
		},
		"complex filename": {
			headerValue:  "attachment; filename=\"backup-2024-01-15_v2.tar.gz\"",
			expectedName: "backup-2024-01-15_v2.tar.gz",
		},
		"empty filename with quotes": {
			headerValue:  "attachment; filename=\"\"",
			expectedName: "",
		},
		"missing header": {
			headerValue:    "",
			expectError:    true,
			expectedErrMsg: "missing Content-Disposition header",
		},
		"wrong prefix - inline": {
			headerValue:    "inline; filename=\"data.json\"",
			expectError:    true,
			expectedErrMsg: "failed to determine filename",
		},
		"wrong prefix - just filename": {
			headerValue:    "filename=\"test.txt\"",
			expectError:    true,
			expectedErrMsg: "failed to determine filename",
		},
		"no filename parameter": {
			headerValue:    "attachment; size=1024",
			expectError:    true,
			expectedErrMsg: "failed to determine filename",
		},
		"just attachment": {
			headerValue:    "attachment",
			expectError:    true,
			expectedErrMsg: "failed to determine filename",
		},
		"unicode filename": {
			headerValue:  "attachment; filename=\"文档.txt\"",
			expectedName: "文档.txt",
		},
		"filename with special chars": {
			headerValue:  "attachment; filename=\"file_name-v2.0.tar.gz\"",
			expectedName: "file_name-v2.0.tar.gz",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			header := http.Header{}
			if tc.headerValue != "" {
				header.Set(contentDispositionHeader, tc.headerValue)
			}

			filename, err := ParseFilenameFromHeader(header)

			if tc.expectError {
				require.Error(t, err)
				assert.True(t, errors.Is(err, errox.NotFound), "error should be NotFound type")
				if tc.expectedErrMsg != "" {
					assert.Contains(t, err.Error(), tc.expectedErrMsg)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedName, filename)
			}
		})
	}
}
