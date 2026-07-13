package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test_jobsToRetestFromComments checks which jobs jobsToRetestFromComments
// decides still need a fresh "/test <job>" comment, given how many retests a
// "/retest-times N <job>" comment requested and how many "/test <job>"
// comments the bot has already posted for that job (userComments).
func Test_jobsToRetestFromComments(t *testing.T) {
	tests := []struct {
		name             string
		userComments     []string
		allComments      []string
		wantJobsToRetest []string
		wantSkipped      []skipReason
		error            string
	}{
		{
			name:             "nil",
			allComments:      nil,
			wantJobsToRetest: []string{},
		},
		{
			name:             "empty",
			allComments:      nil,
			wantJobsToRetest: []string{},
		},
		{
			name:             "not matching regexp",
			allComments:      []string{"lorem ipsum"},
			wantJobsToRetest: []string{},
		},
		{
			name:        "request test 10 times",
			allComments: []string{"/retest-times 10 job-name-1"},
			wantJobsToRetest: []string{
				"job-name-1",
			},
		},
		{
			name:        "extra whitespace between count and job name",
			allComments: []string{"/retest-times 10  job-name-1"},
			wantJobsToRetest: []string{
				"job-name-1",
			},
		},
		{
			name:        "extra whitespace before count",
			allComments: []string{"/retest-times    3 job-name-1"},
			wantJobsToRetest: []string{
				"job-name-1",
			},
		},
		{
			name:        "extra whitespace in all positions",
			allComments: []string{"/retest-times   10   job-name-1"},
			wantJobsToRetest: []string{
				"job-name-1",
			},
		},
		{
			name:        "tab between count and job name",
			allComments: []string{"/retest-times 5\tjob-name-1"},
			wantJobsToRetest: []string{
				"job-name-1",
			},
		},
		{
			name:        "trailing whitespace in job name",
			allComments: []string{"/retest-times 3 job-name-1   "},
			wantJobsToRetest: []string{
				"job-name-1",
			},
		},
		{
			name: "extra whitespace — bot /test comments also have extra space",
			userComments: []string{
				"/test  job-name-1",
				"/test  job-name-1",
			},
			allComments: []string{
				"/retest-times 10  job-name-1",
				"/test  job-name-1",
				"/test  job-name-1",
			},
			wantJobsToRetest: []string{
				"job-name-1",
			},
		},
		{
			name: "mixed whitespace — single and double space refer to same job",
			userComments: []string{
				"/test job-name-1",
				"/test  job-name-1",
			},
			allComments: []string{
				"/retest-times 3 job-name-1",
				"/test job-name-1",
				"/test  job-name-1",
			},
			wantJobsToRetest: []string{
				"job-name-1",
			},
		},
		{
			name:        "too many",
			allComments: []string{"/retest-times 101 job-name-1"},
			error:       `invalid retest number requested: "/retest-times 101 job-name-1"`,
		},
		{
			name:        "invalid number",
			allComments: []string{"/retest-times 99999999999999999999999 job-name-1"},
			error:       `got an error in a comment "/retest-times 99999999999999999999999 job-name-1": strconv.Atoi: parsing "99999999999999999999999": value out of range`,
		},
		{
			name: "request test 10 times, with 5 already done",
			userComments: []string{
				"/test job-name-1",
				"/test job-name-1",
				"/test job-name-1",
				"/test job-name-1",
				"/test job-name-1",
			},
			allComments: []string{
				"/retest-times 10 job-name-1",
				"/test job-name-1",
				"/test job-name-1",
				"/test job-name-1",
				"/test job-name-1",
				"/test job-name-1",
			},
			wantJobsToRetest: []string{
				"job-name-1",
			},
		},
		{
			name: "request test 10 times, with 3 already done and other as well",
			userComments: []string{
				"/test job-name-1",
				"/test job-name-2",
				"/test job-name-3",
				"/test job-name-1",
				"/test job-name-1",
			},
			allComments: []string{
				"/retest-times 10 job-name-1",
				"/test job-name-1",
				"/test job-name-2",
				"/test job-name-3",
				"/test job-name-1",
				"/test job-name-1",
			},
			wantJobsToRetest: []string{
				"job-name-1",
			},
		},
		{
			name: "request test 10 times for multiple jobs",
			userComments: []string{
				"/test job-name-2",
				"/test job-name-3",
				"/test job-name-3",
				"/test job-name-3",
				"/test job-name-3",
			},
			allComments: []string{
				"/retest-times 10 job-name-1",
				"/test job-name-2",
				"/test job-name-3",
				"/test job-name-3",
				"/retest-times 10 job-name-1",
				"/retest-times 10 job-name-2",
				"/test job-name-3",
				"/test job-name-3",
			},
			wantJobsToRetest: []string{
				"job-name-1",
				"job-name-2",
			},
		},
		{
			name: "request test 10 times for multiple jobs",
			userComments: []string{
				"/test job-name-1",
				"/test job-name-2",
				"/test job-name-1",
			},
			allComments: []string{
				"/retest-times 1 job-name-1",
				"/test job-name-1",
				"/test job-name-2",
				"/test job-name-1",
				"/retest-times 1 job-name-1",
				"/retest-times 1 job-name-2",
			},
			wantJobsToRetest: []string{},
			wantSkipped: []skipReason{
				{job: "job-name-1", message: "exceeded retest budget of 2, already tested 2 times"},
				{job: "job-name-2", message: "exceeded retest budget of 1, already tested 1 times"},
			},
		},
		{
			name:         "request test 1 and one retested by another user",
			userComments: []string{},
			allComments: []string{
				"/retest-times 1 job-name-1",
				"/test job-name-1",
			},
			wantJobsToRetest: []string{
				"job-name-1",
			},
		},
		{
			name: "request test 1 and one retested by current user",
			userComments: []string{
				"/test job-name-1",
			},
			allComments: []string{
				"/retest-times 1 job-name-1",
				"/test job-name-1",
			},
			wantJobsToRetest: []string{},
			wantSkipped: []skipReason{
				{job: "job-name-1", message: "exceeded retest budget of 1, already tested 1 times"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotJobsToRetest, gotSkipped, err := jobsToRetestFromComments(tt.userComments, tt.allComments)
			if tt.error == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.error)
			}
			assert.Equal(t, tt.wantJobsToRetest, gotJobsToRetest)
			assert.Equal(t, tt.wantSkipped, gotSkipped)
		})
	}
}

