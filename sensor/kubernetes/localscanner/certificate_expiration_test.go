package localscanner

import (
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/mtls"
	testutilsMTLS "github.com/stackrox/rox/pkg/mtls/testutils"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
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
	envIsolator *envisolator.EnvIsolator
}

func (s *getSecretRenewalTimeSuite) SetupSuite() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
}

func (s *getSecretRenewalTimeSuite) SetupTest() {
	err := testutilsMTLS.LoadTestMTLSCerts(s.envIsolator)
	s.Require().NoError(err)
}

func (s *getSecretRenewalTimeSuite) TearDownTest() {
	s.envIsolator.RestoreAll()
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

func issueCertificatePEM(issueOption mtls.IssueCertOption) ([]byte, error) {
	ca, err := mtls.CAForSigning()
	if err != nil {
		return nil, err
	}
	subject := mtls.NewSubject("clusterId", storage.ServiceType_SCANNER_SERVICE)
	cert, err := ca.IssueCertForSubject(subject, issueOption)
	if err != nil {
		return nil, err
	}
	return cert.CertPEM, err
}
