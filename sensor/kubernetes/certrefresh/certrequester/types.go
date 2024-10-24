package certrequester

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// CertificateRequester defines an interface for requesting TLS certificates from Central
type CertificateRequester interface {
	Start()
	Stop()
	RequestCertificates(ctx context.Context) (*IssueCertsResponse, error)
}

// IssueCertsResponse represents the response of a certificate request (a set of certificates or an error)
type IssueCertsResponse struct {
	RequestId    string
	ErrorMessage *string
	Certificates *storage.TypedServiceCertificateSet
}
