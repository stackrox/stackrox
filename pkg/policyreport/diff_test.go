package policyreport

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func newTestEvent(id string, message string) SecurityEvent {
	return SecurityEvent{
		ID:     id,
		Source: SourceKyverno,
		Type:   EventTypePolicyResult,
		Report: ReportRef{UID: "report-1"},
		Subject: Subject{
			Kind: "Pod",
			UID:  "pod-1",
		},
		Details: PolicyResult{
			ReportedPolicy: "policy-a",
			ReportedRule:   "rule-a",
			Result:         PolicyResultFail,
			Message:        message,
		},
	}
}

func TestDiff(t *testing.T) {
	for name, tc := range map[string]struct {
		old             []SecurityEvent
		new             []SecurityEvent
		expectedCreated []string
		expectedUpdated []string
		expectedRemoved []string
	}{
		"new report, no prior events: everything created": {
			old:             nil,
			new:             []SecurityEvent{newTestEvent("a", "m1"), newTestEvent("b", "m2")},
			expectedCreated: []string{"a", "b"},
		},
		"unchanged content: no created/updated/removed": {
			old: []SecurityEvent{newTestEvent("a", "m1")},
			new: []SecurityEvent{newTestEvent("a", "m1")},
			// all nil
		},
		"changed content, same ID: updated": {
			old:             []SecurityEvent{newTestEvent("a", "m1")},
			new:             []SecurityEvent{newTestEvent("a", "m2")},
			expectedUpdated: []string{"a"},
		},
		"result no longer present: removed": {
			old:             []SecurityEvent{newTestEvent("a", "m1"), newTestEvent("b", "m2")},
			new:             []SecurityEvent{newTestEvent("a", "m1")},
			expectedRemoved: []string{"b"},
		},
		"report deleted entirely: everything removed": {
			old:             []SecurityEvent{newTestEvent("a", "m1"), newTestEvent("b", "m2")},
			new:             nil,
			expectedRemoved: []string{"a", "b"},
		},
		"mixed create, update, remove in one diff": {
			old:             []SecurityEvent{newTestEvent("a", "m1"), newTestEvent("b", "m2")},
			new:             []SecurityEvent{newTestEvent("a", "m1-changed"), newTestEvent("c", "m3")},
			expectedCreated: []string{"c"},
			expectedUpdated: []string{"a"},
			expectedRemoved: []string{"b"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			created, updated, removed := Diff(tc.old, tc.new)
			assert.Equal(t, tc.expectedCreated, idsOf(created))
			assert.Equal(t, tc.expectedUpdated, idsOf(updated))
			assert.Equal(t, tc.expectedRemoved, idsOf(removed))
		})
	}
}

func idsOf(events []SecurityEvent) []string {
	if len(events) == 0 {
		return nil
	}
	ids := make([]string, len(events))
	for i, e := range events {
		ids[i] = e.ID
	}
	return ids
}

func TestDiff_IsIdempotentWhenAppliedTwiceToSameInputs(t *testing.T) {
	old := []SecurityEvent{newTestEvent("a", "m1")}
	new := []SecurityEvent{newTestEvent("a", "m1-changed"), newTestEvent("b", "m2")}

	created1, updated1, removed1 := Diff(old, new)
	created2, updated2, removed2 := Diff(old, new)

	assert.Equal(t, created1, created2)
	assert.Equal(t, updated1, updated2)
	assert.Equal(t, removed1, removed2)
}
