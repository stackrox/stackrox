package environment

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/auth"
	"github.com/stackrox/rox/roxctl/common/config"
	configMock "github.com/stackrox/rox/roxctl/common/config/mocks"
	roxctlIO "github.com/stackrox/rox/roxctl/common/io"
	"github.com/stackrox/rox/roxctl/common/logger"
	"github.com/stackrox/rox/roxctl/common/printer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestRetrieveToken_Empty(t *testing.T) {
	cfgStore := configMock.NewMockStore(gomock.NewController(t))
	env, _ := newMockEnvironment(t, nil, cfgStore, false)
	cm := &configMethod{env: env}

	// 1. Completely empty central config.
	cfgStore.EXPECT().Read().Return(nil, nil)

	token, err := cm.retrieveToken("some-url")
	assert.Empty(t, token)
	assert.ErrorIs(t, err, errox.NoCredentials)

	// 2. Empty access token and refresh token
	cfgStore.EXPECT().Read().Return(
		&config.RoxctlConfig{
			CentralConfigs: map[config.CentralURL]*config.CentralConfig{
				"some-url": {AccessConfig: &config.CentralAccessConfig{}},
			},
		}, nil,
	)
	token, err = cm.retrieveToken("some-url")
	assert.Empty(t, token)
	assert.ErrorIs(t, err, errox.NoCredentials)
}

func TestRetrieveToken_NonExpiredAccessToken(t *testing.T) {
	cfgStore := configMock.NewMockStore(gomock.NewController(t))
	env, _ := newMockEnvironment(t, nil, cfgStore, false)
	cm := &configMethod{env: env}

	tomorrow := time.Now().UTC().Add(24 * time.Hour)
	cfgStore.EXPECT().Read().Return(
		&config.RoxctlConfig{
			CentralConfigs: map[config.CentralURL]*config.CentralConfig{
				"some-url": {AccessConfig: &config.CentralAccessConfig{
					AccessToken: "some-token",
					ExpiresAt:   &tomorrow,
				}},
			},
		}, nil,
	)

	token, err := cm.retrieveToken("some-url")
	assert.NoError(t, err)
	assert.Equal(t, "some-token", token)
}

func TestRetrieveToken_EmptyRefreshToken(t *testing.T) {
	cfgStore := configMock.NewMockStore(gomock.NewController(t))
	env, errOut := newMockEnvironment(t, nil, cfgStore, false)
	cm := &configMethod{env: env}

	oneHourAgo := time.Now().UTC().Add(-time.Hour)
	cfgStore.EXPECT().Read().Return(
		&config.RoxctlConfig{
			CentralConfigs: map[config.CentralURL]*config.CentralConfig{
				"some-url": {AccessConfig: &config.CentralAccessConfig{
					AccessToken: "some-token",
					ExpiresAt:   &oneHourAgo,
				}},
			},
		}, nil,
	)

	token, err := cm.retrieveToken("some-url")
	assert.NoError(t, err)
	assert.Equal(t, "some-token", token)
	assert.Equal(t,
		`WARN:	No indication about expiration found or already expired for access token for central some-url.
Still, trying to authenticate with the given token. In case there's any issues, run "roxctl central login" again.
`,
		errOut.String())
}

var (
	errClient = errors.New("client error")
	errNewReq = errors.New("failed creating request")
	errReq    = errors.New("failed executing request")
)

func TestRetrieveToken_RefreshAccessToken(t *testing.T) {
	cfg := &config.CentralAccessConfig{
		RefreshToken: "some-refresh-token",
	}
	initialCfg := *cfg

	// 1. Fail on obtaining the HTTP client.
	env, _ := newMockEnvironment(t, nil, nil, true)
	cm := &configMethod{env: env}
	err := cm.refreshAccessToken("some-url", cfg)
	assert.ErrorIs(t, err, errClient)
	assert.Equal(t, &initialCfg, cfg)

	// 2. Fail on creating the HTTP request.
	client := &mockRoxctlHTTPClient{t: t, failNew: true}
	env, _ = newMockEnvironment(t, client, nil, false)
	cm = &configMethod{env: env}
	err = cm.refreshAccessToken("some-url", cfg)
	assert.ErrorIs(t, err, errNewReq)
	assert.Equal(t, &initialCfg, cfg)

	// 3. Fail executing the HTTP request.
	client = &mockRoxctlHTTPClient{t: t, failRequest: true}
	env, _ = newMockEnvironment(t, client, nil, false)
	cm = &configMethod{env: env}
	err = cm.refreshAccessToken("some-url", cfg)
	assert.ErrorIs(t, err, errReq)
	assert.Equal(t, &initialCfg, cfg)

	// 4. Succeed in token refresh request.
	expiry := time.Now().UTC()
	client = &mockRoxctlHTTPClient{
		t:                    t,
		expectedRefreshToken: "some-refresh-token",
		response: authproviders.TokenRefreshResponse{
			Token:  "some-token",
			Expiry: expiry,
		},
	}
	env, _ = newMockEnvironment(t, client, nil, false)
	cm = &configMethod{env: env}
	err = cm.refreshAccessToken("some-url", cfg)
	assert.NoError(t, err)

	assert.Equal(t, "some-token", cfg.AccessToken)
	assert.NotNil(t, cfg.IssuedAt)
	assert.Equal(t, expiry, *cfg.ExpiresAt)
	assert.Equal(t, "some-refresh-token", cfg.RefreshToken)
}

type mockRoxctlHTTPClient struct {
	t           *testing.T
	failNew     bool
	failRequest bool

	expectedRefreshToken string
	response             authproviders.TokenRefreshResponse
	common.RoxctlHTTPClient
}

func (m *mockRoxctlHTTPClient) NewReq(_, _ string, _ io.Reader) (*http.Request, error) {
	if m.failNew {
		return nil, errNewReq
	}
	return http.NewRequest("", "", nil)
}

func (m *mockRoxctlHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if m.failRequest {
		return nil, errReq
	}

	cookie, err := req.Cookie(authproviders.RefreshTokenCookieName)
	assert.NoError(m.t, err)
	assert.Equal(m.t, m.expectedRefreshToken, cookie.Value)

	raw, err := json.Marshal(&m.response)
	require.NoError(m.t, err)
	buf := bytes.NewBuffer(raw)
	return &http.Response{Body: io.NopCloser(buf), StatusCode: http.StatusOK}, nil
}

func newMockEnvironment(t *testing.T, client common.RoxctlHTTPClient, store config.Store, fail bool) (Environment, *bytes.Buffer) {
	testIO, _, _, errOut := roxctlIO.TestIO()
	env := NewTestCLIEnvironment(t, testIO, printer.DefaultColorPrinter())
	return &mockEnvironment{
		client: client,
		logger: env.Logger(),
		store:  store,
		fail:   fail,
	}, errOut
}

type mockEnvironment struct {
	client common.RoxctlHTTPClient
	store  config.Store
	logger logger.Logger
	fail   bool
	Environment
}

func (m *mockEnvironment) HTTPClient(_ time.Duration, _ ...auth.Method) (common.RoxctlHTTPClient, error) {
	if m.fail {
		return nil, errClient
	}
	return m.client, nil
}

func (m *mockEnvironment) Logger() logger.Logger {
	return m.logger
}

func (m *mockEnvironment) ConfigStore() (config.Store, error) {
	return m.store, nil
}
