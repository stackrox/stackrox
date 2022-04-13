package common

import (
	"strings"

	"github.com/stackrox/stackrox/central/compliance/framework"
	"github.com/stackrox/stackrox/generated/storage"
)

func hostOperatingSystem(ctx framework.ComplianceContext, node *storage.Node) {
	osName := node.GetOsImage()
	if strings.Contains(osName, "Container") {
		framework.PassNowf(ctx, "Host is using %q operating system", osName)
	}
	framework.Notef(ctx, "Host is using %q operating system", osName)
}

// CheckKHostOperatingSystem verifies if the host is running minimal OS.
func CheckKHostOperatingSystem(ctx framework.ComplianceContext) {
	framework.ForEachNode(ctx, hostOperatingSystem)
}
