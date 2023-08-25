package certificate

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/clusters"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/uuid"
)

// Provider provides CA and service certificates to the cluster init backend.
//
//go:generate mockgen-wrapper
type Provider interface {
	GetCA() (string, error)
	GetBundle() (clusters.CertBundle, uuid.UUID, error)
}

type certProviderImpl struct{}

func (c *certProviderImpl) GetCA() (string, error) {
	caCert, err := mtls.CACertPEM()
	if err != nil {
		return "", errors.Wrap(err, "retrieving CA certificate")
	}

	return string(caCert), nil
}

func (c *certProviderImpl) GetBundle() (clusters.CertBundle, uuid.UUID, error) {
	certBundle, id, err := clusters.IssueSecuredClusterInitCertificates()
	if err != nil {
		return nil, uuid.Nil, errors.Wrap(err, "generating certificates for init bundle")
	}
	return certBundle, id, nil
}

// NewProvider returns a new certificate provider.
func NewProvider() Provider {
	return &certProviderImpl{}
}
