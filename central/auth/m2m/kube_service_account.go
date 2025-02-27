package m2m

import (
	"os"

	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/pkg/errors"
	pkgjwt "github.com/stackrox/rox/pkg/jwt"
)

//go:generate mockgen-wrapper
type ServiceAccountIssuerFetcher interface {
	GetServiceAccountIssuer() (string, error)
}

type serviceAccountTokenReader interface {
	readToken() (string, error)
}

type kubeServiceAccountIssuerFetcher struct {
	reader serviceAccountTokenReader
}

// GetServiceAccountIssuer takes a base64-encoded JWT and returns the "iss" (issuer) claim value.
func (k kubeServiceAccountIssuerFetcher) GetServiceAccountIssuer() (string, error) {
	token, err := k.reader.readToken()
	if err != nil {
		return "", errors.Wrap(err, "Failed to read kube service account token")
	}

	parsedJwt, err := pkgjwt.ParseSigned(token)
	if err != nil {
		return "", errors.Wrap(err, "Failed to parse service account JWT")
	}

	claims := jwt.Claims{}
	if err = parsedJwt.UnsafeClaimsWithoutVerification(&claims); err != nil {
		return "", errors.Wrap(err, "Failed to parse service account JWT claims")
	}

	return claims.Issuer, nil
}

type kubeServiceAccountTokenReader struct{}

func (k kubeServiceAccountTokenReader) readToken() (string, error) {
	token, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return "", errors.WithMessage(err, "error reading service account token file")
	}

	return string(token), nil
}

func NewServiceAccountIssuerFetcher() kubeServiceAccountIssuerFetcher {
	return kubeServiceAccountIssuerFetcher{
		reader: kubeServiceAccountTokenReader{},
	}
}
