package common

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/stackrox/rox/pkg/grpc/authn/basic"
	"github.com/stackrox/rox/roxctl/common/flags"
)

// Auth provides an abstraction to inject authentication information within http.Request
type Auth interface {
	SetAuth(req *http.Request) error
}

// newAuth creates a new Auth type which will be inferred based off of the values of flags.APITokenFile and flags.Password.
func newAuth() (Auth, error) {
	token, err := RetrieveAuthToken()
	if err != nil {
		return nil, err
	}
	if token == "" {
		// If Password flag is set, use the basic authenticator
		if flags.Password() != "" {
			return &basicAuthenticator{pw: flags.Password()}, nil
		}
		return nil, errors.New("no token set via either token file or the environment variable ROX_API_TOKEN")
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
