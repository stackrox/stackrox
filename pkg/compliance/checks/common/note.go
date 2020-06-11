package common

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compliance/checks/standards"
)

// NoteCheck marks every node with a NoteStatus with the evidence being the description
func NoteCheck(desc string) *standards.CheckAndInterpretation {
	return &standards.CheckAndInterpretation{
		CheckFunc: func(complianceData *standards.ComplianceData) []*storage.ComplianceResultValue_Evidence {
			return NoteList("Requires manual introspection: " + desc)
		},
		InterpretationText: fmt.Sprintf("The following property cannot be checked automatically by StackRox, and thus must be ensured manually: %s", desc),
	}
}
