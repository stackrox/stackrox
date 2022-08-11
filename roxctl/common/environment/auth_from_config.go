package environment

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/grpc/client/authn/tokenbased"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/auth"
	"github.com/stackrox/rox/roxctl/config"
	"google.golang.org/grpc/credentials"
)

type authFromConfig struct {
	env Environment
}

func (a authFromConfig) Name() string {
	return "config file"
}

func (a authFromConfig) GetCreds(baseURL string) (credentials.PerRPCCredentials, error) {
	token, err := a.getToken(baseURL)
	if err != nil {
		return nil, err
	}
	return tokenbased.PerRPCCredentials(token), nil
}

func (a authFromConfig) getToken(baseURL string) (string, error) {
	cfg, err := config.Load()
	if err != nil {
		return "", errors.Wrap(err, "loading config file")
	}
	hostAccess := cfg.GetHosts()[baseURL].GetAccess()
	if hostAccess == nil || (hostAccess.Token == "" && hostAccess.RefreshToken == "") {
		return "", errors.Errorf("no credentials stored for %s, use 'roxctl login'", baseURL)
	}
	if hostAccess.Token != "" && hostAccess.ExpiresAt.Before(time.Now().Add(-30*time.Second)) {
		return hostAccess.Token, nil
	}

	if hostAccess.RefreshToken == "" {
		a.env.Logger().WarnfLn("Access token for %s is about to expire (or already expired), but no refresh token is available. If the command fails, use 'roxctl login' to obtain a new access token", baseURL)
		return hostAccess.Token, nil
	}

	client, err := a.env.HTTPClient(10*time.Second, auth.Anonymous())
	if err != nil {
		return "", errors.Wrap(err, "could not create HTTP client for token refresh")
	}

	req, err := client.NewReq(http.MethodGet, "/sso/session/tokenrefresh", nil)
	if err != nil {
		return "", errors.Wrap(err, "creating HTTP request for token refresh")
	}
	req.AddCookie(&http.Cookie{
		Name:  "RoxRefreshToken",
		Value: hostAccess.RefreshToken,
	})

	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "performing token refresh request")
	}
	defer utils.IgnoreError(resp.Body.Close)
	if resp.StatusCode != http.StatusOK {
		return "", errors.Errorf("token refresh returned %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.New("reading response body from token refresh")
	}

	var tokenRefresh authproviders.TokenRefreshResponse
	if err := json.Unmarshal(body, &tokenRefresh); err != nil {
		return "", errors.Wrap(err, "could not unmarshal token refresh response")
	}

	a.env.Logger().InfofLn("Refreshed access token for %s, new token expires at %v", baseURL, tokenRefresh.Expiry)

	hostAccess.Token = tokenRefresh.Token
	hostAccess.ExpiresAt = tokenRefresh.Expiry
	now := time.Now()
	hostAccess.IssuedAt = &now

	for _, c := range resp.Cookies() {
		if c.Valid() != nil || c.Name != authproviders.RefreshTokenCookieName {
			continue
		}
		hostAccess.RefreshToken = c.Value
	}

	if err := config.Store(cfg); err != nil {
		return "", fmt.Errorf("failed to store token after token refresh: %w", err)
	}

	return hostAccess.Token, nil
}
