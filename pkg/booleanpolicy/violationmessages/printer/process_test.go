package printer

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func getProcessIndicator(name string, path string, args string, uid uint32) *storage.ProcessIndicator {
	return &storage.ProcessIndicator{
		Signal: &storage.ProcessSignal{
			Name:         name,
			ExecFilePath: path,
			Args:         args,
			Uid:          uid,
		},
	}
}

func getAlertProcessViolation(message string, processes ...*storage.ProcessIndicator) *storage.Alert_ProcessViolation {
	return &storage.Alert_ProcessViolation{
		Message:   message,
		Processes: processes,
	}
}

func TestUpdateRuntimeAlertViolationMessage(t *testing.T) {
	proc := []*storage.ProcessIndicator{
		getProcessIndicator("a", "/bin/a", "--arg", 0),
		getProcessIndicator("b", "/bin/b", "--arg", 0),
		getProcessIndicator("c", "/bin/c", "--arg", 0),
		getProcessIndicator("a", "/bin/a", "--arg-alt", 0),
		getProcessIndicator("b", "/bin/b", "--arg-alt", 0),
		getProcessIndicator("a", "/bin/a", "--arg", 1),
		getProcessIndicator("a0", "/bin/a0", "--arg", 0),
		getProcessIndicator("b0", "/bin/b0", "--arg", 0),
		getProcessIndicator("c0", "/bin/c0", "--arg", 0),
		getProcessIndicator("a1", "/bin/a1", "--arg", 0),
		getProcessIndicator("b1", "/bin/b1", "--arg", 0),
		getProcessIndicator("c1", "/bin/c1", "--arg", 0),
		getProcessIndicator("c2", "/bin/c2", "--arg", 0),
	}

	cases := []struct {
		desc            string
		processes       []*storage.ProcessIndicator
		expectedMessage string
	}{
		{
			desc:            "empty",
			processes:       nil,
			expectedMessage: "",
		},
		{
			desc:            "1 binary",
			processes:       []*storage.ProcessIndicator{proc[0]},
			expectedMessage: "Binary '/bin/a' executed with arguments '--arg' under user ID 0",
		},
		{
			desc:            "2 binaries",
			processes:       []*storage.ProcessIndicator{proc[0], proc[1]},
			expectedMessage: "Binaries '/bin/a' and '/bin/b' executed with arguments '--arg' under user ID 0",
		},
		{
			desc:            "3 binaries",
			processes:       []*storage.ProcessIndicator{proc[0], proc[1], proc[2]},
			expectedMessage: "Binaries '/bin/a', '/bin/b', and '/bin/c' executed with arguments '--arg' under user ID 0",
		},
		{
			desc:            "3 binaries different args",
			processes:       []*storage.ProcessIndicator{proc[0], proc[1], proc[2], proc[3], proc[4]},
			expectedMessage: "Binaries '/bin/a', '/bin/b', and '/bin/c' executed with 2 different arguments under user ID 0",
		},
		{
			desc:            "3 binaries different args and uids",
			processes:       []*storage.ProcessIndicator{proc[0], proc[1], proc[2], proc[3], proc[4], proc[5]},
			expectedMessage: "Binaries '/bin/a', '/bin/b', and '/bin/c' executed with 2 different arguments under 2 different user IDs",
		},
		{
			desc:            "10 binaries different args and uids",
			processes:       proc,
			expectedMessage: "10 binaries executed with 2 different arguments under 2 different user IDs",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			processViolation := getAlertProcessViolation("", tc.processes...)
			UpdateProcessAlertViolationMessage(processViolation)
			assert.Equal(t, getAlertProcessViolation(tc.expectedMessage, tc.processes...), processViolation)
		})
	}
}
