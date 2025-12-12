package printer

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
)

func getAlertFileViolation(msg string, accesses []*storage.FileAccess) *storage.Alert_FileAccessViolation {
	return &storage.Alert_FileAccessViolation{
		Message:  msg,
		Accesses: accesses,
	}
}

func TestUpdateFileAccessMessage(t *testing.T) {
	testCases := []struct {
		desc     string
		activity []*storage.FileAccess
		expected string
	}{
		{
			desc:     "empty activity list",
			activity: nil,
			expected: "",
		},
		{
			desc:     "empty activity slice",
			activity: []*storage.FileAccess{},
			expected: "",
		},
		{
			desc: "single file activity",
			activity: []*storage.FileAccess{
				{
					File: &storage.FileAccess_File{
						ActualFilePath: "/etc/passwd",
					},
					Operation: storage.FileAccess_OPEN,
					Process: &storage.ProcessIndicator{
						Signal: &storage.ProcessSignal{
							Name: "cat",
						},
					},
				},
			},
			expected: "'/etc/passwd' accessed (OPEN)",
		},
		{
			desc: "multiple activities on same file",
			activity: []*storage.FileAccess{
				{
					File: &storage.FileAccess_File{
						ActualFilePath: "/etc/passwd",
					},
					Operation: storage.FileAccess_OPEN,
					Process: &storage.ProcessIndicator{
						Signal: &storage.ProcessSignal{
							Name: "cat",
						},
					},
				},
				{
					File: &storage.FileAccess_File{
						ActualFilePath: "/etc/passwd",
					},
					Operation: storage.FileAccess_WRITE,
					Process: &storage.ProcessIndicator{
						Signal: &storage.ProcessSignal{
							Name: "vim",
						},
					},
				},
			},
			expected: "'/etc/passwd' accessed (OPEN, WRITE)",
		},
		{
			desc: "multiple activities on different files",
			activity: []*storage.FileAccess{
				{
					File: &storage.FileAccess_File{
						ActualFilePath: "/etc/passwd",
					},
					Operation: storage.FileAccess_OPEN,
					Process: &storage.ProcessIndicator{
						Signal: &storage.ProcessSignal{
							Name: "cat",
						},
					},
				},
				{
					File: &storage.FileAccess_File{
						ActualFilePath: "/etc/shadow",
					},
					Operation: storage.FileAccess_OPEN,
					Process: &storage.ProcessIndicator{
						Signal: &storage.ProcessSignal{
							Name: "grep",
						},
					},
				},
			},
			expected: "'/etc/passwd' accessed (OPEN); '/etc/shadow' accessed (OPEN)",
		},
		{
			desc: "exactly 10 unique files - should use summary format",
			activity: []*storage.FileAccess{
				{File: &storage.FileAccess_File{ActualFilePath: "/file1"}, Operation: storage.FileAccess_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc1"}}},
				{File: &storage.FileAccess_File{ActualFilePath: "/file2"}, Operation: storage.FileAccess_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc2"}}},
				{File: &storage.FileAccess_File{ActualFilePath: "/file3"}, Operation: storage.FileAccess_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc3"}}},
				{File: &storage.FileAccess_File{ActualFilePath: "/file4"}, Operation: storage.FileAccess_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc4"}}},
				{File: &storage.FileAccess_File{ActualFilePath: "/file5"}, Operation: storage.FileAccess_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc5"}}},
				{File: &storage.FileAccess_File{ActualFilePath: "/file6"}, Operation: storage.FileAccess_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc6"}}},
				{File: &storage.FileAccess_File{ActualFilePath: "/file7"}, Operation: storage.FileAccess_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc7"}}},
				{File: &storage.FileAccess_File{ActualFilePath: "/file8"}, Operation: storage.FileAccess_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc8"}}},
				{File: &storage.FileAccess_File{ActualFilePath: "/file9"}, Operation: storage.FileAccess_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc9"}}},
				{File: &storage.FileAccess_File{ActualFilePath: "/file10"}, Operation: storage.FileAccess_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc10"}}},
			},
			expected: "10 sensitive files accessed",
		},
		{
			desc: "more than 10 unique files - should use summary format",
			activity: []*storage.FileAccess{
				{File: &storage.FileAccess_File{ActualFilePath: "/file1"}, Operation: storage.FileAccess_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc1"}}},
				{File: &storage.FileAccess_File{ActualFilePath: "/file2"}, Operation: storage.FileAccess_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc2"}}},
				{File: &storage.FileAccess_File{ActualFilePath: "/file3"}, Operation: storage.FileAccess_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc3"}}},
				{File: &storage.FileAccess_File{ActualFilePath: "/file4"}, Operation: storage.FileAccess_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc4"}}},
				{File: &storage.FileAccess_File{ActualFilePath: "/file5"}, Operation: storage.FileAccess_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc5"}}},
				{File: &storage.FileAccess_File{ActualFilePath: "/file6"}, Operation: storage.FileAccess_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc6"}}},
				{File: &storage.FileAccess_File{ActualFilePath: "/file7"}, Operation: storage.FileAccess_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc7"}}},
				{File: &storage.FileAccess_File{ActualFilePath: "/file8"}, Operation: storage.FileAccess_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc8"}}},
				{File: &storage.FileAccess_File{ActualFilePath: "/file9"}, Operation: storage.FileAccess_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc9"}}},
				{File: &storage.FileAccess_File{ActualFilePath: "/file10"}, Operation: storage.FileAccess_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc10"}}},
				{File: &storage.FileAccess_File{ActualFilePath: "/file11"}, Operation: storage.FileAccess_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc11"}}},
				{File: &storage.FileAccess_File{ActualFilePath: "/file12"}, Operation: storage.FileAccess_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc12"}}},
			},
			expected: "12 sensitive files accessed",
		},
		{
			desc: "9 unique files with multiple activities each - should use detailed format",
			activity: []*storage.FileAccess{
				{File: &storage.FileAccess_File{ActualFilePath: "/file1"}, Operation: storage.FileAccess_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc1"}}},
				{File: &storage.FileAccess_File{ActualFilePath: "/file1"}, Operation: storage.FileAccess_WRITE, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc1"}}},
				{File: &storage.FileAccess_File{ActualFilePath: "/file2"}, Operation: storage.FileAccess_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc2"}}},
				{File: &storage.FileAccess_File{ActualFilePath: "/file3"}, Operation: storage.FileAccess_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc3"}}},
				{File: &storage.FileAccess_File{ActualFilePath: "/file4"}, Operation: storage.FileAccess_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc4"}}},
				{File: &storage.FileAccess_File{ActualFilePath: "/file5"}, Operation: storage.FileAccess_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc5"}}},
				{File: &storage.FileAccess_File{ActualFilePath: "/file6"}, Operation: storage.FileAccess_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc6"}}},
				{File: &storage.FileAccess_File{ActualFilePath: "/file7"}, Operation: storage.FileAccess_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc7"}}},
				{File: &storage.FileAccess_File{ActualFilePath: "/file8"}, Operation: storage.FileAccess_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc8"}}},
				{File: &storage.FileAccess_File{ActualFilePath: "/file9"}, Operation: storage.FileAccess_OPEN, Process: &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "proc9"}}},
			},
			expected: "'/file1' accessed (OPEN, WRITE); '/file2' accessed (OPEN); '/file3' accessed (OPEN); '/file4' accessed (OPEN); '/file5' accessed (OPEN); '/file6' accessed (OPEN); '/file7' accessed (OPEN); '/file8' accessed (OPEN); '/file9' accessed (OPEN)",
		},
		{
			desc: "different file operations",
			activity: []*storage.FileAccess{
				{
					File:      &storage.FileAccess_File{ActualFilePath: "/tmp/new_file"},
					Operation: storage.FileAccess_CREATE,
					Process:   &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "touch"}},
				},
				{
					File:      &storage.FileAccess_File{ActualFilePath: "/tmp/old_file"},
					Operation: storage.FileAccess_UNLINK,
					Process:   &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "rm"}},
				},
				{
					File:      &storage.FileAccess_File{ActualFilePath: "/tmp/renamed_file"},
					Operation: storage.FileAccess_RENAME,
					Process:   &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "mv"}},
				},
				{
					File:      &storage.FileAccess_File{ActualFilePath: "/tmp/chmod_file"},
					Operation: storage.FileAccess_PERMISSION_CHANGE,
					Process:   &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "chmod"}},
				},
				{
					File:      &storage.FileAccess_File{ActualFilePath: "/tmp/chown_file"},
					Operation: storage.FileAccess_OWNERSHIP_CHANGE,
					Process:   &storage.ProcessIndicator{Signal: &storage.ProcessSignal{Name: "chown"}},
				},
			},
			expected: "'/tmp/chmod_file' accessed (PERMISSION_CHANGE); '/tmp/chown_file' accessed (OWNERSHIP_CHANGE); '/tmp/new_file' accessed (CREATE); '/tmp/old_file' accessed (UNLINK); '/tmp/renamed_file' accessed (RENAME)",
		},
		{
			desc: "nil file path handling",
			activity: []*storage.FileAccess{
				{
					File:      nil,
					Operation: storage.FileAccess_OPEN,
					Process: &storage.ProcessIndicator{
						Signal: &storage.ProcessSignal{Name: "test"},
					},
				},
			},
			expected: "'' accessed (OPEN)",
		},
		{
			desc: "nil process handling",
			activity: []*storage.FileAccess{
				{
					File:      &storage.FileAccess_File{ActualFilePath: "/test/file"},
					Operation: storage.FileAccess_OPEN,
					Process:   nil,
				},
			},
			expected: "'/test/file' accessed (OPEN)",
		},
		{
			desc: "nil process signal handling",
			activity: []*storage.FileAccess{
				{
					File:      &storage.FileAccess_File{ActualFilePath: "/test/file"},
					Operation: storage.FileAccess_OPEN,
					Process: &storage.ProcessIndicator{
						Signal: nil,
					},
				},
			},
			expected: "'/test/file' accessed (OPEN)",
		},
		{
			desc: "empty file path",
			activity: []*storage.FileAccess{
				{
					File:      &storage.FileAccess_File{ActualFilePath: ""},
					Operation: storage.FileAccess_OPEN,
					Process: &storage.ProcessIndicator{
						Signal: &storage.ProcessSignal{Name: "test"},
					},
				},
			},
			expected: "'' accessed (OPEN)",
		},
		{
			desc: "empty process name",
			activity: []*storage.FileAccess{
				{
					File:      &storage.FileAccess_File{ActualFilePath: "/test/file"},
					Operation: storage.FileAccess_OPEN,
					Process: &storage.ProcessIndicator{
						Signal: &storage.ProcessSignal{Name: ""},
					},
				},
			},
			expected: "'/test/file' accessed (OPEN)",
		},
		{
			desc: "same file, many opens",
			activity: []*storage.FileAccess{
				{
					File:      &storage.FileAccess_File{ActualFilePath: "/test/file"},
					Operation: storage.FileAccess_OPEN,
				},
				{
					File:      &storage.FileAccess_File{ActualFilePath: "/test/file"},
					Operation: storage.FileAccess_OPEN,
				},
				{
					File:      &storage.FileAccess_File{ActualFilePath: "/test/file"},
					Operation: storage.FileAccess_OPEN,
				},
			},
			expected: "'/test/file' accessed (OPEN)",
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
