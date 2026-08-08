package idputil

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSplitState tests the SplitState function with known inputs
func TestSplitState(t *testing.T) {
	tests := map[string]struct {
		input               string
		expectedProviderID  string
		expectedClientState string
	}{
		"normal state with provider and client state": {
			input:               "provider-123:client-state-456",
			expectedProviderID:  "provider-123",
			expectedClientState: "client-state-456",
		},
		"state with multiple colons": {
			input:               "provider:client:state:with:colons",
			expectedProviderID:  "provider",
			expectedClientState: "client:state:with:colons",
		},
		"state with only provider": {
			input:               "provider-only",
			expectedProviderID:  "provider-only",
			expectedClientState: "",
		},
		"state with empty provider": {
			input:               ":client-state",
			expectedProviderID:  "",
			expectedClientState: "client-state",
		},
		"empty state": {
			input:               "",
			expectedProviderID:  "",
			expectedClientState: "",
		},
		"state with only colon": {
			input:               ":",
			expectedProviderID:  "",
			expectedClientState: "",
		},
		"state with UUID-like values": {
			input:               "oidc-provider:e003ba41-9cc1-48ee-b6a9-2dd7c21da92e",
			expectedProviderID:  "oidc-provider",
			expectedClientState: "e003ba41-9cc1-48ee-b6a9-2dd7c21da92e",
		},
		"state with URL in client state": {
			input:               "saml:https://example.com/callback?param=value",
			expectedProviderID:  "saml",
			expectedClientState: "https://example.com/callback?param=value",
		},
		"state with special characters": {
			input:               "provider#123:client@state!456",
			expectedProviderID:  "provider#123",
			expectedClientState: "client@state!456",
		},
		"state with whitespace": {
			input:               "provider with spaces:client state",
			expectedProviderID:  "provider with spaces",
			expectedClientState: "client state",
		},
		"state with newlines": {
			input:               "provider\n:client\nstate",
			expectedProviderID:  "provider\n",
			expectedClientState: "client\nstate",
		},
		"state with tabs": {
			input:               "provider\t:client\tstate",
			expectedProviderID:  "provider\t",
			expectedClientState: "client\tstate",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			providerID, clientState := SplitState(tc.input)
			assert.Equal(t, tc.expectedProviderID, providerID)
			assert.Equal(t, tc.expectedClientState, clientState)
		})
	}
}

// TestParseClientState tests the ParseClientState function with known inputs
func TestParseClientState(t *testing.T) {
	tests := map[string]struct {
		input            string
		expectedState    string
		expectedAuthMode AuthMode
	}{
		"normal client state without test mode": {
			input:            "normal-client-state",
			expectedState:    "normal-client-state",
			expectedAuthMode: LoginAuthMode,
		},
		"test mode with client state": {
			input:            TestLoginClientState + "#some-client-state",
			expectedState:    "some-client-state",
			expectedAuthMode: TestAuthMode,
		},
		"test mode without client state": {
			input:            TestLoginClientState,
			expectedState:    "",
			expectedAuthMode: TestAuthMode,
		},
		"roxctl authorize mode with callback URL": {
			input:            AuthorizeRoxctlClientState + "#http://localhost:8000/callback",
			expectedState:    "http://localhost:8000/callback",
			expectedAuthMode: AuthorizeRoxctlMode,
		},
		"roxctl authorize mode without callback URL": {
			input:            AuthorizeRoxctlClientState,
			expectedState:    "",
			expectedAuthMode: AuthorizeRoxctlMode,
		},
		"empty state": {
			input:            "",
			expectedState:    "",
			expectedAuthMode: LoginAuthMode,
		},
		"state with hash but not test or roxctl": {
			input:            "random#state#with#hashes",
			expectedState:    "random#state#with#hashes",
			expectedAuthMode: LoginAuthMode,
		},
		"state starting with hash": {
			input:            "#some-state",
			expectedState:    "some-state",
			expectedAuthMode: LoginAuthMode,
		},
		"state with only hash": {
			input:            "#",
			expectedState:    "",
			expectedAuthMode: LoginAuthMode,
		},
		"test mode with multiple hashes": {
			input:            TestLoginClientState + "#client#state#with#hashes",
			expectedState:    "client#state#with#hashes",
			expectedAuthMode: TestAuthMode,
		},
		"roxctl mode with multiple hashes": {
			input:            AuthorizeRoxctlClientState + "#http://localhost:8000#extra",
			expectedState:    "http://localhost:8000#extra",
			expectedAuthMode: AuthorizeRoxctlMode,
		},
		"state with special characters": {
			input:            "client!@$%^&*()state",
			expectedState:    "client!@$%^&*()state",
			expectedAuthMode: LoginAuthMode,
		},
		"state with whitespace": {
			input:            "client state with spaces",
			expectedState:    "client state with spaces",
			expectedAuthMode: LoginAuthMode,
		},
		"state with newlines": {
			input:            "client\nstate\n",
			expectedState:    "client\nstate\n",
			expectedAuthMode: LoginAuthMode,
		},
		"test mode with URL-encoded characters": {
			input:            TestLoginClientState + "#https%3A%2F%2Fexample.com",
			expectedState:    "https%3A%2F%2Fexample.com",
			expectedAuthMode: TestAuthMode,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			state, mode := ParseClientState(tc.input)
			assert.Equal(t, tc.expectedState, state)
			assert.Equal(t, tc.expectedAuthMode, mode)
		})
	}
}

