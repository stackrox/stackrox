package idputil

import (
	"fmt"
	"strings"
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
