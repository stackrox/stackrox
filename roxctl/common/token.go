package common

import (
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/roxctl/common/flags"
)

const userHelpLiteralToken = `There is no token in file %q. The token file should only contain a single authentication token.
To provide a token value directly, set the ROX_API_TOKEN environment variable.
`

// RetrieveAuthToken retrieves an authentication token. Token files specified on the command line have precedence over tokens
// configured in the environment.
// Returns an empty token if neither a token file is specified nor a token is configured in the environment.
// May print to stderr.
func RetrieveAuthToken() (string, error) {
	var apiToken string

	// Try to retrieve API token. First via --token-file parameter and then from the environment.
	if tokenFile := flags.APITokenFile(); tokenFile != "" {
		// Error out if --token-file and --password is present on the command line.
		if flags.Password() != "" {
			return "", errors.New("cannot use password- and token-based authentication at the same time")
		}

		token, err := flags.ReadTokenFromFile(tokenFile)
		if err != nil {
			if !strings.Contains(tokenFile, "/") {
				// Specified token file looks somewhat like a literal token, try to help the user.
				fmt.Fprintf(os.Stderr, userHelpLiteralToken, tokenFile)
			}
			return "", err
		}
		apiToken = token
	} else if token := env.TokenEnv.Setting(); token != "" {
		apiToken = token
	}

	return strings.TrimSpace(apiToken), nil
}
