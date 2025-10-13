package authn

import (
	"errors"
	"log"
	"os"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
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
