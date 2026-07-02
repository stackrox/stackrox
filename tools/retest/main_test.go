package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_retestNTimes(t *testing.T) {
	tests := []struct {
		name         string
		userComments []string
		allComments  []string
		want         []string
		error        string
	}{
		{
			name:        "nil",
			allComments: nil,
			want:        []string{},
		},
		{
			name:        "empty",
			allComments: nil,
			want:        []string{},
		},
		{
			name:        "not matching regexp",
			allComments: []string{"lorem ipsum"},
			want:        []string{},
		},
		{
			name:        "request test 10 times",
			allComments: []string{"/retest-times 10 job-name-1"},
			want: []string{
				"job-name-1",
			},
		},
		{
			name:        "extra whitespace between count and job name",
			allComments: []string{"/retest-times 10  job-name-1"},
			want: []string{
				"job-name-1",
			},
		},
		{
			name:        "extra whitespace before count",
			allComments: []string{"/retest-times    3 job-name-1"},
			want: []string{
				"job-name-1",
			},
		},
		{
			name:        "extra whitespace in all positions",
			allComments: []string{"/retest-times   10   job-name-1"},
			want: []string{
				"job-name-1",
			},
		},
		{
			name:        "tab between count and job name",
			allComments: []string{"/retest-times 5\tjob-name-1"},
			want: []string{
				"job-name-1",
			},
		},
		{
			name:        "trailing whitespace in job name",
			allComments: []string{"/retest-times 3 job-name-1   "},
			want: []string{
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
			want: []string{
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
			want: []string{
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
			want: []string{
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
			want: []string{
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
			want: []string{
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
			want: []string{},
		},
		{
			name:         "request test 1 and one retested by another user",
			userComments: []string{},
			allComments: []string{
				"/retest-times 1 job-name-1",
				"/test job-name-1",
			},
			want: []string{
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
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := jobsToRetestFromComments(tt.userComments, tt.allComments)
			if tt.error == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.error)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_shouldRetest(t *testing.T) {
	tests := []struct {
		name     string
		statuses map[string]string
		comments []string
		want     bool
	}{
		{
			name:     "nil",
			statuses: nil,
			comments: nil,
			want:     false,
		},
		{
			name:     "empty",
			statuses: map[string]string{},
			comments: []string{},
			want:     false,
		},
		{
			name: "all success",
			statuses: map[string]string{
				"a": "success",
				"b": "success",
				"c": "success",
			},
			comments: []string{},
			want:     false,
		},
		{
			name: "one failure",
			statuses: map[string]string{
				"a": "success",
				"b": "failure",
				"c": "success",
			},
			comments: []string{},
			want:     true,
		},
		{
			name: "one failure but already retested",
			statuses: map[string]string{
				"a": "success",
				"b": "failure",
				"c": "success",
			},
			comments: []string{"/retest"},
			want:     true,
		},
		{
			name: "one failure but already retested",
			statuses: map[string]string{
				"a": "success",
				"b": "failure",
				"c": "success",
			},
			comments: []string{"/retest", "/retest", "/retest", "/retest"},
			want:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldRetestFailedStatuses(tt.statuses, tt.comments)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_commentsToCreate(t *testing.T) {
	tests := []struct {
		name         string
		statuses     map[string]string
		jobsToRetest []string
		shouldRetest bool
		want         []string
	}{
		{
			name:         "nil",
			statuses:     nil,
			jobsToRetest: nil,
			want:         nil,
		},
		{
			name:         "empty",
			statuses:     map[string]string{},
			jobsToRetest: []string{},
			want:         nil,
		},
		{
			name:         "competed",
			statuses:     map[string]string{"job-1": "succeeded"},
			jobsToRetest: []string{"job-1"},
			want:         []string{"/test job-1"},
		},
		{
			name:         "competed",
			statuses:     map[string]string{"job-1": "pending"},
			jobsToRetest: []string{"job-1"},
			want:         nil,
		},
		{
			name:         "competed",
			statuses:     map[string]string{"job-1": "succeeded"},
			jobsToRetest: []string{"job-1"},
			want:         []string{"/test job-1"},
		},
		{
			name:         "retest",
			statuses:     map[string]string{"job-1": "failure"},
			jobsToRetest: []string{},
			shouldRetest: true,
			want:         []string{"/retest"},
		},
		{
			name:         "retest",
			statuses:     map[string]string{"job-1": "failure"},
			jobsToRetest: []string{},
			want:         nil,
		},
		{
			name:         "just test no retest",
			statuses:     map[string]string{"job-1": "failure"},
			jobsToRetest: []string{"job-1"},
			shouldRetest: true,
			want:         []string{"/test job-1"},
		},
		{
			name:         "pending job is skipped even with trimmed name",
			statuses:     map[string]string{"job-1": "pending"},
			jobsToRetest: []string{"job-1"},
			want:         nil,
		},
		{
			name:         "multiple jobs — pending skipped, completed retested",
			statuses:     map[string]string{"job-1": "pending", "job-2": "failure"},
			jobsToRetest: []string{"job-1", "job-2"},
			want:         []string{"/test job-2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := commentsToCreate(tt.statuses, tt.jobsToRetest, tt.shouldRetest)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_allowedCheckFailures(t *testing.T) {
	expectedAllowed := []string{
		"codecov/patch",
		"e2e-byodb-test",
		"e2e-nongroovy-tests",
		"e2e-db-backup-restore-test",
	}
	for _, name := range expectedAllowed {
		t.Run("allowed_"+name, func(t *testing.T) {
			_, ok := allowedCheckFailures[name]
			assert.True(t, ok, "expected %q to be in allowedCheckFailures", name)
		})
	}

	notAllowed := []string{
		"e2e-upgrade-tests",
		"e2e-groovy-tests",
		"unit-tests",
		"",
		"codecov/project",
	}
	for _, name := range notAllowed {
		t.Run("not_allowed_"+name, func(t *testing.T) {
			_, ok := allowedCheckFailures[name]
			assert.False(t, ok, "expected %q to NOT be in allowedCheckFailures", name)
		})
	}
}

func Test_allowedCheckFailuresExactCount(t *testing.T) {
	assert.Len(t, allowedCheckFailures, 4,
		"allowedCheckFailures should contain exactly 4 entries: codecov/patch, e2e-byodb-test, e2e-nongroovy-tests, e2e-db-backup-restore-test")
}

func Test_splitMultilineComment(t *testing.T) {
	tests := []struct {
		comment string

	}{
		{
			comment: "",
			want:    []string{},
		},
		{
			comment: "a\nb\nc",
			want:    []string{"a", "b", "c"},
		},
		{
			comment: "a \nb \t \n c \t \n \t",
			want:    []string{"a", "b", "c"},
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
			want: []string{
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
			assert.Equal(t, tt.want, splitMultilineComment(tt.comment))
		})
	}
}
