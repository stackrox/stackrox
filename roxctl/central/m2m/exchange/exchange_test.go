package exchange

import (
	"bytes"
	"encoding/json"
	sysIO "io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
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

const (
	expectedToken1 = "some-token"
	responseToken1 = "test-token"

	expectedToken2 = "some-other-token"
	responseToken2 = "other-test-token"
)

func mockEnvWithHTTPClient(t *testing.T, store *cfgMock.MockStore) environment.Environment {
	mockEnv := mocks.NewMockEnvironment(gomock.NewController(t))
	testIO, _, _, _ := io.TestIO()
	env := environment.NewTestCLIEnvironment(t, testIO, printer.DefaultColorPrinter())

	mockEnv.EXPECT().InputOutput().AnyTimes().Return(env.InputOutput())
	mockEnv.EXPECT().Logger().AnyTimes().Return(env.Logger())
	mockEnv.EXPECT().GRPCConnection(gomock.Any()).AnyTimes().Return(nil, nil)
	mockEnv.EXPECT().ColorWriter().AnyTimes().Return(env.ColorWriter())
	mockEnv.EXPECT().ConfigStore().AnyTimes().Return(store, nil)
	mockEnv.EXPECT().HTTPClient(gomock.Any(), gomock.Any()).AnyTimes().Return(
		common.GetRoxctlHTTPClient(auth.Anonymous(), 30*time.Second, false, true, env.Logger()))

	return mockEnv
}

func TestExchange_From_TokenFlag(t *testing.T) {
	env, closeFn := setupTest(t, expectedToken1, responseToken1, exchangeHandle)
	defer closeFn()

	exchangeCmd := Command(env)
	exchangeCmd.SetArgs([]string{"--token", expectedToken1})
	assert.NoError(t, exchangeCmd.Execute())
}

func TestExchange_From_TokenFile(t *testing.T) {
	env, closeFn := setupTest(t, expectedToken1, responseToken1, exchangeHandle)
	defer closeFn()

	tokenFilePath := path.Join(t.TempDir(), "token-file")

	require.NoError(t, os.WriteFile(tokenFilePath, []byte(expectedToken1), 0644))

	exchangeCmd := Command(env)
	exchangeCmd.SetArgs([]string{"--token-file", tokenFilePath})
	assert.NoError(t, exchangeCmd.Execute())
}

func TestExchange_Raw_From_TokenFlag(t *testing.T) {
	env, closeFn := setupTest(t, expectedToken2, responseToken2, exchangeRawHandle)
	defer closeFn()

	exchangeCmd := Command(env)
	exchangeCmd.SetArgs([]string{"--token", expectedToken2})
	assert.NoError(t, exchangeCmd.Execute())
}

func TestExchange_Raw_From_TokenFile(t *testing.T) {
	env, closeFn := setupTest(t, expectedToken2, responseToken2, exchangeRawHandle)
	defer closeFn()

	tokenFilePath := path.Join(t.TempDir(), "token-file")

	require.NoError(t, os.WriteFile(tokenFilePath, []byte(expectedToken2), 0644))

	exchangeCmd := Command(env)
	exchangeCmd.SetArgs([]string{"--token-file", tokenFilePath})
	assert.NoError(t, exchangeCmd.Execute())
}

func setupTest(
	t *testing.T,
	expectedToken string,
	responseToken string,
	handlerFactory func(*testing.T, string, string) http.HandlerFunc,
) (environment.Environment, func()) {
	server := httptest.NewServer(handlerFactory(t, expectedToken, responseToken))
	// Required for picking up the endpoint used by GetRoxctlHTTPClient. Currently, it is not possible to inject this
	// otherwise.
	t.Setenv("ROX_ENDPOINT", server.URL)

	serverURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	mockStore := cfgMock.NewMockStore(gomock.NewController(t))
	emptyCfg := &config.RoxctlConfig{CentralConfigs: map[string]*config.CentralConfig{}}
	mockStore.EXPECT().Read().AnyTimes().Return(emptyCfg, nil)
	cfgKey := config.NewConfigKey(serverURL)
	mockStore.EXPECT().Write(matchesConfig(&config.RoxctlConfig{CentralConfigs: map[config.CentralURL]*config.CentralConfig{
		cfgKey: {AccessConfig: &config.CentralAccessConfig{
			AccessToken: responseToken,
		}},
	}}, cfgKey)).AnyTimes().Return(nil)

	return mockEnvWithHTTPClient(t, mockStore), func() {
		server.Close()
	}
}

type centralCfgMatcher struct {
	roxctlConfig *config.RoxctlConfig
	configKey    string
}

func (c centralCfgMatcher) String() string {
	return "config matcher"
}

func (c centralCfgMatcher) Matches(arg interface{}) bool {
	cfgArg := arg.(*config.RoxctlConfig)
	if cfgArg == nil {
		return false
	}

	centralCfg, exists := cfgArg.CentralConfigs[c.configKey]
	if !exists {
		return false
	}

	return c.roxctlConfig.CentralConfigs[c.configKey].AccessConfig.AccessToken == centralCfg.AccessConfig.AccessToken &&
		centralCfg.AccessConfig.IssuedAt != nil &&
		centralCfg.AccessConfig.ExpiresAt == nil &&
		c.roxctlConfig.CentralConfigs[c.configKey].AccessConfig.RefreshToken == centralCfg.AccessConfig.RefreshToken
}

func matchesConfig(roxctlConfig *config.RoxctlConfig, key string) gomock.Matcher {
	return centralCfgMatcher{configKey: key, roxctlConfig: roxctlConfig}
}

func exchangeHandle(t *testing.T, expectedToken, responseToken string) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		var m2mRequest v1.ExchangeAuthMachineToMachineTokenRequest
		assert.NoError(t, jsonutil.JSONReaderToProto(request.Body, &m2mRequest))
		assert.Equal(t, expectedToken, m2mRequest.GetIdToken())

		assert.NoError(t, jsonutil.MarshalPretty(writer, &v1.ExchangeAuthMachineToMachineTokenResponse{
			AccessToken: responseToken,
		}))
	}
}

