package clusters

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	testutilsMTLS "github.com/stackrox/rox/pkg/mtls/testutils"
	"github.com/stretchr/testify/suite"
)

func TestIdentity(t *testing.T) {
	suite.Run(t, new(identityTestSuite))
}

type identityTestSuite struct {
	suite.Suite
}

func (s *identityTestSuite) SetupTest() {
	err := testutilsMTLS.LoadTestMTLSCerts(s.T())
	s.Require().NoError(err)
}

func (s *identityTestSuite) TestIssueSecuredClusterInitCertificates() {
	certs, bundleUUID, err := IssueSecuredClusterInitCertificates()
	s.Require().NoError(err)
	s.NotEmpty(bundleUUID)

	issuedSensorCert := certs[storage.ServiceType_SENSOR_SERVICE]
	issuedAdmissionCert := certs[storage.ServiceType_ADMISSION_CONTROL_SERVICE]
	issuedCollectorCert := certs[storage.ServiceType_COLLECTOR_SERVICE]

	s.Equal("SENSOR_SERVICE: 00000000-0000-0000-0000-000000000000", issuedSensorCert.X509Cert.Subject.CommonName)
	s.Equal("ADMISSION_CONTROL_SERVICE: 00000000-0000-0000-0000-000000000000", issuedAdmissionCert.X509Cert.Subject.CommonName)
	s.Equal("COLLECTOR_SERVICE: 00000000-0000-0000-0000-000000000000", issuedCollectorCert.X509Cert.Subject.CommonName)

	// Validate organization contains the same init bundle IDs
	bundleID := bundleUUID.String()
	s.Equal(bundleID, issuedAdmissionCert.X509Cert.Subject.Organization[0], "Expected Organization to contain equal bundle IDs")
	s.Equal(bundleID, issuedCollectorCert.X509Cert.Subject.Organization[0], "Expected Organization to contain equal bundle IDs")
	s.Equal(bundleID, issuedSensorCert.X509Cert.Subject.Organization[0], "Expected Organization to contain equal bundle IDs")
}
