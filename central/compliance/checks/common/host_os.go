package common

import (
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/stackrox/rox/central/compliance/framework"
	internalTypes "github.com/stackrox/rox/pkg/docker/types"
)

func hostOperatingSystem(ctx framework.ComplianceContext, info types.Info) {
	if strings.Contains(info.OperatingSystem, "Container") {
		framework.PassNowf(ctx, "Host is using %q operating system", info.OperatingSystem)
	}
	framework.Notef(ctx, "Host is using %q operating system", info.OperatingSystem)
}

// CheckKHostOperatingSystem verifies if the host is running minimal OS.
func CheckKHostOperatingSystem(ctx framework.ComplianceContext) {
	PerNodeCheckWithDockerData(func(ctx framework.ComplianceContext, data *internalTypes.Data) {
		hostOperatingSystem(ctx, data.Info)
	})(ctx)
}
