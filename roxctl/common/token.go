package common

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/roxctl/common/flags"
)

// retrieveAuthToken retrieves an authentication token. Token files specified on the command line have precedence over tokens
// configured in the environment.
// Returns an empty token if neither a token file is specified nor a token is configured in the environment.
func retrieveAuthToken() (string, error) {
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

	return strings.TrimSpace(apiToken), nil
}
