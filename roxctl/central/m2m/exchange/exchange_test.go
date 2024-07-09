package exchange

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"testing"
	"time"

	"github.com/golang/protobuf/jsonpb"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/jsonutil"
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
	env, closeFn := setup(t)
	defer closeFn()

	exchangeCmd := Command(env)
	exchangeCmd.SetArgs([]string{"--token", "some-token"})
	assert.NoError(t, exchangeCmd.Execute())
}

func TestExchange_From_TokenFile(t *testing.T) {
	env, closeFn := setup(t)
	defer closeFn()

	tokenFilePath := path.Join(t.TempDir(), "token-file")

	require.NoError(t, os.WriteFile(tokenFilePath, []byte("some-token"), 0644))

	exchangeCmd := Command(env)
	exchangeCmd.SetArgs([]string{"--token-file", tokenFilePath})
	assert.NoError(t, exchangeCmd.Execute())
}

func setup(t *testing.T) (environment.Environment, func()) {
	server := httptest.NewServer(exchangeHandle(t, "some-token", "test-token"))
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
			AccessToken: "test-token",
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

		m := jsonpb.Marshaler{Indent: "  "}
		assert.NoError(t, m.Marshal(writer, &v1.ExchangeAuthMachineToMachineTokenResponse{
			AccessToken: responseToken,
		}))
	}
}
