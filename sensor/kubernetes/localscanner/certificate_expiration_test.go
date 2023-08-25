package localscanner

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/mtls"
	testutilsMTLS "github.com/stackrox/rox/pkg/mtls/testutils"
	"github.com/stretchr/testify/suite"
)

var (
	// should be the same as the expiration corresponding to `mtls.WithValidityExpiringInHours()`.
	afterOffset = 3 * time.Hour
)

func TestGetSecretRenewalTime(t *testing.T) {
	suite.Run(t, new(getSecretRenewalTimeSuite))
}

type getSecretRenewalTimeSuite struct {
	suite.Suite
}

func (s *getSecretRenewalTimeSuite) SetupTest() {
	err := testutilsMTLS.LoadTestMTLSCerts(s.T())
	s.Require().NoError(err)
}

func (s *getSecretRenewalTimeSuite) TestGetSecretsCertRenewalTime() {
	certPEMHours, err := issueCertificatePEM(mtls.WithValidityExpiringInHours())
	s.Require().NoError(err)
	certPEMDays, err := issueCertificatePEM(mtls.WithValidityExpiringInDays())
	s.Require().NoError(err)
	certificates := &storage.TypedServiceCertificateSet{
		CaPem: make([]byte, 0),
		ServiceCerts: []*storage.TypedServiceCertificate{
			{
				ServiceType: storage.ServiceType_SCANNER_SERVICE,
				Cert: &storage.ServiceCertificate{
					CertPem: certPEMHours,
				},
			},
			{
				ServiceType: storage.ServiceType_SCANNER_DB_SERVICE,
				Cert: &storage.ServiceCertificate{
					CertPem: certPEMDays,
				},
			},
		},
	}

	certRenewalTime, err := GetCertsRenewalTime(certificates)

	s.Require().NoError(err)
	certDuration := time.Until(certRenewalTime)
	s.LessOrEqual(certDuration, afterOffset/2)
}

func issueCertificate(serviceType storage.ServiceType, issueOption mtls.IssueCertOption) (*mtls.IssuedCert, error) {
	ca, err := mtls.CAForSigning()
	if err != nil {
		return nil, err
	}
	subject := mtls.NewSubject("clusterId", serviceType)
	cert, err := ca.IssueCertForSubject(subject, issueOption)
	if err != nil {
		return nil, err
	}
	return cert, err
}

func issueCertificatePEM(issueOption mtls.IssueCertOption) ([]byte, error) {
	cert, err := issueCertificate(storage.ServiceType_SCANNER_SERVICE, issueOption)
	if err != nil {
		return nil, err
	}
	return cert.CertPEM, nil
}
