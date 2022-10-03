package debug

import (
	"strings"
)

// ROX12096 returns true if deployment name matches problematic deployment name from SAC E2E test.s
// This code is only for debug purposes and should be removed when issue with missing deployments is resolved.
// TODO(ROX-12096): Remove when resolved.
func ROX12096(log func(template string, args ...interface{}), name string, template string, args ...interface{}) {
	if strings.Contains(name, "sac-deploymentnginx-qa") {
		log(template, args...)
	}
}
