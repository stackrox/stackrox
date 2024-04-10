package main

import (
	"reflect"
	"testing"
)

func Test_retestNTimes(t *testing.T) {
	tests := []struct {
		name     string
		comments []string
		want     []string
	}{
		{
			name:     "nil",
			comments: nil,
			want:     []string{},
		},
		{
			name:     "empty",
			comments: nil,
			want:     []string{},
		},
		{
			name:     "not matching regexp",
			comments: []string{"lorem ipsum"},
			want:     []string{},
		},
		{
			name:     "request test 10 times",
			comments: []string{"Retest job-name-1 10 times"},
			want: []string{
				"job-name-1",
			},
		},
		{
			name:     "too many times",
			comments: []string{"Retest job-name-1 101 times"},
			want:     []string{},
		},
		{
			name:     "invalid number",
			comments: []string{"Retest job-name-1 99999999999999999999999 times"},
			want:     []string{},
		},
		{
			name: "request test 10 times, with 5 already done",
			comments: []string{
				"Retest job-name-1 10 times",
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
			comments: []string{
				"Retest job-name-1 10 times",
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
			comments: []string{
				"Retest job-name-1 10 times",
				"/test job-name-2",
				"/test job-name-3",
				"/test job-name-3",
				"Retest job-name-1 10 times",
				"Retest job-name-2 10 times",
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
			comments: []string{
				"Retest job-name-1 1 times",
				"/test job-name-1",
				"/test job-name-2",
				"/test job-name-1",
				"Retest job-name-1 1 times",
				"Retest job-name-2 1 times",
			},
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := jobsToRetestFromComments(tt.comments); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("jobsToRetestFromComments() = %v, want %v", got, tt.want)
			}
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
			if got := shouldRetest(tt.statuses, tt.comments); got != tt.want {
				t.Errorf("shouldRetest() = %v, want %v", got, tt.want)
			}
		})
	}
}
