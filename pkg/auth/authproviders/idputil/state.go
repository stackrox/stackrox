package idputil

import (
	"fmt"
	"strings"
)

const (
	// TestLoginClientState is the state value indicating a test login flow. DO NOT CHANGE.
	TestLoginClientState = "e003ba41-9cc1-48ee-b6a9-2dd7c21da92e"
)

// MakeState constructs a `state` value out of the given auth provider ID and a backend-specific state value.
func MakeState(providerID, clientState string) string {
	return fmt.Sprintf("%s:%s", providerID, clientState)
}

// SplitState splits a state that was created via `MakeState` into an auth provider ID and a backend-specific state
// value.
func SplitState(state string) (providerID, clientState string) {
	parts := strings.SplitN(state, ":", 2)
	for len(parts) < 2 {
		parts = append(parts, "")
	}
	providerID = parts[0]
	clientState = parts[1]
	return
}

// AttachTestStateOrEmpty prefixes the clientState with test state or empty string if not a test mode.
func AttachTestStateOrEmpty(clientState string, testMode bool) string {
	prefixState := ""
	if testMode {
		prefixState = TestLoginClientState
	}
	return fmt.Sprintf("%s#%s", prefixState, clientState)
}

// ParseClientState parses the clientState and removes test login state in present
func ParseClientState(clientState string) (string, bool) {
	parts := strings.SplitN(clientState, "#", 2)
	if len(parts) == 0 {
		return "", false
	}

	if parts[0] == "" {
		return parts[len(parts)-1], false
	}

	if parts[0] == TestLoginClientState {
		if len(parts) == 1 {
			return "", true
		}
		return parts[1], true
	}
	// if AttachTestStateOrEmpty was not called before ParseClientState, we have actually valid clientState
	return clientState, false
}
