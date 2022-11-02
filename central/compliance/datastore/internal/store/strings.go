package store

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
)

type stringCollector struct {
	stringsProto  *storage.ComplianceStrings
	stringIndices map[string]int
}

func newStringCollector(runID string) *stringCollector {
	return &stringCollector{
		stringsProto: &storage.ComplianceStrings{
			Id: runID,
		},
		stringIndices: make(map[string]int),
	}
}

func (c *stringCollector) Collect(s string) int {
	idx, ok := c.stringIndices[s]
	if !ok {
		idx = len(c.stringsProto.Strings)
		c.stringsProto.Strings = append(c.stringsProto.Strings, s)
		c.stringIndices[s] = idx
	}
	return idx
}

// ExternalizeStrings modifies resultsProto to contain only empty `Message` fields in the evidence proto, and creates
// and returns a `ComplianceStrings` proto that contains the strings and allows looking up the original message strings
// through the newly populated `MessageId` field in the evidence record.
func ExternalizeStrings(resultsProto *storage.ComplianceRunResults) *storage.ComplianceStrings {
	sc := newStringCollector(resultsProto.GetRunMetadata().GetRunId())
	externalizeStringsForEntity(resultsProto.GetClusterResults(), sc)
	for _, deploymentResults := range resultsProto.GetDeploymentResults() {
		externalizeStringsForEntity(deploymentResults, sc)
	}
	for _, nodeResults := range resultsProto.GetNodeResults() {
		externalizeStringsForEntity(nodeResults, sc)
	}
	return sc.stringsProto
}

func externalizeStringsForEntity(entityResults *storage.ComplianceRunResults_EntityResults, strings *stringCollector) {
	for _, resultVal := range entityResults.GetControlResults() {
		for _, e := range resultVal.GetEvidence() {
			if e.Message == "" {
				continue
			}
			e.MessageId = int32(strings.Collect(e.Message)) + 1
			e.Message = ""
		}
	}
}

// ReconstituteStrings populates all messages in the evidence records of the given result, by looking up the string
// value for the message ID in the given strings proto.
func ReconstituteStrings(resultsProto *storage.ComplianceRunResults, stringsProto *storage.ComplianceStrings) bool {
	allFound := reconstituteStringsForEntity(resultsProto.GetClusterResults(), stringsProto)
	for _, deploymentResults := range resultsProto.GetDeploymentResults() {
		allFound = reconstituteStringsForEntity(deploymentResults, stringsProto) && allFound
	}
	for _, nodeResults := range resultsProto.GetNodeResults() {
		allFound = reconstituteStringsForEntity(nodeResults, stringsProto) && allFound
	}
	return allFound
}

func reconstituteStringsForEntity(entityResults *storage.ComplianceRunResults_EntityResults, stringsProto *storage.ComplianceStrings) bool {
	allFound := true
	for _, resultVal := range entityResults.GetControlResults() {
		for _, e := range resultVal.GetEvidence() {
			if e.GetMessage() != "" || e.GetMessageId() == 0 {
				continue
			}
			idx := int(e.GetMessageId()) - 1
			var msg string
			if idx < 0 || idx >= len(stringsProto.Strings) {
				msg = fmt.Sprintf("#Invalid message ID %d", idx)
				allFound = false
			} else {
				msg = stringsProto.Strings[idx]
			}
			e.Message = msg
			e.MessageId = 0
		}
	}
	return allFound
}
