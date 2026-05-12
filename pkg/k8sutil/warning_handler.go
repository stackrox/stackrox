package k8sutil

import (
	"strings"

	"k8s.io/client-go/rest"
)

// suppressedWarnings lists API server warning substrings that should be silently dropped.
var suppressedWarnings = []string{
	"DeploymentConfig is deprecated",
}

// filteredWarningHandler delegates to the default WarningLogger but suppresses
// known-noisy API server deprecation warnings that cannot be avoided.
type filteredWarningHandler struct {
	delegate rest.WarningHandler
}

func (h filteredWarningHandler) HandleWarningHeader(code int, agent string, text string) {
	for _, substr := range suppressedWarnings {
		if strings.Contains(text, substr) {
			return
		}
	}
	h.delegate.HandleWarningHeader(code, agent, text)
}

// NewFilteredWarningHandler returns a WarningHandler that suppresses known
// noisy deprecation warnings while forwarding everything else to the default
// client-go WarningLogger.
func NewFilteredWarningHandler() rest.WarningHandler {
	return filteredWarningHandler{delegate: rest.WarningLogger{}}
}
