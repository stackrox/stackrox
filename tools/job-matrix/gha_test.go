package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func Test_triggersFromOn(t *testing.T) {
	tests := map[string]struct {
		yaml string
		want []Trigger
	}{
		"single string trigger": {
			yaml: `on: workflow_call`,
			want: []Trigger{TriggerReusable},
		},
		"list of trigger names": {
			yaml: "on: [push, pull_request]",
			want: []Trigger{TriggerPostMerge, TriggerPreMerge},
		},
		"map form with nested config": {
			yaml: "on:\n  push:\n    branches: [master]\n  schedule:\n    - cron: '0 2 * * *'\n",
			want: []Trigger{TriggerPostMerge, TriggerPeriodic},
		},
		"map form with multiple triggers including manual": {
			yaml: "on:\n  pull_request_target: {}\n  workflow_dispatch: {}\n",
			want: []Trigger{TriggerPreMerge, TriggerManual},
		},
		"unrecognized event names yield unknown": {
			yaml: "on: issue_comment",
			want: []Trigger{TriggerUnknown},
		},
		"no on key yields unknown": {
			yaml: "name: foo",
			want: []Trigger{TriggerUnknown},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var wf workflowFile
			require.NoError(t, yaml.Unmarshal([]byte(tt.yaml), &wf))
			assert.ElementsMatch(t, tt.want, triggersFromOn(wf.On))
		})
	}
}

func Test_GHAWorkflows(t *testing.T) {
	dir := t.TempDir()

	mappedContent := "name: Mapped Workflow\non:\n  push: {}\n  pull_request: {}\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "e2e-qa-tests-gke.yaml"), []byte(mappedContent), 0o600))

	unmappedContent := "name: Brand New Workflow\non: workflow_dispatch\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "brand-new-workflow.yaml"), []byte(unmappedContent), 0o600))

	labelGatedContent := "name: Backport PR to release branch\non:\n  pull_request:\n    types: [closed, labeled]\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "backport-pr.yaml"), []byte(labelGatedContent), 0o600))

	// Non-workflow files in the directory must be ignored.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("not a workflow"), 0o600))

	entries, err := GHAWorkflows(dir)
	require.NoError(t, err)
	require.Len(t, entries, 3)

	byName := map[string]GHAEntry{}
	for _, e := range entries {
		byName[e.FileName] = e
	}

	mapped := byName["e2e-qa-tests-gke.yaml"]
	assert.Equal(t, "Mapped Workflow", mapped.WorkflowName)
	assert.ElementsMatch(t, []Trigger{TriggerPostMerge, TriggerPreMerge}, mapped.Triggers)
	assert.Equal(t, "GKE (direct provisioning via scripts/ci/gke.sh)", mapped.ClusterType)
	assert.Equal(t, "qa-e2e-tests", mapped.TestFamily)
	assert.Empty(t, mapped.RequiredLabel)

	unmapped := byName["brand-new-workflow.yaml"]
	assert.Equal(t, "Brand New Workflow", unmapped.WorkflowName)
	assert.Equal(t, []Trigger{TriggerManual}, unmapped.Triggers)
	assert.Equal(t, "Unknown", unmapped.ClusterType)
	assert.Equal(t, "Unknown", unmapped.Description)
	assert.Empty(t, unmapped.TestFamily)
	assert.Empty(t, unmapped.RequiredLabel)

	// backport-pr.yaml only does anything once a "backport release-*" label
	// is present, so it must carry a non-empty RequiredLabel even though its
	// on: trigger is a plain pull_request.
	labelGated := byName["backport-pr.yaml"]
	assert.ElementsMatch(t, []Trigger{TriggerPreMerge}, labelGated.Triggers)
	assert.NotEmpty(t, labelGated.RequiredLabel)
}
