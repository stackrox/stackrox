package zip

import (
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/stackrox/rox/central/serviceidentities/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	testutilsMTLS "github.com/stackrox/rox/pkg/mtls/testutils"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestGenerateCerts(t *testing.T) {
	suite.Run(t, new(generateCertsTestSuite))
}

type generateCertsTestSuite struct {
	suite.Suite
}

func (s *generateCertsTestSuite) SetupTest() {
	err := testutilsMTLS.LoadTestMTLSCerts(s.T())
	s.Require().NoError(err)
}

func (s *generateCertsTestSuite) TestGenerateCertsAndAddToZip() {
	cluster := &storage.Cluster{
		Id:                  "123",
		AdmissionController: true,
	}

	ctrl := gomock.NewController(s.T())
	mockStore := mocks.NewMockDataStore(ctrl)
	mockStore.EXPECT().AddServiceIdentity(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)

	certs, err := GenerateCertsAndAddToZip(nil, cluster, mockStore)
	s.Require().NoError(err)

	x509CACert, _ := pem.Decode(certs.Files["secrets/ca.pem"])
	caPem, err := x509.ParseCertificate(x509CACert.Bytes)
	s.Require().NoError(err)
	s.Equal("StackRox Certificate Authority", caPem.Subject.CommonName)

	x509SensorCert, _ := pem.Decode(certs.Files["secrets/sensor-cert.pem"])
	sensorPem, err := x509.ParseCertificate(x509SensorCert.Bytes)
	s.Require().NoError(err)
	s.Equal("SENSOR_SERVICE: 123", sensorPem.Subject.CommonName)

	x509CollectorCert, _ := pem.Decode(certs.Files["secrets/collector-cert.pem"])
	collectorPem, err := x509.ParseCertificate(x509CollectorCert.Bytes)
	s.Require().NoError(err)
	s.Equal("COLLECTOR_SERVICE: 123", collectorPem.Subject.CommonName)

	x509AdmissionsCert, _ := pem.Decode(certs.Files["secrets/admission-control-cert.pem"])
	admissionPem, err := x509.ParseCertificate(x509AdmissionsCert.Bytes)
	s.Require().NoError(err)
	s.Equal("ADMISSION_CONTROL_SERVICE: 123", admissionPem.Subject.CommonName)
}
