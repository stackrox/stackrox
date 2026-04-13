package authn

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/stackrox/rox/pkg/scannerv4/client"
)

// BasicAuthSetting is the environment variable which specifies the basic authentication
// scannerctl uses. It should be in the form username:password.
const BasicAuthSetting = "ROX_SCANNERCTL_BASIC_AUTH"

// ParseBasic parses basic authentication from the environment
// or, if set, the given string.
func ParseBasic(auth string) (authn.Authenticator, error) {
	if auth == "" {
		auth = os.Getenv(BasicAuthSetting)
	}
	if auth == "" {
		log.Printf("auth unspecified: using anonymous auth (use %s to set auth)", BasicAuthSetting)
		return authn.Anonymous, nil
	}

	u, p, ok := strings.Cut(auth, ":")
	if !ok {
		return nil, errors.New("invalid basic auth: expecting the username and the " +
			"password with a colon (aladdin:opensesame)")
	}

	return &authn.Basic{
		Username: u,
		Password: p,
	}, nil
}

// ToRegistryAuth converts an authn.Authenticator to a client.RegistryAuth
// by extracting the username and password from the authenticator's config.
func ToRegistryAuth(auth authn.Authenticator) (client.RegistryAuth, error) {
	cfg, err := auth.Authorization()
	if err != nil {
		return client.RegistryAuth{}, fmt.Errorf("getting auth config: %w", err)
	}
	return client.RegistryAuth{
		Username: cfg.Username,
		Password: cfg.Password,
	}, nil
}
