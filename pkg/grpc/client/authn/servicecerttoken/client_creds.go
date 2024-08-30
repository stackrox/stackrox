package servicecerttoken

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/grpc/common/authn/servicecerttoken"
	"google.golang.org/grpc/credentials"
)

// serviceCertClientCreds injects ServiceCert token into GRPC calls.
type serviceCertClientCreds struct {
	cert *tls.Certificate
}

// NewServiceCertClientCreds creates an injector injecting ServiceCert tokens derived from the given certificate.
func NewServiceCertClientCreds(cert *tls.Certificate) credentials.PerRPCCredentials {
	return &serviceCertClientCreds{
		cert: cert,
	}
}

func (i *serviceCertClientCreds) GetRequestMetadata(_ context.Context, _ ...string) (map[string]string, error) {
	// There is no way to derived from the TLS connection state whether a client certificate is in use, so just inject
	// the authorization header in any case to be on the safe side.
	token, err := servicecerttoken.CreateToken(i.cert, time.Now())
	if err != nil {
		return nil, errors.Wrap(err, "creating service cert token")
	}
	return map[string]string{
		"authorization": fmt.Sprintf("%s %s", servicecerttoken.TokenType, token),
	}, nil
}

func (i *serviceCertClientCreds) RequireTransportSecurity() bool {
	return true
}
