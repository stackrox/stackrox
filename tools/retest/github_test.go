package main

import (
	"testing"

	"github.com/google/go-github/v60/github"
	"github.com/stretchr/testify/assert"
)

// Test_jobStateMapping checks how parseJobState and checkToState each map
// their own API's raw value — a Statuses API state, or a Checks API
// conclusion — onto the shared jobState type.
func Test_jobStateMapping(t *testing.T) {
	tests := map[string]struct {
		raw       string
		fromCheck bool
		wantState jobState
	}{
		"status: success maps to ok": {
			raw:       "success",
			wantState: jobOK,
		},
		"status: pending maps to pending": {
			raw:       "pending",
			wantState: jobPending,
		},
		"status: failure maps to failure": {
			raw:       "failure",
			wantState: jobFailure,
		},
		"status: error maps to failure, since it also warrants a retest": {
			raw:       "error",
			wantState: jobFailure,
		},
		"status: unknown state falls back to ok": {
			raw:       "something-unexpected",
			wantState: jobOK,
		},
		"status: empty string falls back to ok": {
			raw:       "",
			wantState: jobOK,
		},
		"check: success maps to ok": {
			raw:       "success",
			fromCheck: true,
			wantState: jobOK,
		},
		"check: failure maps to failure": {
			raw:       "failure",
			fromCheck: true,
			wantState: jobFailure,
		},
		"check: timed_out maps to failure, since it also warrants a retest": {
			raw:       "timed_out",
			fromCheck: true,
			wantState: jobFailure,
		},
		"check: neutral falls back to ok": {
			raw:       "neutral",
			fromCheck: true,
			wantState: jobOK,
		},
		"check: cancelled falls back to ok": {
			raw:       "cancelled",
			fromCheck: true,
			wantState: jobOK,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if tt.fromCheck {
				check := &github.CheckRun{Conclusion: github.String(tt.raw)}
				assert.Equal(t, tt.wantState, checkToState(check))
				return
			}
			assert.Equal(t, tt.wantState, parseJobState(tt.raw))
		})
	}
}
