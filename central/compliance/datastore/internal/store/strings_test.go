package store

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestStringsRoundTrip(t *testing.T) {
	results := &storage.ComplianceRunResults{
		ClusterResults: &storage.ComplianceRunResults_EntityResults{
			ControlResults: map[string]*storage.ComplianceResultValue{
				"ctrl1": {
					Evidence: []*storage.ComplianceResultValue_Evidence{
						{
							Message: "foo",
						},
						{
							Message: "bar",
						},
					},
				},
			},
		},
		NodeResults: map[string]*storage.ComplianceRunResults_EntityResults{
			"node1": {
				ControlResults: map[string]*storage.ComplianceResultValue{
					"ctrl2": {
						Evidence: []*storage.ComplianceResultValue_Evidence{
							{
								Message: "baz",
							},
						},
					},
				},
			},
		},
		DeploymentResults: map[string]*storage.ComplianceRunResults_EntityResults{
			"deployment1": {
				ControlResults: map[string]*storage.ComplianceResultValue{
					"ctrl3": {
						Evidence: []*storage.ComplianceResultValue_Evidence{
							{
								Message: "foo",
							},
						},
					},
				},
			},
		},
	}

	resultsWithoutStrings := results.Clone()
	stringsProto := ExternalizeStrings(resultsWithoutStrings)
	assert.ElementsMatch(t, stringsProto.Strings, []string{"foo", "bar", "baz"})

	for _, cr := range resultsWithoutStrings.GetClusterResults().GetControlResults() {
		for _, e := range cr.GetEvidence() {
			assert.Empty(t, e.GetMessage())
			assert.NotZero(t, e.GetMessageId())
		}
	}
	for _, nr := range resultsWithoutStrings.GetNodeResults() {
		for _, cr := range nr.GetControlResults() {
			for _, e := range cr.GetEvidence() {
				assert.Empty(t, e.GetMessage())
				assert.NotZero(t, e.GetMessageId())
			}
		}
	}
	for _, dr := range resultsWithoutStrings.GetNodeResults() {
		for _, cr := range dr.GetControlResults() {
			for _, e := range cr.GetEvidence() {
				assert.Empty(t, e.GetMessage())
				assert.NotZero(t, e.GetMessageId())
			}
		}
	}

	assert.True(t, ReconstituteStrings(resultsWithoutStrings, stringsProto))
	assert.Equal(t, results, resultsWithoutStrings)
}
