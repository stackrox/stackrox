package common

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/authn/basic"
	"github.com/stackrox/rox/roxctl/common/flags"
)

// Auth provides an abstraction to inject authentication information within http.Request
type Auth interface {
	SetAuth(req *http.Request) error
}

// NewAuth create a new Auth type which will be inferred based off of the values of flags.APITokenFile and flags.Password
func NewAuth() (Auth, error) {
	if flags.APITokenFile() != "" && flags.Password() != "" {
		return nil, errors.New("cannot use password- and token-based authentication at the same time")
	}
	// If Password flag is set, use the basic authenticator
	if flags.Password() != "" {
		return &basicAuthenticator{pw: flags.Password()}, nil
	}
	return newAPITokenAuthenticator()
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

func newAPITokenAuthenticator() (*apiTokenAuthenticator, error) {
	if flags.APITokenFile() == "" && env.TokenEnv.Setting() == "" {
		return nil, errors.New("no token set via either token file or the environment variable ROX_API_TOKEN")
	}
	tokenAuthenticator := &apiTokenAuthenticator{}
	tokenAuthenticator.token = env.TokenEnv.Setting()
	if tokenFile := flags.APITokenFile(); tokenFile != "" {
		token, err := flags.ReadTokenFromFile(tokenFile)
		if err != nil {
			return nil, err
		}
		tokenAuthenticator.token = token
	}
	return tokenAuthenticator, nil
}

// SetAuth sets the required authorization header with a token in bearer format on the given http.Request
func (a *apiTokenAuthenticator) SetAuth(req *http.Request) error {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.token))
	return nil
}
