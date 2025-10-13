package exchange

import (
	"bytes"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/jsonutil"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/auth"
	"github.com/stackrox/rox/roxctl/common/config"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
)

// Command to exchange an OIDC token for a short-lived access token.
func Command(cliEnvironment environment.Environment) *cobra.Command {
	exchangeCmd := exchangeCommand{env: cliEnvironment}

	cmd := &cobra.Command{
		Use:   "exchange",
		Short: "Exchanges an OIDC token for a short-lived access token",
		Long: `Obtain a short-lived access token from Central by exchanging an OIDC token.
This works by configuring a machine-to-machine access configuration within Central beforehand.
Based on the OIDC token's issuer, a short-lived access token will be exchanged.

The access token will be stored in the roxctl configuration file and used for authentication in other commands.`,
		RunE: util.RunENoArgs(func(command *cobra.Command) error {
			if err := exchangeCmd.construct(command); err != nil {
				return err
			}
			return exchangeCmd.exchange()
		}),
	}
	cmd.Flags().StringVar(&exchangeCmd.token, "token", "",
		"OIDC identity token to exchange for a short-lived access token.")
	cmd.Flags().StringVar(&exchangeCmd.tokenFile, "token-file", "",
		"File containing an OIDC identity token to exchange for a short-lived access token.")
	cmd.MarkFlagsOneRequired("token", "token-file")
	cmd.MarkFlagsMutuallyExclusive("token", "token-file")
	flags.AddTimeoutWithDefault(cmd, 1*time.Minute)
	return cmd
}

type exchangeCommand struct {
	env        environment.Environment
	timeout    time.Duration
	centralURL *url.URL
	token      string
	tokenFile  string
}

func (e *exchangeCommand) construct(cmd *cobra.Command) error {
	e.timeout = flags.Timeout(cmd)
	centralURL, err := flags.CentralURL()
	if err != nil {
		return errors.Wrap(err, "retrieving Central URL")
	}
	e.centralURL = centralURL

	if e.tokenFile != "" {
		fileContents, err := os.ReadFile(e.tokenFile)
		if err != nil {
			return errors.Wrapf(err, "reading token from file %q", e.tokenFile)
		}
		token := strings.TrimSpace(string(fileContents))
		if token == "" {
			return errox.InvalidArgs.Newf("empty token given from file %q", e.tokenFile)
		}
		e.token = token
	}
	return nil
}

func (e *exchangeCommand) exchange() error {
	// The exchange API is anonymous, no auth is required.
	httpClient, err := e.env.HTTPClient(e.timeout, common.WithAuthMethod(auth.Anonymous()))
	if err != nil {
		return errors.Wrap(err, "creating HTTP client")
	}

	req := &v1.ExchangeAuthMachineToMachineTokenRequest{
		IdToken: e.token,
	}
	buf := &bytes.Buffer{}
	if err := jsonutil.Marshal(buf, req); err != nil {
		return errors.Wrap(err, "creating request body")
	}

	// Exchange the OIDC token for a short-lived access token.

	resp, err := httpClient.DoReqAndVerifyStatusCode("/v1/auth/m2m/exchange", http.MethodPost,
		http.StatusOK, buf)
	if err != nil {
		return errors.Wrap(err, "exchange request failed")
	}
	var exchangeResp v1.ExchangeAuthMachineToMachineTokenResponse
	if err := jsonutil.JSONReaderToProto(resp.Body, &exchangeResp); err != nil {
		return errors.Wrap(err, "unmarshalling exchange request response")
	}

	// Store the OIDC token locally to allow other commands to make use of it.

	cfgStore, err := e.env.ConfigStore()
	if err != nil {
		return errors.Wrap(err, "retrieving config store")
	}
	cfg, err := cfgStore.Read()
	if err != nil {
		return errors.Wrap(err, "reading configuration")
	}
	configKey := config.NewConfigKey(e.centralURL)

	existingCfg := cfg.GetCentralConfigs().GetCentralConfig(configKey)
	now := time.Now()
	existingCfg.AccessConfig = &config.CentralAccessConfig{
		AccessToken:  exchangeResp.GetAccessToken(),
		IssuedAt:     &now,
		ExpiresAt:    nil,
		RefreshToken: "",
	}

	if err := cfgStore.Write(cfg); err != nil {
		return errors.Wrap(err, "writing configuration")
	}

	e.env.Logger().InfofLn(`Successfully persisted the authentication information for central %s.

You can now use the exchanged short-lived access token for all other commands!

Note that in case the token is expired, you have to run "roxctl central machine-to-machine exchange" again.`, configKey)
	return nil
}
