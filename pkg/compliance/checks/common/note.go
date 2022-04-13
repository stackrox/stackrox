package common

import (
	"fmt"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/compliance/checks/standards"
	"github.com/stackrox/stackrox/pkg/compliance/framework"
)

// NoteCheck marks every node with a NoteStatus with the evidence being the description
func NoteCheck(desc string) *standards.CheckAndMetadata {
	return &standards.CheckAndMetadata{
		CheckFunc: func(complianceData *standards.ComplianceData) []*storage.ComplianceResultValue_Evidence {
			return NoteList("Requires manual introspection: " + desc)
		},
		Metadata: &standards.Metadata{
			InterpretationText: fmt.Sprintf("The following property cannot be checked automatically by StackRox, and thus must be ensured manually: %s", desc),
			TargetKind:         framework.NodeKind,
		},
	}
}