// Test_skipRetestReason checks whether skipRetestReason decides a PR's
// "/retest" should be issued (nil) or withheld (a reason), given its
// statuses, its retestable check results, and how many times it has already
// been retested.
func Test_skipRetestReason(t *testing.T) {
	tests := map[string]struct {
		statuses          map[string]string
		comments          []string
		checks            map[string]bool
		wantSkipReasonMsg string
	}{
		"nil": {
			statuses:          nil,
			comments:          nil,
			wantSkipReasonMsg: "no failing status or retestable check found",
		},
		"empty": {
			statuses:          map[string]string{},
			comments:          []string{},
			wantSkipReasonMsg: "no failing status or retestable check found",
		},
		"all success": {
			statuses: map[string]string{
				"a": "success",
				"b": "success",
				"c": "success",
			},
			comments:          []string{},
			wantSkipReasonMsg: "no failing status or retestable check found",
		},
		"one failure": {
			statuses: map[string]string{
				"a": "success",
				"b": "failure",
				"c": "success",
			},
			comments: []string{},
		},
		"one failure but already retested once": {
			statuses: map[string]string{
				"a": "success",
				"b": "failure",
				"c": "success",
			},
			comments: []string{"/retest"},
		},
		"one failure but already retested too many times": {
			statuses: map[string]string{
				"a": "success",
				"b": "failure",
				"c": "success",
			},
			comments:          []string{"/retest", "/retest", "/retest", "/retest"},
			wantSkipReasonMsg: "PR has already been retested 4 times",
		},
		"failed e2e check triggers retest": {
			statuses: map[string]string{},
			comments: []string{},
			checks:   map[string]bool{"e2e-gke-tests": false},
		},
		"failed e2e check but already retested too many times": {
			statuses:          map[string]string{},
			comments:          []string{"/retest", "/retest", "/retest", "/retest"},
			checks:            map[string]bool{"e2e-gke-tests": false},
			wantSkipReasonMsg: "PR has already been retested 4 times",
		},
		"failed e2e check with prow success": {
			statuses: map[string]string{
				"a": "success",
			},
			comments: []string{},
			checks:   map[string]bool{"e2e-gke-tests": false},
		},
		"passing e2e check does not trigger retest": {
			statuses:          map[string]string{},
			comments:          []string{},
			checks:            map[string]bool{"e2e-gke-tests": true},
			wantSkipReasonMsg: "no failing status or retestable check found",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			gotReason := skipRetestReason(tt.statuses, tt.comments, tt.checks)
			if tt.wantSkipReasonMsg == "" {
				assert.NoError(t, gotReason)
			} else {
				assert.EqualError(t, gotReason, tt.wantSkipReasonMsg)
			}
		})
	}
}

