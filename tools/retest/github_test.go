package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test_parseJobState checks how parseJobState maps the Statuses API's raw
// state strings (success/pending/failure/error, plus any other value it
// might report) onto jobState.
func Test_parseJobState(t *testing.T) {
	tests := map[string]struct {
		raw       string
		wantState jobState
	}{
		"success maps to ok": {
			raw:       "success",
			wantState: jobOK,
		},
		"pending maps to pending": {
			raw:       "pending",
			wantState: jobPending,
		},
		"failure maps to failure": {
			raw:       "failure",
			wantState: jobFailure,
		},
		"error maps to failure, since it also warrants a retest": {
			raw:       "error",
			wantState: jobFailure,
		},
		"unknown state falls back to ok": {
			raw:       "something-unexpected",
			wantState: jobOK,
		},
		"empty string falls back to ok": {
			raw:       "",
			wantState: jobOK,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.wantState, parseJobState(tt.raw))
		})
	}
}
