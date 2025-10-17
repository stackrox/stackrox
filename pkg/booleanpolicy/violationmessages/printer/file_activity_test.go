package printer

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
)

func getAlertFileViolation(msg string, activity []*storage.FileActivity) *storage.Alert_FileActivityViolation {
	return &storage.Alert_FileActivityViolation{
		Message:  msg,
		Activity: activity,
	}
}

func TestFileActivityPrinter(t *testing.T) {
	testCases := []struct {
		desc     string
		activity []*storage.FileActivity
		expected string
	}{
		{
			desc:     "empty activity list",
			activity: nil,
			expected: "",
		},
		{
			desc:     "empty activity slice",
			activity: []*storage.FileActivity{},
			expected: "",
		},
		{
			desc: "single file activity",
			activity: []*storage.FileActivity{
				{
					File: &storage.FileActivity_File{
						Path: "/etc/passwd",
					},
					Operation: storage.FileActivity_OPEN,
					Process: &storage.ProcessIndicator{
						Signal: &storage.ProcessSignal{
							Name: "cat",
						},
					},
				},
			},
			expected: "'/etc/passwd' accessed (OPEN) by cat",
		},
		{
			desc: "multiple activities on same file",
			activity: []*storage.FileActivity{
				{
					File: &storage.FileActivity_File{
						Path: "/etc/passwd",
					},
					Operation: storage.FileActivity_OPEN,
					Process: &storage.ProcessIndicator{
						Signal: &storage.ProcessSignal{
							Name: "cat",
						},
					},
				},
				{
					File: &storage.FileActivity_File{
						Path: "/etc/passwd",
					},
					Operation: storage.FileActivity_WRITE,
					Process: &storage.ProcessIndicator{
						Signal: &storage.ProcessSignal{
							Name: "vim",
						},
					},
				},
			},
			expected: "'/etc/passwd' accessed (OPEN) by cat; '/etc/passwd' accessed (WRITE) by vim",
		},
		{
			desc: "multiple activities on different files",
			activity: []*storage.FileActivity{
				{
					File: &storage.FileActivity_File{
						Path: "/etc/passwd",
					},
					Operation: storage.FileActivity_OPEN,
					Process: &storage.ProcessIndicator{
						Signal: &storage.ProcessSignal{
							Name: "cat",
						},
					},
				},
				{
					File: &storage.FileActivity_File{
						Path: "/etc/shadow",
					},
					Operation: storage.FileActivity_OPEN,
					Process: &storage.ProcessIndicator{
						Signal: &storage.ProcessSignal{
							Name: "grep",
						},
					},
				},
			},
			expected: "'/etc/passwd' accessed (OPEN) by cat; '/etc/shadow' accessed (OPEN) by grep",
		},
		{
			desc: "exactly 10 unique files - should use summary format",
			activity: []*storage.FileActivity{
				{File: &storage.FileActivity_File{Path: "/file1"}, Operation: storage.FileActivity_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc1"}}},
				{File: &storage.FileActivity_File{Path: "/file2"}, Operation: storage.FileActivity_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc2"}}},
				{File: &storage.FileActivity_File{Path: "/file3"}, Operation: storage.FileActivity_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc3"}}},
				{File: &storage.FileActivity_File{Path: "/file4"}, Operation: storage.FileActivity_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc4"}}},
				{File: &storage.FileActivity_File{Path: "/file5"}, Operation: storage.FileActivity_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc5"}}},
				{File: &storage.FileActivity_File{Path: "/file6"}, Operation: storage.FileActivity_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc6"}}},
				{File: &storage.FileActivity_File{Path: "/file7"}, Operation: storage.FileActivity_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc7"}}},
				{File: &storage.FileActivity_File{Path: "/file8"}, Operation: storage.FileActivity_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc8"}}},
				{File: &storage.FileActivity_File{Path: "/file9"}, Operation: storage.FileActivity_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc9"}}},
				{File: &storage.FileActivity_File{Path: "/file10"}, Operation: storage.FileActivity_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc10"}}},
			},
			expected: "10 sensitive files accessed",
		},
		{
			desc: "more than 10 unique files - should use summary format",
			activity: []*storage.FileActivity{
				{File: &storage.FileActivity_File{Path: "/file1"}, Operation: storage.FileActivity_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc1"}}},
				{File: &storage.FileActivity_File{Path: "/file2"}, Operation: storage.FileActivity_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc2"}}},
				{File: &storage.FileActivity_File{Path: "/file3"}, Operation: storage.FileActivity_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc3"}}},
				{File: &storage.FileActivity_File{Path: "/file4"}, Operation: storage.FileActivity_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc4"}}},
				{File: &storage.FileActivity_File{Path: "/file5"}, Operation: storage.FileActivity_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc5"}}},
				{File: &storage.FileActivity_File{Path: "/file6"}, Operation: storage.FileActivity_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc6"}}},
				{File: &storage.FileActivity_File{Path: "/file7"}, Operation: storage.FileActivity_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc7"}}},
				{File: &storage.FileActivity_File{Path: "/file8"}, Operation: storage.FileActivity_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc8"}}},
				{File: &storage.FileActivity_File{Path: "/file9"}, Operation: storage.FileActivity_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc9"}}},
				{File: &storage.FileActivity_File{Path: "/file10"}, Operation: storage.FileActivity_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc10"}}},
				{File: &storage.FileActivity_File{Path: "/file11"}, Operation: storage.FileActivity_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc11"}}},
				{File: &storage.FileActivity_File{Path: "/file12"}, Operation: storage.FileActivity_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc12"}}},
			},
			expected: "12 sensitive files accessed",
		},
		{
			desc: "9 unique files with multiple activities each - should use detailed format",
			activity: []*storage.FileActivity{
				{File: &storage.FileActivity_File{Path: "/file1"}, Operation: storage.FileActivity_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc1"}}},
				{File: &storage.FileActivity_File{Path: "/file1"}, Operation: storage.FileActivity_WRITE, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc1"}}},
				{File: &storage.FileActivity_File{Path: "/file2"}, Operation: storage.FileActivity_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc2"}}},
				{File: &storage.FileActivity_File{Path: "/file3"}, Operation: storage.FileActivity_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc3"}}},
				{File: &storage.FileActivity_File{Path: "/file4"}, Operation: storage.FileActivity_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc4"}}},
				{File: &storage.FileActivity_File{Path: "/file5"}, Operation: storage.FileActivity_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc5"}}},
				{File: &storage.FileActivity_File{Path: "/file6"}, Operation: storage.FileActivity_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc6"}}},
				{File: &storage.FileActivity_File{Path: "/file7"}, Operation: storage.FileActivity_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc7"}}},
				{File: &storage.FileActivity_File{Path: "/file8"}, Operation: storage.FileActivity_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc8"}}},
				{File: &storage.FileActivity_File{Path: "/file9"}, Operation: storage.FileActivity_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc9"}}},
			},
			expected: "'/file1' accessed (OPEN) by proc1; '/file1' accessed (WRITE) by proc1; '/file2' accessed (OPEN) by proc2; '/file3' accessed (OPEN) by proc3; '/file4' accessed (OPEN) by proc4; '/file5' accessed (OPEN) by proc5; '/file6' accessed (OPEN) by proc6; '/file7' accessed (OPEN) by proc7; '/file8' accessed (OPEN) by proc8; '/file9' accessed (OPEN) by proc9",
		},
		{
			desc: "different file operations",
			activity: []*storage.FileActivity{
				{
					File: &storage.FileActivity_File{Path: "/tmp/new_file"},
					Operation: storage.FileActivity_CREATE,
					Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "touch"}},
				},
				{
					File: &storage.FileActivity_File{Path: "/tmp/old_file"},
					Operation: storage.FileActivity_UNLINK,
					Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "rm"}},
				},
				{
					File: &storage.FileActivity_File{Path: "/tmp/renamed_file"},
					Operation: storage.FileActivity_RENAME,
					Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "mv"}},
				},
				{
					File: &storage.FileActivity_File{Path: "/tmp/chmod_file"},
					Operation: storage.FileActivity_PERMISSION_CHANGE,
					Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "chmod"}},
				},
				{
					File: &storage.FileActivity_File{Path: "/tmp/chown_file"},
					Operation: storage.FileActivity_OWNERSHIP_CHANGE,
					Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "chown"}},
				},
			},
			expected: "'/tmp/new_file' accessed (CREATE) by touch; '/tmp/old_file' accessed (UNLINK) by rm; '/tmp/renamed_file' accessed (RENAME) by mv; '/tmp/chmod_file' accessed (PERMISSION_CHANGE) by chmod; '/tmp/chown_file' accessed (OWNERSHIP_CHANGE) by chown",
		},
		{
			desc: "nil file path handling",
			activity: []*storage.FileActivity{
				{
					File:      nil,
					Operation: storage.FileActivity_OPEN,
					Process: &storage.ProcessIndicator{
						Signal: &storage.ProcessSignal{Name: "test"},
					},
				},
			},
			expected: "'' accessed (OPEN) by test",
		},
		{
			desc: "nil process handling",
			activity: []*storage.FileActivity{
				{
					File: &storage.FileActivity_File{Path: "/test/file"},
					Operation: storage.FileActivity_OPEN,
					Process:   nil,
				},
			},
			expected: "'/test/file' accessed (OPEN) by ",
		},
		{
			desc: "nil process signal handling",
			activity: []*storage.FileActivity{
				{
					File: &storage.FileActivity_File{Path: "/test/file"},
					Operation: storage.FileActivity_OPEN,
					Process: &storage.ProcessIndicator{
						Signal: nil,
					},
				},
			},
			expected: "'/test/file' accessed (OPEN) by ",
		},
		{
			desc: "empty file path",
			activity: []*storage.FileActivity{
				{
					File: &storage.FileActivity_File{Path: ""},
					Operation: storage.FileActivity_OPEN,
					Process: &storage.ProcessIndicator{
						Signal: &storage.ProcessSignal{Name: "test"},
					},
				},
			},
			expected: "'' accessed (OPEN) by test",
		},
		{
			desc: "empty process name",
			activity: []*storage.FileActivity{
				{
					File: &storage.FileActivity_File{Path: "/test/file"},
					Operation: storage.FileActivity_OPEN,
					Process: &storage.ProcessIndicator{
						Signal: &storage.ProcessSignal{Name: ""},
					},
				},
			},
			expected: "'/test/file' accessed (OPEN) by ",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			violation := getAlertFileViolation("", tc.activity)
			UpdateFileActivityAlertViolationMessage(violation)
			protoassert.Equal(t, getAlertFileViolation(tc.expected, tc.activity), violation)
		})
	}
}
