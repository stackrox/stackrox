package common

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn/basic"
	"github.com/stackrox/rox/roxctl/common/flags"
	. "github.com/stackrox/rox/roxctl/common/logger"
)

// Auth provides an abstraction to inject authentication information within http.Request
type Auth interface {
	SetAuth(req *http.Request) error
}

func checkAuthParameters() error {
	if flags.APITokenFile() != "" && flags.Password() != "" {
		return errox.InvalidArgs.New("cannot use password- and token-based authentication at the same time")
	}
	if flags.APITokenFile() == "" && env.TokenEnv.Setting() == "" && flags.Password() == "" {
		return errox.InvalidArgs.New("no token set via either token file or the environment variable ROX_API_TOKEN")
	}

	return nil
}

const userHelpLiteralToken = `There is no token in file %q. The token file should only contain a single authentication token.
To provide a token value directly, set the ROX_API_TOKEN environment variable.
`

func printAuthHelp(logger Logger) {
	if !strings.Contains(flags.APITokenFile(), "/") {
		// Specified token file looks somewhat like a literal token, try to help the user.
		logger.PrintfLn(userHelpLiteralToken, flags.APITokenFile())
	}
}

// newAuth creates a new Auth type which will be inferred based off of the values of flags.APITokenFile and flags.Password.
func newAuth(logger Logger) (Auth, error) {
	if err := checkAuthParameters(); err != nil {
		return nil, err
	}

	if flags.Password() != "" {
		return &basicAuthenticator{pw: flags.Password()}, nil
	}

	token, err := retrieveAuthToken()
	if err != nil {
		printAuthHelp(logger)
		return nil, err
	}
	return &apiTokenAuthenticator{token}, nil
}

type basicAuthenticator struct {
	pw string
}

// SetAuth sets required headers for basic authentication on the given http.Request
func (b *basicAuthenticator) SetAuth(req *http.Request) error {
	req.SetBasicAuth(basic.DefaultUsername, b.pw)
	return nil
}

type apiTokenAuthenticator struct {
	token string
}

// SetAuth sets the required authorization header with a token in bearer format on the given http.Request
func (a *apiTokenAuthenticator) SetAuth(req *http.Request) error {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.token))
	return nil
}
