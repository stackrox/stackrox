package userpass

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/stackrox/rox/central/role"
	"github.com/stackrox/rox/pkg/auth/htpasswd"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/logging"
)

const (
	filename = "/run/secrets/stackrox.io/htpasswd/htpasswd"

	defaultTTL = 24 * time.Hour
)

var (
	singleton     Issuer
	initSingleton sync.Once

	log = logging.LoggerForModule()
)

// A Issuer checks provided username/password combinations and exchanges
// them for signed JSON Web Tokens (JWTs).
type Issuer struct {
	hashes *htpasswd.HashFile
	issuer tokens.Issuer
}

// Singleton creates or returns an Issuer.
func Singleton(issuerFactory tokens.IssuerFactory, r io.Reader) *Issuer {
	initSingleton.Do(func() {
		u, err := newUserPass(issuerFactory, r)
		if err != nil {
			log.Panicf("Could not create username/password checker: %s", err)
		}
		singleton = *u
	})
	return &singleton
}

// MustOpenHtpasswd opens the htpasswd hash file or panics.
func MustOpenHtpasswd() io.Reader {
	f, err := os.Open(filename)
	if err != nil {
		log.Panicf("htpasswd open: %s", err)
	}
	return f
}

func newUserPass(issuerFactory tokens.IssuerFactory, r io.Reader) (*Issuer, error) {
	hf, err := htpasswd.ReadHashFile(r)
	if err != nil {
		return nil, fmt.Errorf("read: %s", err)
	}
	issuer, err := issuerFactory.CreateIssuer(&source{}, tokens.WithDefaultTTL(defaultTTL))
	if err != nil {
		return nil, fmt.Errorf("creating issuer: %v", err)
	}
	return &Issuer{
		hashes: hf,
		issuer: issuer,
	}, nil
}

// IssueToken issues a token to the specified user if the provided
// username:password combination exists in the set of known hashes.
func (u *Issuer) IssueToken(user, pass string) (string, error) {
	if !u.hashes.Check(user, pass) {
		return "", errors.New("authentication failed: unknown username or password")
	}
	ti, err := u.issuer.Issue(tokens.RoxClaims{
		RoleName: role.Admin,
	})
	if err != nil {
		return "", err
	}
	return ti.Token, nil
}
