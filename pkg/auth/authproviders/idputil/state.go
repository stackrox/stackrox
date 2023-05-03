package idputil

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/netutil"
)

const (
	// TestLoginClientState is the state value indicating a test login flow. DO NOT CHANGE.
	TestLoginClientState = "e003ba41-9cc1-48ee-b6a9-2dd7c21da92e"
	// AuthorizeRoxctlClientState is the state value indicating a roxctl authorization flow. DO NOT CHANGE.
	AuthorizeRoxctlClientState = "2ed17ca6-4b3c-4279-8317-f26f8ba01c52"
)

// AuthMode is the authentication mode.
type AuthMode int

// Authentication modes currently supported.
const (
	LoginAuthMode AuthMode = iota
	TestAuthMode
	AuthorizeRoxctlMode
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

// attachTestState prefixes the clientState with test state or empty string if not a test mode.
func attachTestState(clientState string) string {
	return fmt.Sprintf("%s#%s", TestLoginClientState, clientState)
}

// attachAuthorizeRoxctlState prefixes the callback URL with the AuthorizeRoxctlClientState.
func attachAuthorizeRoxctlState(callBackURL string) (string, error) {
	parsedCallbackURL, err := url.Parse(callBackURL)
	if err != nil {
		return "", errors.Wrapf(err, "unable to parse URL %q", callBackURL)
	}
	if !netutil.IsLocalHost(parsedCallbackURL.Hostname()) {
		return "", errox.InvalidArgs.New("roxctl authorization is only allowed for localhost as callback target")
	}
	return fmt.Sprintf("%s#%s", AuthorizeRoxctlClientState, parsedCallbackURL), nil
}

// AttachStateOrEmpty may modify the given clientState.
// In case testMode == true, the clientState will be prefixed with TestLoginClientState and returned.
// In case callbackURL != "", the clientState will be AuthorizeRoxctlClientState and the parsed callback URL.
// If both are default values (testMode == false && callbackURL == ""), the clientState will be returned.
func AttachStateOrEmpty(clientState string, testMode bool, callbackURL string) (string, error) {
	if testMode && callbackURL != "" {
		return "", errox.InvalidArgs.New("cannot use test mode and roxctl authorize in conjunction")
	}

	if testMode {
		return attachTestState(clientState), nil
	}

	if callbackURL != "" {
		return attachAuthorizeRoxctlState(callbackURL)
	}

	return clientState, nil
}

// ParseClientState parses the clientState and removes test login state in present
func ParseClientState(clientState string) (string, AuthMode) {
	parts := strings.SplitN(clientState, "#", 2)
	if len(parts) == 0 {
		return "", LoginAuthMode
	}

	switch parts[0] {
	case "":
		return parts[len(parts)-1], LoginAuthMode
	case TestLoginClientState:
		if len(parts) == 1 {
			return "", TestAuthMode
		}
		return parts[1], TestAuthMode
	case AuthorizeRoxctlClientState:
		if len(parts) == 1 {
			return "", AuthorizeRoxctlMode
		}
		return parts[1], AuthorizeRoxctlMode
	default:
		return clientState, LoginAuthMode
	}
}
