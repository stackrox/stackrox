package servicecerttoken

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/pkg/errors"
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

func (i *serviceCertClientCreds) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	token, err := createToken(i.cert, time.Now())
	if err != nil {
		return nil, errors.Wrap(err, "creating service cert token")
	}
	return map[string]string{
		"authorization": fmt.Sprintf("%s %s", tokenType, token),
	}, nil
}

func (i *serviceCertClientCreds) RequireTransportSecurity() bool {
	return true
}
