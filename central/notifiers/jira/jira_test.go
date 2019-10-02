package jira

import (
	"strconv"
	"testing"

	jiraLib "github.com/andygrunwald/go-jira"
	"github.com/stackrox/rox/generated/storage"
)

func TestMapPriorities(t *testing.T) {
	testCases := []struct {
		input  string
		output storage.Severity
	}{
		// P0/Blocker P1/Major P2/Normal P3/Minor P4 Lowest
		{"P3-Mumble", storage.Severity_LOW_SEVERITY},
		{"P0/Blocker", storage.Severity_CRITICAL_SEVERITY},
		{"P4", storage.Severity_UNSET_SEVERITY},
	}
	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			m := mapPriorities([]jiraLib.Priority{{Name: tc.input}})
			v, ok := m[tc.output]
			if !ok {
				t.Errorf("Key %q not found in map", string(tc.output))
				return
			}
			if v != tc.input {
				t.Errorf("Expected %q but got %q", tc.input, v)
			}
		})
	}
}
