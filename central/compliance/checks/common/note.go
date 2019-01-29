package common

import (
	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/storage"
)

// PerNodeNoteCheck marks every node with a NoteStatus with the evidence being the description
func PerNodeNoteCheck(id, desc string) framework.Check {
	return framework.NewCheckFromFunc(id, framework.NodeKind, nil, func(ctx framework.ComplianceContext) {
		framework.ForEachNode(ctx, func(ctx framework.ComplianceContext, node *storage.Node) {
			framework.NoteNow(ctx, "Requires manual introspection: "+desc)
		})
	})
}
