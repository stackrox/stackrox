package jira

import (
	"testing"

	jiraLib "github.com/andygrunwald/go-jira"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestSeveritiesSlice(t *testing.T) {
	assert.Equal(t, len(storage.Severity_name)-1, len(severities))
}

func TestMapPriorities(t *testing.T) {
	cases := []struct {
		name        string
		prios       []jiraLib.Priority
		integration *storage.Jira
		output      map[storage.Severity]string
		shouldError bool
	}{
		{
			name:  "mapping with no priorities",
			prios: nil,
			integration: &storage.Jira{
				PriorityMappings: []*storage.Jira_PriorityMapping{
					{
						Severity:     storage.Severity_LOW_SEVERITY,
						PriorityName: "1",
					},
					{
						Severity:     storage.Severity_MEDIUM_SEVERITY,
						PriorityName: "2",
					},
					{
						Severity:     storage.Severity_HIGH_SEVERITY,
						PriorityName: "3",
					},
					{
						Severity:     storage.Severity_CRITICAL_SEVERITY,
						PriorityName: "4",
					},
				},
			},
			output: map[storage.Severity]string{
				storage.Severity_LOW_SEVERITY:      "1",
				storage.Severity_MEDIUM_SEVERITY:   "2",
				storage.Severity_HIGH_SEVERITY:     "3",
				storage.Severity_CRITICAL_SEVERITY: "4",
			},
			// It's a problem if we can't fetch priorities from Jira
			shouldError: true,
		},
		{
			name: "mapping with priorities but no mapping config",
			prios: []jiraLib.Priority{
				{
					ID:   "4",
					Name: "P3-Mumble",
				},
				{
					ID:   "1",
					Name: "P0/Blocker",
				},
				{
					ID:   "2",
					Name: "P1/Major",
				},
				{
					ID:   "3",
					Name: "P2/Normal",
				},
				{
					ID:   "5",
					Name: "P4 Lowest",
				},
			},
			output: map[storage.Severity]string{
				storage.Severity_LOW_SEVERITY:      "P3-Mumble",
				storage.Severity_MEDIUM_SEVERITY:   "P2/Normal",
				storage.Severity_HIGH_SEVERITY:     "P1/Major",
				storage.Severity_CRITICAL_SEVERITY: "P0/Blocker",
			},
			// No mapping in the notifier config is an error
			shouldError: true,
		},
		{
			name: "mapping with priorities",
			prios: []jiraLib.Priority{
				{
					ID:   "4",
					Name: "P3-Mumble",
				},
				{
					ID:   "1",
					Name: "P0/Blocker",
				},
				{
					ID:   "2",
					Name: "P1/Major",
				},
				{
					ID:   "3",
					Name: "P2/Normal",
				},
				{
					ID:   "5",
					Name: "P4 Lowest",
				},
			},
			integration: &storage.Jira{
				PriorityMappings: []*storage.Jira_PriorityMapping{
					{
						Severity:     storage.Severity_LOW_SEVERITY,
						PriorityName: "P3-Mumble",
					},
					{
						Severity:     storage.Severity_MEDIUM_SEVERITY,
						PriorityName: "P2/Normal",
					},
					{
						Severity:     storage.Severity_HIGH_SEVERITY,
						PriorityName: "P1/Major",
					},
					{
						Severity:     storage.Severity_CRITICAL_SEVERITY,
						PriorityName: "P0/Blocker",
					},
				},
			},
			output: map[storage.Severity]string{
				storage.Severity_LOW_SEVERITY:      "P3-Mumble",
				storage.Severity_MEDIUM_SEVERITY:   "P2/Normal",
				storage.Severity_HIGH_SEVERITY:     "P1/Major",
				storage.Severity_CRITICAL_SEVERITY: "P0/Blocker",
			},
			shouldError: false,
		},
		{
			name: "too few priorities",
			prios: []jiraLib.Priority{
				{
					ID:   "2",
					Name: "Major",
				},
				{
					ID:   "1",
					Name: "Critical",
				},
			},
			integration: &storage.Jira{
				PriorityMappings: []*storage.Jira_PriorityMapping{
					{
						Severity:     storage.Severity_LOW_SEVERITY,
						PriorityName: "Minor",
					},
					{
						Severity:     storage.Severity_MEDIUM_SEVERITY,
						PriorityName: "Normal",
					},
					{
						Severity:     storage.Severity_HIGH_SEVERITY,
						PriorityName: "Major",
					},
					{
						Severity:     storage.Severity_CRITICAL_SEVERITY,
						PriorityName: "Critical",
					},
				},
			},
			output: nil,
			// Every priority referenced in mapping has to exist in Jira
			shouldError: true,
		},
		{
			name: "partial mapping",
			prios: []jiraLib.Priority{
				{
					ID:   "4",
					Name: "Minor",
				},
				{
					ID:   "3",
					Name: "Normal",
				},
				{
					ID:   "2",
					Name: "Major",
				},
				{
					ID:   "1",
					Name: "Critical",
				},
			},
			integration: &storage.Jira{
				PriorityMappings: []*storage.Jira_PriorityMapping{
					{
						Severity:     storage.Severity_MEDIUM_SEVERITY,
						PriorityName: "Normal",
					},
					{
						Severity:     storage.Severity_HIGH_SEVERITY,
						PriorityName: "Major",
					},
					{
						Severity:     storage.Severity_CRITICAL_SEVERITY,
						PriorityName: "Critical",
					},
				},
			},
			// Note that this is not really valid, but the Validate function in jira.go should catch
			// the case where there aren't enough in the storage mappings
			output: map[storage.Severity]string{
				storage.Severity_MEDIUM_SEVERITY:   "Normal",
				storage.Severity_HIGH_SEVERITY:     "Major",
				storage.Severity_CRITICAL_SEVERITY: "Critical",
			},
			shouldError: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mappedSeverities, err := mapPriorities(c.prios, c.integration.GetPriorityMappings())
			if c.shouldError {
				assert.Error(t, err)
			} else {
				assert.Equal(t, c.output, mappedSeverities)
			}
		})
	}
}