// TestMakeAndSplitStateRoundTrip tests that MakeState and SplitState are inverse operations
func TestMakeAndSplitStateRoundTrip(t *testing.T) {
	tests := map[string]struct {
		providerID  string
		clientState string
	}{
		"normal values": {
			providerID:  "oidc-provider",
			clientState: "client-state-123",
		},
		"empty provider": {
			providerID:  "",
			clientState: "client-state",
		},
		"empty client state": {
			providerID:  "provider",
			clientState: "",
		},
		"both empty": {
			providerID:  "",
			clientState: "",
		},
		"client state with colons": {
			providerID:  "provider",
			clientState: "client:state:with:colons",
		},
		"provider with special chars": {
			providerID:  "provider-id-123",
			clientState: "https://example.com/callback?param=value",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			state := MakeState(tc.providerID, tc.clientState)
			gotProviderID, gotClientState := SplitState(state)
			assert.Equal(t, tc.providerID, gotProviderID)
			assert.Equal(t, tc.clientState, gotClientState)
		})
	}
}

// FuzzSplitState fuzzes the SplitState function to ensure it never panics
func FuzzSplitState(f *testing.F) {
	// Seed corpus from test cases
	f.Add("")
	f.Add(":")
	f.Add("provider:client")
	f.Add("provider-only")
	f.Add(":client-state")
	f.Add("provider:client:state:with:colons")
	f.Add("oidc-provider:e003ba41-9cc1-48ee-b6a9-2dd7c21da92e")
	f.Add("saml:https://example.com/callback?param=value")
	f.Add("provider#123:client@state!456")
	f.Add("provider with spaces:client state")
	f.Add("provider\n:client\nstate")
	f.Add("provider\t:client\tstate")
	f.Add(strings.Repeat("a", 1000) + ":" + strings.Repeat("b", 1000))
	f.Add(strings.Repeat(":", 100))
	f.Add("\x00\x01\x02:state")
	f.Add("provider:\x00\x01\x02")

	f.Fuzz(func(t *testing.T, input string) {
		// This should never panic
		providerID, clientState := SplitState(input)

		// Verify basic properties
		// 1. If input contains ":", the providerID should be everything before first ":"
		if idx := strings.Index(input, ":"); idx >= 0 {
			assert.Equal(t, input[:idx], providerID)
			assert.Equal(t, input[idx+1:], clientState)
		} else {
			// If no ":", providerID should be the input and clientState should be empty
			assert.Equal(t, input, providerID)
			assert.Equal(t, "", clientState)
		}

		// 2. Round-trip property: MakeState(SplitState(x)) should reconstruct the original
		//    if and only if the original contains at least one ":"
		reconstructed := MakeState(providerID, clientState)
		if strings.Contains(input, ":") {
			assert.Equal(t, input, reconstructed)
		}
	})
}

