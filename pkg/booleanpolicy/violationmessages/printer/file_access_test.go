package printer

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
)

func getAlertFileViolation(msg string, access *storage.FileAccess) *storage.Alert_Violation {
	if access == nil {
		return &storage.Alert_Violation{
			Type:    storage.Alert_Violation_FILE_ACCESS,
			Message: msg,
		}
	}
	return &storage.Alert_Violation{
		Type:    storage.Alert_Violation_FILE_ACCESS,
		Message: msg,
		MessageAttributes: &storage.Alert_Violation_FileAccess{
			FileAccess: access,
		},
	}
}

func TestUpdateFileAccessMessage(t *testing.T) {
	testCases := []struct {
		desc     string
		activity *storage.FileAccess
		expected string
	}{
		{
			desc:     "nil file access",
			activity: nil,
			expected: "",
		},
		{
			desc: "single file activity",
			activity: &storage.FileAccess{
				File: &storage.FileAccess_File{
					ActualPath: "/etc/passwd",
				},
				Operation: storage.FileAccess_OPEN,
				Process: &storage.ProcessIndicator{
					Signal: &storage.ProcessSignal{
						Name: "cat",
					},
				},
			},
			expected: "'/etc/passwd' opened writable",
		},
		{
			desc: "file CREATE operation",
			activity: &storage.FileAccess{
				File:      &storage.FileAccess_File{ActualPath: "/tmp/new_file"},
				Operation: storage.FileAccess_CREATE,
				Process:   &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "touch"}},
			},
			expected: "'/tmp/new_file' created",
		},
		{
			desc: "file UNLINK operation",
			activity: &storage.FileAccess{
				File:      &storage.FileAccess_File{ActualPath: "/tmp/old_file"},
				Operation: storage.FileAccess_UNLINK,
				Process:   &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "rm"}},
			},
			expected: "'/tmp/old_file' deleted",
		},
		{
			desc: "file RENAME operation",
			activity: &storage.FileAccess{
				File:      &storage.FileAccess_File{ActualPath: "/tmp/renamed_file"},
				Operation: storage.FileAccess_RENAME,
				Process:   &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "mv"}},
			},
			expected: "'/tmp/renamed_file' renamed",
		},
		{
			desc: "file PERMISSION_CHANGE operation",
			activity: &storage.FileAccess{
				File:      &storage.FileAccess_File{ActualPath: "/tmp/chmod_file"},
				Operation: storage.FileAccess_PERMISSION_CHANGE,
				Process:   &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "chmod"}},
			},
			expected: "'/tmp/chmod_file' permission changed",
		},
		{
			desc: "file OWNERSHIP_CHANGE operation",
			activity: &storage.FileAccess{
				File:      &storage.FileAccess_File{ActualPath: "/tmp/chown_file"},
				Operation: storage.FileAccess_OWNERSHIP_CHANGE,
				Process:   &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "chown"}},
			},
			expected: "'/tmp/chown_file' ownership changed",
		},
		{
			desc: "nil file path handling",
			activity: &storage.FileAccess{
				File:      nil,
				Operation: storage.FileAccess_OPEN,
				Process: &storage.ProcessIndicator{
					Signal: &storage.ProcessSignal{Name: "test"},
				},
			},
			expected: "'" + UNKNOWN_FILE + "' opened writable",
		},
		{
			desc: "nil process handling",
			activity: &storage.FileAccess{
				File:      &storage.FileAccess_File{ActualPath: "/test/file"},
				Operation: storage.FileAccess_OPEN,
				Process:   nil,
			},
			expected: "'/test/file' opened writable",
		},
		{
			desc: "nil process signal handling",
			activity: &storage.FileAccess{
				File:      &storage.FileAccess_File{ActualPath: "/test/file"},
				Operation: storage.FileAccess_OPEN,
				Process: &storage.ProcessIndicator{
					Signal: nil,
				},
			},
			expected: "'/test/file' opened writable",
		},
		{
			desc: "empty file path",
			activity: &storage.FileAccess{
				File:      &storage.FileAccess_File{ActualPath: ""},
				Operation: storage.FileAccess_OPEN,
				Process: &storage.ProcessIndicator{
					Signal: &storage.ProcessSignal{Name: "test"},
				},
			},
			expected: "'" + UNKNOWN_FILE + "' opened writable",
		},
		{
			desc: "Use EffectivePath if ActualPath is empty",
			activity: &storage.FileAccess{
				File:      &storage.FileAccess_File{ActualPath: "", EffectivePath: "/test/file"},
				Operation: storage.FileAccess_OPEN,
				Process: &storage.ProcessIndicator{
					Signal: &storage.ProcessSignal{Name: "test"},
				},
			},
			expected: "'/test/file' opened writable",
		},
		{
			desc: "empty process name",
			activity: &storage.FileAccess{
				File:      &storage.FileAccess_File{ActualPath: "/test/file"},
				Operation: storage.FileAccess_OPEN,
				Process: &storage.ProcessIndicator{
					Signal: &storage.ProcessSignal{Name: ""},
				},
			},
			expected: "'/test/file' opened writable",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			violation := getAlertFileViolation("", tc.activity)
			UpdateFileAccessAlertViolationMessage(violation)
			protoassert.Equal(t, getAlertFileViolation(tc.expected, tc.activity), violation)
		})
	}
}
