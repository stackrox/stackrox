package auth

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/client/authn/tokenbased"
	"github.com/stackrox/rox/roxctl/common/flags"
	"google.golang.org/grpc/credentials"
)

var (
	_ Method = (*tokenMethod)(nil)
)

// TokenAuth provides an auth.Method for token-based authentication.
// It will use the inputs from the --token-file flag or the ROX_API_TOKEN environment variable.
func TokenAuth() Method {
	return &tokenMethod{}
}

type tokenMethod struct {
}

func (t tokenMethod) Type() string {
	return "token based auth"
}

func (t tokenMethod) GetCredentials(_ string) (credentials.PerRPCCredentials, error) {
	token, err := t.retrieveAuthToken()
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve API token")
	}
	return tokenbased.PerRPCCredentials(token), nil
}

func (t tokenMethod) retrieveAuthToken() (string, error) {
	var apiToken string
	// Try to retrieve API token. First via --token-file parameter and then from the environment.
	if tokenFile := flags.APITokenFile(); tokenFile != "" {
		token, err := flags.ReadTokenFromFile(tokenFile)
		if err != nil {
			return "", errors.Wrapf(err, "could not read token from %q", tokenFile)
		}
		apiToken = token
	} else if token := env.TokenEnv.Setting(); token != "" {
		apiToken = token
	}

	if apiToken == "" {
		return "", errox.InvalidArgs.New(`No valid token is set.
Set the token file via the --token-file flag, and ensure only a single authentication token is contained within it.
Alternatively, provide the value directly by setting the ROX_API_TOKEN environment variable.
`)
	}

	return strings.TrimSpace(apiToken), nil
}
