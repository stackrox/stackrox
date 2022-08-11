package idputil

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

// AuthnMode is the authentication mode
type AuthnMode int

// Authentication mode values
const (
	AuthnModeLogin AuthnMode = iota
	AuthnModeTest
	AuthnModeAuthorizeCLI
)

const (
	// TestLoginClientState is the state value indicating a test login flow. DO NOT CHANGE.
	TestLoginClientState = "e003ba41-9cc1-48ee-b6a9-2dd7c21da92e"

	// AuthorizeCLIClientState bla
	AuthorizeCLIClientState = "2a71b0dd-ff43-483a-a52a-3d6a8a02f4eb"
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

// AuthorizeCLICallbackURLState bla
func AuthorizeCLICallbackURLState(callbackURL string) (string, error) {
	urlParsed, err := url.Parse(callbackURL)
	if err != nil {
		return "", errors.Wrap(err, "unparseable callback URL")
	}
	if urlParsed.Hostname() != "localhost" && urlParsed.Hostname() != "127.0.0.1" {
		return "", errors.Wrap(err, "only localhost is allowed as a CLI authorization callback target")
	}
	return fmt.Sprintf("%s#%s", AuthorizeCLIClientState, urlParsed), nil
}

// ParseClientState parses the clientState and removes test login state in present
func ParseClientState(clientState string) (string, AuthnMode) {
	parts := strings.SplitN(clientState, "#", 2)
	if len(parts) == 0 {
		return "", AuthnModeLogin
	}

	if parts[0] == "" {
		return parts[len(parts)-1], AuthnModeLogin
	}

	if parts[0] == TestLoginClientState {
		if len(parts) == 1 {
			return "", AuthnModeTest
		}
		return parts[1], AuthnModeTest
	} else if parts[0] == AuthorizeCLIClientState {
		if len(parts) == 1 {
			return "", AuthnModeAuthorizeCLI
		}
		return parts[1], AuthnModeAuthorizeCLI
	}
	// if AttachTestStateOrEmpty was not called before ParseClientState, we have actually valid clientState
	return clientState, AuthnModeLogin
}
