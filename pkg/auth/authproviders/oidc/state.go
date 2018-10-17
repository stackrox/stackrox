package oidc

import (
	"fmt"
	"strings"
)

func makeState(providerID, clientState string) string {
	return fmt.Sprintf("%s:%s", providerID, clientState)
}

func splitState(state string) (providerID, clientState string) {
	parts := strings.SplitN(state, ":", 2)
	if len(parts) != 2 {
		return
	}
	providerID = parts[0]
	clientState = parts[1]
	return
}
