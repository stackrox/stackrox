package notifiers

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/assert"
)

func getProcessIndicatorWithID(id string) *storage.ProcessIndicator {
	p := fixtures.GetProcessIndicator()
	p.Id = id
	return p

}

func TestFilterProcesses(t *testing.T) {
	cases := []struct {
		name              string
		processes         []*storage.ProcessIndicator
		currSize, maxSize int
		expectedNumber    int
	}{
		{
			name: "no trim",
			processes: []*storage.ProcessIndicator{
				getProcessIndicatorWithID("A"),
				getProcessIndicatorWithID("B"),
				getProcessIndicatorWithID("C"),
			},
			currSize:       100,
			maxSize:        10000,
			expectedNumber: 3,
		},
		{
			name: "trim",
			processes: []*storage.ProcessIndicator{
				getProcessIndicatorWithID("A"),
				getProcessIndicatorWithID("B"),
				getProcessIndicatorWithID("C"),
			},
			expectedNumber: 2,
			currSize:       400,
			maxSize:        300,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for _, p := range c.processes {
				cleanProcessIndicator(p)
			}
			assert.Equal(t, c.processes[:c.expectedNumber], filterProcesses(c.processes, c.maxSize, &c.currSize))
		})
	}
}

func TestFilterViolations(t *testing.T) {
	cases := []struct {
		name              string
		violations        []*storage.Alert_Violation
		currSize, maxSize int
		expectedNumber    int
	}{
		{
			name: "no trim",
			violations: []*storage.Alert_Violation{
				{
					Message: "message a",
				},
				{
					Message: "message b",
				},
				{
					Message: "message c",
				},
				{
					Message: "message d",
				},
			},
			currSize:       100,
			maxSize:        10000,
			expectedNumber: 4,
		},
		{
			name: "trim",
			violations: []*storage.Alert_Violation{
				{
					Message: "message a",
				},
				{
					Message: "message b",
				},
				{
					Message: "message c",
				},
				{
					Message: "message d",
				},
			},
			currSize:       140,
			maxSize:        100,
			expectedNumber: 2,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.violations[:c.expectedNumber], filterViolations(c.violations, c.maxSize, &c.currSize))
		})
	}
}
