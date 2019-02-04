package common

import (
	"fmt"

	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/storage"
)

// PerNodeNoteCheck marks every node with a NoteStatus with the evidence being the description
func PerNodeNoteCheck(id, desc string) framework.Check {
	md := framework.CheckMetadata{
		ID:                 id,
		Scope:              framework.NodeKind,
		InterpretationText: fmt.Sprintf("The following property cannot be checked automatically by StackRox, and thus must be ensured manually: %s", desc),
	}
	return framework.NewCheckFromFunc(md, func(ctx framework.ComplianceContext) {
		framework.ForEachNode(ctx, func(ctx framework.ComplianceContext, node *storage.Node) {
			framework.NoteNow(ctx, "Requires manual introspection: "+desc)
		})
	})
}