// FuzzParseClientState fuzzes the ParseClientState function to ensure it never panics
func FuzzParseClientState(f *testing.F) {
	// Seed corpus from test cases
	f.Add("")
	f.Add("#")
	f.Add("normal-client-state")
	f.Add(TestLoginClientState)
	f.Add(TestLoginClientState + "#")
	f.Add(TestLoginClientState + "#some-client-state")
	f.Add(AuthorizeRoxctlClientState)
	f.Add(AuthorizeRoxctlClientState + "#")
	f.Add(AuthorizeRoxctlClientState + "#http://localhost:8000/callback")
	f.Add("random#state#with#hashes")
	f.Add("#some-state")
	f.Add(TestLoginClientState + "#client#state#with#hashes")
	f.Add("client!@$%^&*()state")
	f.Add("client state with spaces")
	f.Add("client\nstate\n")
	f.Add(strings.Repeat("a", 1000))
	f.Add(strings.Repeat("#", 100))
	f.Add("\x00\x01\x02#state")
	f.Add("state#\x00\x01\x02")

	f.Fuzz(func(t *testing.T, input string) {
		// This should never panic
		state, mode := ParseClientState(input)

		// Verify basic properties
		// 1. The returned mode should always be one of the valid modes
		assert.True(t, mode == LoginAuthMode || mode == TestAuthMode || mode == AuthorizeRoxctlMode,
			"mode should be a valid AuthMode")

		// 2. If input is empty, state should be empty and mode should be LoginAuthMode
		if input == "" {
			assert.Equal(t, "", state)
			assert.Equal(t, LoginAuthMode, mode)
			return
		}

		// 3. If input starts with TestLoginClientState, mode should be TestAuthMode
		if strings.HasPrefix(input, TestLoginClientState) {
			assert.Equal(t, TestAuthMode, mode)
			if input == TestLoginClientState {
				assert.Equal(t, "", state)
			} else if strings.HasPrefix(input, TestLoginClientState+"#") {
				assert.Equal(t, input[len(TestLoginClientState)+1:], state)
			}
			return
		}

		// 4. If input starts with AuthorizeRoxctlClientState, mode should be AuthorizeRoxctlMode
		if strings.HasPrefix(input, AuthorizeRoxctlClientState) {
			assert.Equal(t, AuthorizeRoxctlMode, mode)
			if input == AuthorizeRoxctlClientState {
				assert.Equal(t, "", state)
			} else if strings.HasPrefix(input, AuthorizeRoxctlClientState+"#") {
				assert.Equal(t, input[len(AuthorizeRoxctlClientState)+1:], state)
			}
			return
		}

		// 5. If input starts with "#", the state should be everything after the first "#"
		if strings.HasPrefix(input, "#") {
			assert.Equal(t, LoginAuthMode, mode)
			parts := strings.SplitN(input, "#", 2)
			if len(parts) == 2 {
				assert.Equal(t, parts[1], state)
			}
			return
		}

		// 6. Otherwise, if no "#" in input, state should equal input and mode should be LoginAuthMode
		if !strings.Contains(input, "#") {
			assert.Equal(t, input, state)
			assert.Equal(t, LoginAuthMode, mode)
			return
		}

		// 7. For all other cases (contains "#" but doesn't match special prefixes),
		//    state should equal input and mode should be LoginAuthMode
		assert.Equal(t, input, state)
		assert.Equal(t, LoginAuthMode, mode)
	})
}

// FuzzAttachStateOrEmpty fuzzes the AttachStateOrEmpty function
func FuzzAttachStateOrEmpty(f *testing.F) {
	// Seed corpus
	f.Add("client-state", false, "")
	f.Add("client-state", true, "")
	f.Add("", false, "http://localhost:8000")
	f.Add("client-state", false, "http://localhost:8000/callback")
	f.Add("", false, "")
	f.Add("state", false, "http://localhost:12345")
	f.Add("state\nwith\nnewlines", false, "")
	f.Add("state\twith\ttabs", true, "")

	f.Fuzz(func(t *testing.T, clientState string, testMode bool, callbackURL string) {
		result, err := AttachStateOrEmpty(clientState, testMode, callbackURL)

		// 1. If both testMode and callbackURL are set, should return error
		if testMode && callbackURL != "" {
			assert.Error(t, err)
			return
		}

		// Should not error in other cases (unless URL parsing fails)
		// 2. If testMode is true, result should be prefixed with TestLoginClientState
		if testMode {
			assert.NoError(t, err)
			assert.True(t, strings.HasPrefix(result, TestLoginClientState+"#"))
			return
		}

		// 3. If callbackURL is set, result should be prefixed with AuthorizeRoxctlClientState
		//    but only if the URL is valid and localhost
		if callbackURL != "" {
			// May error if URL is invalid or not localhost
			if err == nil {
				assert.True(t, strings.HasPrefix(result, AuthorizeRoxctlClientState+"#"))
			}
			return
		}

		// 4. If neither testMode nor callbackURL, result should equal clientState
		assert.NoError(t, err)
		assert.Equal(t, clientState, result)
	})
}
