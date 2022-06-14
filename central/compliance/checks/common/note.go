package common

import (
	"fmt"

	"github.com/stackrox/rox/central/compliance/framework"
	"github.com/stackrox/rox/generated/storage"
	pkgFramework "github.com/stackrox/rox/pkg/compliance/framework"
)

// PerNodeNoteCheck marks every node with a NoteStatus with the evidence being the description
func PerNodeNoteCheck(id, desc string) framework.Check {
	md := framework.CheckMetadata{
		ID:                 id,
		Scope:              pkgFramework.NodeKind,
		InterpretationText: fmt.Sprintf("The following property cannot be checked automatically by StackRox, and thus must be ensured manually: %s", desc),
	}
	return framework.NewCheckFromFunc(md, func(ctx framework.ComplianceContext) {
		framework.ForEachNode(ctx, func(ctx framework.ComplianceContext, node *storage.Node) {
			framework.NoteNow(ctx, "Requires manual introspection: "+desc)
		})
	})
}

// PerNodeDeprecatedCheck marks every node with a SkipStatus with the evidence being the description
func PerNodeDeprecatedCheck(id, desc string) framework.Check {
	md := framework.CheckMetadata{
		ID:                 id,
		Scope:              pkgFramework.NodeKind,
		InterpretationText: fmt.Sprintf("The check has been retired because: %s", desc),
	}
	return framework.NewCheckFromFunc(md, func(ctx framework.ComplianceContext) {
		framework.ForEachNode(ctx, func(ctx framework.ComplianceContext, node *storage.Node) {
			framework.Skipf(ctx, "Deprecated check: %s", desc)
		})
	})
}
