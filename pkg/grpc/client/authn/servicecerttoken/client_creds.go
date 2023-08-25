package servicecerttoken

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/grpc/common/authn/servicecerttoken"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/sliceutils"
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

// NewServiceCertInjectingRoundTripper returns an `http.RoundTripper` that injects a ServiceCert token into an HTTP
// request (provided there is no existing authorization header) before delegating to the given underlying roundtripper.
func NewServiceCertInjectingRoundTripper(cert *tls.Certificate, rt http.RoundTripper) http.RoundTripper {
	return httputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		// Do not attempt to modify an existing authorization header
		if req.Header.Get("authorization") != "" {
			return rt.RoundTrip(req)
		}

		token, err := servicecerttoken.CreateToken(cert, time.Now())
		if err != nil {
			return nil, errors.Wrap(err, "creating service cert token")
		}

		reqShallowCopy := *req
		newHeader := make(http.Header)
		for k, vs := range req.Header {
			newHeader[k] = sliceutils.ShallowClone(vs)
		}

		newHeader.Set("authorization", fmt.Sprintf("%s %s", servicecerttoken.TokenType, token))
		reqShallowCopy.Header = newHeader

		return rt.RoundTrip(&reqShallowCopy)
	})
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
