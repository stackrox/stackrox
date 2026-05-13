package k8sutil

import (
	"github.com/stackrox/rox/pkg/set"
	"k8s.io/client-go/rest"
)

// suppressedWarnings contains API server warnings that should be silently dropped.
var suppressedWarnings = set.NewFrozenStringSet(
	"apps.openshift.io/v1 DeploymentConfig is deprecated in v4.14+, unavailable in v4.10000+",
)

// filteredWarningHandler delegates to the default WarningLogger but suppresses
// known-noisy API server deprecation warnings that cannot be avoided.
type filteredWarningHandler struct {
	delegate rest.WarningHandler
}

func (h filteredWarningHandler) HandleWarningHeader(code int, agent string, text string) {
	if suppressedWarnings.Contains(text) {
		return
	}
	h.delegate.HandleWarningHeader(code, agent, text)
}

// NewFilteredWarningHandler returns a WarningHandler that suppresses known
// noisy deprecation warnings while forwarding everything else to the default
// client-go WarningLogger.
func NewFilteredWarningHandler() rest.WarningHandler {
	return filteredWarningHandler{delegate: rest.WarningLogger{}}
}
