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
		},
		{
			name: "mapping with priorities that match",
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
		},
		{
			name: "mapping with priorities, but matches ordering",
			prios: []jiraLib.Priority{
				{
					ID:   "4",
					Name: "Minor",
				},
				{
					ID:   "1",
					Name: "Critical",
				},
				{
					ID:   "2",
					Name: "Major",
				},
				{
					ID:   "3",
					Name: "Normal",
				},
				{
					ID:   "5",
					Name: "Lowest",
				},
			},
			output: map[storage.Severity]string{
				storage.Severity_LOW_SEVERITY:      "Minor",
				storage.Severity_MEDIUM_SEVERITY:   "Normal",
				storage.Severity_HIGH_SEVERITY:     "Major",
				storage.Severity_CRITICAL_SEVERITY: "Critical",
			},
		},
		{
			name: "too few priorities so attribute the last couple to last priorities",
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
			output: map[storage.Severity]string{
				storage.Severity_LOW_SEVERITY:      "Major",
				storage.Severity_MEDIUM_SEVERITY:   "Major",
				storage.Severity_HIGH_SEVERITY:     "Major",
				storage.Severity_CRITICAL_SEVERITY: "Critical",
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.output, mapPriorities(c.integration, c.prios))
		})
	}
}