// Test_commentsToCreate checks which "/test <job>" and "/retest" comments
// commentsToCreate decides to post, given the jobs that need retesting and
// whether the PR as a whole warrants a "/retest".
func Test_commentsToCreate(t *testing.T) {
	tests := []struct {
		name         string
		statuses     map[string]string
		jobsToRetest []string
		shouldRetest bool
		wantComments []string
		wantSkipped  []skipReason
	}{
		{
			name:         "nil",
			statuses:     nil,
			jobsToRetest: nil,
			wantComments: nil,
		},
		{
			name:         "empty",
			statuses:     map[string]string{},
			jobsToRetest: []string{},
			wantComments: nil,
		},
		{
			name:         "competed",
			statuses:     map[string]string{"job-1": "succeeded"},
			jobsToRetest: []string{"job-1"},
			wantComments: []string{"/test job-1"},
		},
		{
			name:         "competed",
			statuses:     map[string]string{"job-1": "pending"},
			jobsToRetest: []string{"job-1"},
			wantComments: nil,
			wantSkipped:  []skipReason{{job: "job-1", message: "already pending"}},
		},
		{
			name:         "competed",
			statuses:     map[string]string{"job-1": "succeeded"},
			jobsToRetest: []string{"job-1"},
			wantComments: []string{"/test job-1"},
		},
		{
			name:         "retest",
			statuses:     map[string]string{"job-1": "failure"},
			jobsToRetest: []string{},
			shouldRetest: true,
			wantComments: []string{"/retest"},
		},
		{
			name:         "retest",
			statuses:     map[string]string{"job-1": "failure"},
			jobsToRetest: []string{},
			wantComments: nil,
		},
		{
			name:         "just test no retest",
			statuses:     map[string]string{"job-1": "failure"},
			jobsToRetest: []string{"job-1"},
			shouldRetest: true,
			wantComments: []string{"/test job-1"},
		},
		{
			name:         "pending job is skipped even with trimmed name",
			statuses:     map[string]string{"job-1": "pending"},
			jobsToRetest: []string{"job-1"},
			wantComments: nil,
			wantSkipped:  []skipReason{{job: "job-1", message: "already pending"}},
		},
		{
			name:         "multiple jobs — pending skipped, completed retested",
			statuses:     map[string]string{"job-1": "pending", "job-2": "failure"},
			jobsToRetest: []string{"job-1", "job-2"},
			wantComments: []string{"/test job-2"},
			wantSkipped:  []skipReason{{job: "job-1", message: "already pending"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotComments, gotSkipped := commentsToCreate(tt.statuses, tt.jobsToRetest, tt.shouldRetest)
			assert.Equal(t, tt.wantComments, gotComments)
			assert.Equal(t, tt.wantSkipped, gotSkipped)
		})
	}
}

// Test_splitMultilineComment checks that splitMultilineComment breaks a
// (possibly multi-line, possibly indented) GitHub comment body into a list
// of trimmed, non-empty lines.
func Test_splitMultilineComment(t *testing.T) {
	tests := []struct {
		comment   string
		wantLines []string
	}{
		{
			comment:   "",
			wantLines: []string{},
		},
		{
			comment:   "a\nb\nc",
			wantLines: []string{"a", "b", "c"},
		},
		{
			comment:   "a \nb \t \n c \t \n \t",
			wantLines: []string{"a", "b", "c"},
		},
		{
			comment: `
				/retest-times 1 job-name-1
				/test job-name-1
				/test job-name-2
				/test job-name-1
				/retest-times 1 job-name-1
				/retest-times 1 job-name-2
			`,
			wantLines: []string{
				"/retest-times 1 job-name-1",
				"/test job-name-1",
				"/test job-name-2",
				"/test job-name-1",
				"/retest-times 1 job-name-1",
				"/retest-times 1 job-name-2",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.comment, func(t *testing.T) {
			gotLines := splitMultilineComment(tt.comment)
			assert.Equal(t, tt.wantLines, gotLines)
		})
	}
}
