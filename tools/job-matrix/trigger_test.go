package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Trigger_String(t *testing.T) {
	tests := map[string]struct {
		trigger Trigger
		want    string
	}{
		"unknown":    {trigger: TriggerUnknown, want: "unknown"},
		"pre-merge":  {trigger: TriggerPreMerge, want: "pre-merge"},
		"post-merge": {trigger: TriggerPostMerge, want: "post-merge"},
		"periodic":   {trigger: TriggerPeriodic, want: "periodic"},
		"nightly":    {trigger: TriggerNightly, want: "nightly"},
		"manual":     {trigger: TriggerManual, want: "manual"},
		"reusable":   {trigger: TriggerReusable, want: "reusable"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.trigger.String())
		})
	}
}

func Test_Trigger_Description_isNonEmptyAndUnique(t *testing.T) {
	triggers := []Trigger{
		TriggerUnknown,
		TriggerPreMerge,
		TriggerPostMerge,
		TriggerPeriodic,
		TriggerNightly,
		TriggerManual,
		TriggerReusable,
	}

	seen := make(map[string]Trigger, len(triggers))
	for _, trigger := range triggers {
		description := trigger.Description()
		assert.NotEmptyf(t, description, "trigger %v has no description", trigger)

		if existing, ok := seen[description]; ok {
			t.Fatalf("triggers %v and %v share the same description %q", existing, trigger, description)
		}
		seen[description] = trigger
	}
}

func Test_Trigger_MarshalJSON(t *testing.T) {
	data, err := TriggerPreMerge.MarshalJSON()
	assert.NoError(t, err)
	assert.Equal(t, `"pre-merge"`, string(data))
}
