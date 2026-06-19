//go:build test && !test_e2e && !test_e2e_vm

package vmhelpers

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"
)

var text = []byte(`Here is some text.
Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore
magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo
consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur.
Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.`)

func TestLogMatcher(t *testing.T) {
	tests := map[string]struct {
		funcs          []LogMatcher
		expectedResult assert.BoolAssertionFunc
		expectedError  assert.ErrorAssertionFunc
		expectedString string
	}{
		"one match": {
			funcs: []LogMatcher{
				ContainsLineMatching(regexp.MustCompile("sunt in culpa qui officia deserunt")),
			},
			expectedResult: assert.True,
			expectedError:  assert.NoError,
			expectedString: `[contains line(s) matching "sunt in culpa qui officia deserunt"]`,
		},
		"two matches": {
			funcs: []LogMatcher{
				ContainsLineMatching(regexp.MustCompile("Lorem ipsum dolor")),
				ContainsLineMatching(regexp.MustCompile("Duis aute irure")),
			},
			expectedResult: assert.True,
			expectedError:  assert.NoError,
			expectedString: `[contains line(s) matching "Lorem ipsum dolor" contains line(s) matching "Duis aute irure"]`,
		},
		"text divided with newline": {
			funcs: []LogMatcher{
				ContainsLineMatching(regexp.MustCompile("labore et dolore.*magna aliqua")),
			},
			expectedResult: assert.False,
			expectedError:  assert.NoError,
			expectedString: `[contains line(s) matching "labore et dolore.*magna aliqua"]`,
		},
	}
	r := bytes.NewReader(text)
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			actual, actualErr := AllMatch(r, test.funcs...)
			test.expectedResult(t, actual)
			test.expectedError(t, actualErr)
			assert.Equal(t, test.expectedString, fmt.Sprintf("%s", test.funcs))
		})
	}
}

func TestCreateK8sClientWithConfig_RetriesOnFailure(t *testing.T) {
	var callCount int

	mockTransport := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		callCount++
		currentCall := callCount

		t.Logf("Mock transport call #%d to %s", currentCall, r.URL.String())

		if currentCall <= 2 {
			return nil, errors.New("network error: connection refused")
		}

		responseBody := `{
			"major": "1",
			"minor": "28",
			"gitVersion": "v1.28.0",
			"gitCommit": "abcd1234",
			"buildDate": "2023-08-01T12:00:00Z",
			"goVersion": "go1.20.6",
			"compiler": "gc",
			"platform": "linux/amd64"
		}`

		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(strings.NewReader(responseBody)),
			Header:     make(http.Header),
		}, nil
	})

	restCfg := &rest.Config{
		Host: "https://mock-k8s-api.example.com",
		WrapTransport: func(http.RoundTripper) http.RoundTripper {
			return mockTransport
		},
	}
	ConfigureRetryableTransport(t, restCfg)

	client := CreateK8sClientWithConfig(t, restCfg)
	require.NotNil(t, client, "client should not be nil")

	version, err := client.Discovery().ServerVersion()

	require.NoError(t, err, "Discovery call should succeed after retries")
	require.NotNil(t, version, "Server version should not be nil")
	assert.Equal(t, "1", version.Major)
	assert.Equal(t, "28", version.Minor)

	assert.Equal(t, 3, callCount, "Should have made exactly 3 calls (2 retries + 1 success)")
	t.Logf("Successfully completed after %d calls (including retries)", callCount)
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}