func exchangeRawHandle(t *testing.T, expectedToken, responseToken string) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		reqBody := request.Body
		defer utils.IgnoreError(reqBody.Close)
		reqBodyData, err := sysIO.ReadAll(reqBody)
		assert.NoError(t, err)
		assert.JSONEq(t, `{"idToken":"`+expectedToken+`"}`, string(reqBodyData))
		assert.Equal(t, http.MethodPost, request.Method)
		assert.Equal(t, "/v1/auth/m2m/exchange", request.URL.Path)

		resp := map[string]string{
			"accessToken":    responseToken,
			"dummyTestField": "to test backward/forward compatibility",
		}
		encoder := json.NewEncoder(writer)
		err = encoder.Encode(resp)
		assert.NoError(t, err)
	}
}

func TestExchangeRawHandle(t *testing.T) {
	requestPayload := `{"idToken":"` + expectedToken2 + `"}`
	requestPayloadBuffer := bytes.NewBufferString(requestPayload)
	testReq := httptest.NewRequest(http.MethodPost, "/v1/auth/m2m/exchange", requestPayloadBuffer)
	responseWriter := httptest.NewRecorder()
	handler := exchangeRawHandle(t, expectedToken2, responseToken2)
	handler.ServeHTTP(responseWriter, testReq)
	expectedResponseBody := `{
	"accessToken": "` + responseToken2 + `",
	"dummyTestField": "to test backward/forward compatibility"
}`
	assert.Equal(t, http.StatusOK, responseWriter.Code)
	assert.JSONEq(t, expectedResponseBody, responseWriter.Body.String())
}
