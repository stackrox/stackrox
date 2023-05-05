package environment

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/client/authn/tokenbased"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/auth"
	"github.com/stackrox/rox/roxctl/common/config"
	"google.golang.org/grpc/credentials"
)

const (
	refreshTokenPath = "/sso/session/tokenrefresh"
)

type configMethod struct {
	env Environment
}

var (
	_ auth.Method = (*configMethod)(nil)
)

// ConfigMethod provides an auth.Method for using authentication via local configuration.
// It will use the configuration store by roxctl central login.
func ConfigMethod(env Environment) auth.Method {
	return &configMethod{
		env: env,
	}
}

func (c configMethod) Type() string {
	return "local configuration"
}

func (c configMethod) GetCredentials(url string) (credentials.PerRPCCredentials, error) {
	token, err := c.retrieveToken(url)
	if err != nil {
		return nil, errors.Wrap(err, "retrieving token")
	}
	return tokenbased.PerRPCCredentials(token), nil
}

func (c configMethod) retrieveToken(url string) (string, error) {
	cfgStore, err := c.env.ConfigStore()
	if err != nil {
		return "", errors.Wrap(err, "retrieving config store")
	}
	cfg, err := cfgStore.Read()
	if err != nil {
		return "", errors.Wrap(err, "reading configuration")
	}

	access := cfg.GetCentralConfigs().GetCentralConfig(url).GetAccess()
	if access == nil || (access.AccessToken == "" && access.RefreshToken == "") {
		return "", errox.NoCredentials.Newf(
			`no credentials found for %s, please run "roxctl central login" to obtain credentials`, url)
	}

	// 1. Check if an access token is given. If an access token is given, and it's not yet expired, then return it.
	if access.AccessToken != "" && access.ExpiresAt != nil && access.ExpiresAt.After(time.Now().UTC().Add(30*time.Second)) {
		return access.AccessToken, nil
	}

	// 2. In case an access token is given, but either no expiration is indicated or the token is already expired, and
	// no refresh token is available, print a warning that the authentication _may_ fail and users have to re-run
	// roxctl central login in case of authentication issues.
	if access.AccessToken != "" && access.RefreshToken == "" {
		c.env.Logger().WarnfLn(`No indication about expiration found or already expired for access token for central %s.
Still, trying to authenticate with the given token. In case there's any issues, run "roxctl central login" again.`, url)
		return access.AccessToken, nil
	}

	// 3. This is the case where an access token is either not set or expired, and a refresh token is available.
	// We attempt to refresh the access token here.
	if err := c.refreshAccessToken(url, access); err != nil {
		c.env.Logger().WarnfLn("An error occurred during access token refresh. Try running roxctl central login again.")
		return "", errors.Wrap(err, "refreshing access token")
	}

	// 4. Write the config with the updated data to the store.
	if err := cfgStore.Write(cfg); err != nil {
		return "", errors.Wrap(err, "writing configuration to store after refreshing the token")
	}

	return access.AccessToken, nil
}

func (c configMethod) refreshAccessToken(url string, accessConfig *config.CentralAccessConfig) error {
	client, err := c.env.HTTPClient(time.Minute, auth.Anonymous())
	if err != nil {
		return errors.Wrap(err, "obtaining client for token refresh")
	}
	req, err := client.NewReq(http.MethodGet, refreshTokenPath, nil)
	if err != nil {
		return errors.Wrap(err, "creating request for token refresh")
	}
	req.AddCookie(&http.Cookie{
		Name:  authproviders.RefreshTokenCookieName,
		Value: accessConfig.RefreshToken,
	})

	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrap(err, "executing token refresh request")
	}
	defer utils.IgnoreError(resp.Body.Close)

	if !httputil.Is2xxStatusCode(resp.StatusCode) {
		body, err := io.ReadAll(resp.Body)
		utils.Should(err)
		return errors.Errorf("unexpected status code from token refresh request %d: %s", resp.StatusCode, body)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "reading response from token refresh request")
	}

	var tokenRefreshResponse authproviders.TokenRefreshResponse
	if err := json.Unmarshal(body, &tokenRefreshResponse); err != nil {
		return errors.Wrap(err, "unmarshalling token refresh request response")
	}

	c.env.Logger().InfofLn("Successfully refreshed access token for central %s, the new token will expire at %s", url,
		tokenRefreshResponse.Expiry)
	accessConfig.AccessToken = tokenRefreshResponse.Token
	accessConfig.ExpiresAt = &tokenRefreshResponse.Expiry
	now := time.Now()
	accessConfig.IssuedAt = &now
	// Although a cookie will be returned, we will _not_ be able to read from it, since the cookie has the
	// HTTPOnly flag set. Hence, ignore the refresh token, and attempt refreshing until the refresh token is invalid -
	// then users will have to run roxctl central login again.
	return nil
}
