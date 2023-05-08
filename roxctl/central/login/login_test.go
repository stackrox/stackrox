package login

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/roxctl/common/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoginHandle(t *testing.T) {
	env, _, _ := mocks.NewEnvWithConn(nil, t)

	centralURL, err := url.Parse("http://localhost:8080")
	require.NoError(t, err)

	callbackURL := "http://localhost:8080/callback"

	loginCmd := loginCommand{
		env:        env,
		centralURL: centralURL,
	}

	server := httptest.NewServer(loginCmd.loginHandle(callbackURL))
	defer server.Close()

	// Create a http client which does not follow redirects automatically, since the login handle func will redirect
	// to central.
	client := http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Get(server.URL)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, http.StatusSeeOther, resp.StatusCode)
	redirectURL := resp.Header.Get("Location")
	assert.NotEmpty(t, redirectURL)

	parsedRedirectURL, err := url.Parse(redirectURL)
	assert.NoError(t, err)
	assert.Equal(t, centralURL.Hostname(), parsedRedirectURL.Hostname())
	assert.Equal(t, centralURL.Port(), parsedRedirectURL.Port())
	assert.Equal(t, "/authorize-roxctl", parsedRedirectURL.Path)
	qp, err := url.ParseQuery(parsedRedirectURL.Fragment)
	assert.NoError(t, err)

	assert.Equal(t, callbackURL, qp.Get(authproviders.AuthorizeCallbackQueryParameter))
}

func TestCallbackHandle_Failures(t *testing.T) {
	cases := map[string]struct {
		query string
		err   error
	}{
		"error set should lead to failure": {
			query: "?error=some-error-happened",
		},
		"no token query parameter set should lead to failure": {
			err: errox.InvalidArgs,
		},
		"empty token query parameter should lead to failure": {
			query: "?token=",
			err:   errox.InvalidArgs,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			env, _, _ := mocks.NewEnvWithConn(nil, t)
			loginCmd := loginCommand{
				env:         env,
				loginSignal: concurrency.NewErrorSignal(),
			}

			server := httptest.NewServer(http.HandlerFunc(loginCmd.callbackHandle))
			defer server.Close()

			serverURL, err := url.Parse(server.URL + c.query)
			require.NoError(t, err)

			_, _ = http.DefaultClient.Get(serverURL.String())
			err = loginCmd.loginSignal.Err()
			assert.Error(t, err)
			if c.err != nil {
				assert.ErrorIs(t, err, c.err)
			}
		})
	}
}

func TestCallbackHandle_Success(t *testing.T) {
	env, _, errOut := mocks.NewEnvWithConn(nil, t)
	loginCmd := loginCommand{
		env:         env,
		loginSignal: concurrency.NewErrorSignal(),
	}

	server := httptest.NewServer(http.HandlerFunc(loginCmd.callbackHandle))
	defer server.Close()

	serverURL, err := url.Parse(server.URL + "?token=my-token&refreshToken=my-refresh-token")
	require.NoError(t, err)

	resp, err := http.DefaultClient.Get(serverURL.String())
	assert.NotNil(t, resp)
	assert.NoError(t, err)

	assert.True(t, loginCmd.loginSignal.IsDone())

	expectedOutput := `INFO:	Received the following after the authorization flow from Central:
INFO:	Access token: my-token
INFO:	Refresh token: my-refresh-token
INFO:	Storing these values under $HOME/.roxctl/login ...
`
	assert.Equal(t, expectedOutput, errOut.String())
}
