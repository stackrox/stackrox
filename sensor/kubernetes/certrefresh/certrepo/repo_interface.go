package certrepo

import (
	"github.com/stackrox/rox/generated/storage"
	"golang.org/x/net/context"
)

// ServiceCertificatesRepo is in charge of persisting and retrieving a set of service certificates, thus implementing
// the [repository pattern](https://martinfowler.com/eaaCatalog/repository.html) for *storage.TypedServiceCertificateSet.
type ServiceCertificatesRepo interface {
	// GetServiceCertificates retrieves the certificates from permanent storage.
	GetServiceCertificates(ctx context.Context) (*storage.TypedServiceCertificateSet, error)
	// EnsureServiceCertificates persists the certificates on permanent storage.
	EnsureServiceCertificates(ctx context.Context, certificates *storage.TypedServiceCertificateSet) ([]*storage.TypedServiceCertificate, error)
}
