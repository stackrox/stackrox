package store

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
)

func TestStringsRoundTrip(t *testing.T) {
	results := storage.ComplianceRunResults_builder{
		ClusterResults: storage.ComplianceRunResults_EntityResults_builder{
			ControlResults: map[string]*storage.ComplianceResultValue{
				"ctrl1": storage.ComplianceResultValue_builder{
					Evidence: []*storage.ComplianceResultValue_Evidence{
						storage.ComplianceResultValue_Evidence_builder{
							Message: "foo",
						}.Build(),
						storage.ComplianceResultValue_Evidence_builder{
							Message: "bar",
						}.Build(),
					},
				}.Build(),
			},
		}.Build(),
		NodeResults: map[string]*storage.ComplianceRunResults_EntityResults{
			"node1": storage.ComplianceRunResults_EntityResults_builder{
				ControlResults: map[string]*storage.ComplianceResultValue{
					"ctrl2": storage.ComplianceResultValue_builder{
						Evidence: []*storage.ComplianceResultValue_Evidence{
							storage.ComplianceResultValue_Evidence_builder{
								Message: "baz",
							}.Build(),
						},
					}.Build(),
				},
			}.Build(),
		},
		DeploymentResults: map[string]*storage.ComplianceRunResults_EntityResults{
			"deployment1": storage.ComplianceRunResults_EntityResults_builder{
				ControlResults: map[string]*storage.ComplianceResultValue{
					"ctrl3": storage.ComplianceResultValue_builder{
						Evidence: []*storage.ComplianceResultValue_Evidence{
							storage.ComplianceResultValue_Evidence_builder{
								Message: "foo",
							}.Build(),
						},
					}.Build(),
				},
			}.Build(),
		},
	}.Build()

	resultsWithoutStrings := results.CloneVT()
	stringsProto := ExternalizeStrings(resultsWithoutStrings)
	assert.ElementsMatch(t, stringsProto.GetStrings(), []string{"foo", "bar", "baz"})

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
	protoassert.Equal(t, results, resultsWithoutStrings)
}
