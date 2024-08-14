package login

import (
	"encoding/json"
	sysIO "io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/authproviders/basic"
	"github.com/stackrox/rox/pkg/auth/authproviders/oidc"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/jsonutil"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/auth"
	"github.com/stackrox/rox/roxctl/common/config"
	cfgMock "github.com/stackrox/rox/roxctl/common/config/mocks"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/environment/mocks"
	"github.com/stackrox/rox/roxctl/common/io"
	"github.com/stackrox/rox/roxctl/common/printer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestVerifyLoginAuthProviders_Successful(t *testing.T) {
	server := httptest.NewServer(loginAuthProvidersHandle(t, []*v1.GetLoginAuthProvidersResponse_LoginAuthProvider{
		{
			Id:   "1",
			Name: "basic",
			Type: basic.TypeName,
		},
		{
			Id:   "2",
			Name: "oidc",
			Type: oidc.TypeName,
		},
	}))
	defer server.Close()

	// Required for picking up the endpoint used by GetRoxctlHTTPClient. Currently, it is not possible to inject this
	// otherwise.
	t.Setenv("ROX_ENDPOINT", server.URL)

	loginCmd := loginCommand{
		env: mockEnvWithHTTPClient(t),
	}

	assert.NoError(t, loginCmd.verifyLoginAuthProviders())
}

func TestVerifyLoginAuthProviders_RawResponseData_Successful(t *testing.T) {
	server := httptest.NewServer(loginAuthProvidersRawHandle(t, []map[string]string{
		{
			"id":             "1",
			"name":           "basic",
			"type":           basic.TypeName,
			"dummyTestField": "to test backward/forward compatibility",
		},
		{
			"id":   "2",
			"name": "oidc",
			"type": oidc.TypeName,
		},
	}))
	defer server.Close()

	// Required for picking up the endpoint used by GetRoxctlHTTPClient. Currently, it is not possible to inject this
	// otherwise.
	t.Setenv("ROX_ENDPOINT", server.URL)

	loginCmd := loginCommand{
		env: mockEnvWithHTTPClient(t),
	}

	assert.NoError(t, loginCmd.verifyLoginAuthProviders())
}

func TestVerifyLoginAuthProviders_Failure(t *testing.T) {
	server := httptest.NewServer(loginAuthProvidersHandle(t, []*v1.GetLoginAuthProvidersResponse_LoginAuthProvider{
		{
			Id:   "1",
			Name: "basic",
			Type: basic.TypeName,
		},
	}))
	defer server.Close()

	// Required for picking up the endpoint used by GetRoxctlHTTPClient. Currently, it is not possible to inject this
	// otherwise.
	t.Setenv("ROX_ENDPOINT", server.URL)

	loginCmd := loginCommand{
		env: mockEnvWithHTTPClient(t),
	}

	assert.ErrorIs(t, loginCmd.verifyLoginAuthProviders(), errNoValidLoginAuthProvider)
}

func loginAuthProvidersHandle(t *testing.T, providers []*v1.GetLoginAuthProvidersResponse_LoginAuthProvider) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, http.MethodGet, request.Method)
		assert.Equal(t, "/v1/login/authproviders", request.URL.Path)
		body := request.Body
		defer utils.IgnoreError(body.Close)
		reqBodyData, err := sysIO.ReadAll(body)
		assert.NoError(t, err)
		assert.Len(t, reqBodyData, 0)
		assert.NoError(t, jsonutil.MarshalPretty(writer, &v1.GetLoginAuthProvidersResponse{
			AuthProviders: providers,
		}))
	}
}

func loginAuthProvidersRawHandle(t *testing.T, providersData []map[string]string) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, http.MethodGet, request.Method)
		assert.Equal(t, "/v1/login/authproviders", request.URL.Path)
		body := request.Body
		defer utils.IgnoreError(body.Close)
		reqBodyData, err := sysIO.ReadAll(body)
		assert.NoError(t, err)
		assert.Len(t, reqBodyData, 0)
		respData := map[string][]map[string]string{
			"authProviders": providersData,
		}
		jsonMarshaler := json.NewEncoder(writer)
		assert.NoError(t, jsonMarshaler.Encode(respData))
	}
}

func TestRawHandle(t *testing.T) {
	handler := loginAuthProvidersRawHandle(t, []map[string]string{
		{
			"id":             "1",
			"name":           "basic",
			"type":           basic.TypeName,
			"dummyTestField": "to test backward/forward compatibility",
		},
		{
			"id":   "2",
			"name": "oidc",
			"type": oidc.TypeName,
		},
	})
	expectedWrittenPayload := `{
	"authProviders": [
		{
			"id": "1",
			"name": "basic",
			"type": "basic",
			"dummyTestField": "to test backward/forward compatibility"
		},
		{
			"id": "2",
			"name": "oidc",
			"type": "oidc"
		}
	]
}`
	rspWriter := httptest.NewRecorder()
	fakeRequest := httptest.NewRequest(http.MethodGet, "/v1/login/authproviders", nil)
	handler.ServeHTTP(rspWriter, fakeRequest)
	assert.Equal(t, http.StatusOK, rspWriter.Code)
	assert.JSONEq(t, expectedWrittenPayload, rspWriter.Body.String())
}

func mockEnvWithHTTPClient(t *testing.T) environment.Environment {
	mockEnv := mocks.NewMockEnvironment(gomock.NewController(t))
	testIO, _, _, _ := io.TestIO()
	env := environment.NewTestCLIEnvironment(t, testIO, printer.DefaultColorPrinter())

	mockEnv.EXPECT().InputOutput().AnyTimes().Return(env.InputOutput())
	mockEnv.EXPECT().Logger().AnyTimes().Return(env.Logger())
	mockEnv.EXPECT().GRPCConnection(gomock.Any()).AnyTimes().Return(nil, nil)
	mockEnv.EXPECT().ColorWriter().AnyTimes().Return(env.ColorWriter())
	mockEnv.EXPECT().HTTPClient(gomock.Any(), gomock.Any()).AnyTimes().Return(
		common.GetRoxctlHTTPClient(auth.Anonymous(), 30*time.Second, false, true, env.Logger()))

	return mockEnv
}

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
	mockStore := cfgMock.NewMockStore(gomock.NewController(t))
	mockStore.EXPECT().Read().AnyTimes().Return(&config.RoxctlConfig{CentralConfigs: map[string]*config.CentralConfig{}}, nil)
	mockStore.EXPECT().Write(gomock.Any()).AnyTimes().Return(nil)
	env, _, errOut := mocks.NewEnv(nil, mockStore, t)

	centralURL, err := url.Parse("http://localhost:8080")
	require.NoError(t, err)
	loginCmd := loginCommand{
		env:         env,
		loginSignal: concurrency.NewErrorSignal(),
		centralURL:  centralURL,
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
INFO:	Successfully persisted the authentication information for central localhost:8080.

You can now use the retrieved access token for all other roxctl commands!

In case the access token is expired and cannot be refreshed, you have to run "roxctl central login" again.
`
	assert.Equal(t, expectedOutput, errOut.String())
}
