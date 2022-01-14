package localscanner

import (
	"fmt"
	"testing"
	"time"

	"github.com/cloudflare/cfssl/helpers"
	testutilsMTLS "github.com/stackrox/rox/central/testutils/mtls"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
)

const (
	namespace = "namespace"
	clusterID = "clusterID"
)

func TestHandler(t *testing.T) {
	suite.Run(t, new(localScannerSuite))
}

type localScannerSuite struct {
	suite.Suite
	envIsolator *envisolator.EnvIsolator
}

func (s *localScannerSuite) SetupSuite() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
}

func (s *localScannerSuite) TearDownTest() {
	s.envIsolator.RestoreAll()
}

func (s *localScannerSuite) SetupTest() {
	err := testutilsMTLS.LoadTestMTLSCerts(s.envIsolator)
	s.Require().NoError(err)
}

func (s *localScannerSuite) TestCertMapContainsExpectedFiles() {
	testCases := []struct {
		service     storage.ServiceType
		expectError bool
	}{
		{storage.ServiceType_SCANNER_SERVICE, false},
		{storage.ServiceType_SCANNER_DB_SERVICE, false},
		{storage.ServiceType_SENSOR_SERVICE, true},
	}

	for _, tc := range testCases {
		certMap, err := generateServiceCertMap(tc.service, namespace, clusterID)
		if tc.expectError {
			s.Require().Error(err, tc.service)
			continue
		} else {
			s.Require().NoError(err, tc.service)
		}
		expectedFiles := []string{"ca.pem", "cert.pem", "key.pem"}
		s.Equal(len(expectedFiles), len(certMap))
		for _, key := range expectedFiles {
			s.Contains(certMap, key, tc.service)
		}
	}
}

func (s *localScannerSuite) TestValidateServiceCertificate() {
	testCases := []storage.ServiceType{
		storage.ServiceType_SCANNER_SERVICE,
		storage.ServiceType_SCANNER_DB_SERVICE,
	}

	for _, serviceType := range testCases {
		certMap, err := generateServiceCertMap(serviceType, namespace, clusterID)
		s.Require().NoError(err, serviceType)
		validatingCA, err := mtls.LoadCAForValidation(certMap["ca.pem"])
		s.Require().NoError(err, serviceType)
		s.NoError(certgen.VerifyServiceCert(certMap, validatingCA, serviceType, ""), serviceType)
	}
}

func (s *localScannerSuite) TestCertificateGeneration() {
	testCases := []struct {
		service                  storage.ServiceType
		expectOU                 string
		expectedAlternativeNames []string
	}{
		{storage.ServiceType_SCANNER_SERVICE, "SCANNER_SERVICE",
			[]string{"scanner.stackrox", "scanner.stackrox.svc", "scanner.namespace", "scanner.namespace.svc"}},
		{storage.ServiceType_SCANNER_DB_SERVICE, "SCANNER_DB_SERVICE",
			[]string{"scanner-db.stackrox", "scanner-db.stackrox.svc", "scanner-db.namespace", "scanner-db.namespace.svc"}},
	}

	for _, tc := range testCases {
		certMap, err := generateServiceCertMap(tc.service, namespace, clusterID)
		s.Require().NoError(err, tc.service)
		cert, err := helpers.ParseCertificatePEM(certMap["cert.pem"])
		s.Require().NoError(err, tc.service)

		subject := cert.Subject
		certOUs := subject.OrganizationalUnit
		s.Equal(1, len(certOUs), tc.service)
		s.Equal(tc.expectOU, certOUs[0], tc.service)

		s.Equal(fmt.Sprintf("%s: %s", tc.expectOU, clusterID), subject.CommonName, tc.service)

		certAlternativeNames := cert.DNSNames
		s.Equal(len(tc.expectedAlternativeNames), len(certAlternativeNames), tc.service)
		for _, name := range tc.expectedAlternativeNames {
			s.Contains(certAlternativeNames, name, tc.service)
		}
		s.Equal(cert.NotBefore.Add(2*24*time.Hour), cert.NotAfter, tc.service)
	}
}

func (s *localScannerSuite) TestServiceIssueLocalScannerCertsFeatureFlagDisabled() {
	s.envIsolator.Setenv(features.LocalImageScanning.EnvVar(), "false")
	if features.LocalImageScanning.Enabled() {
		s.T().Skip()
	}

	_, err := IssueLocalScannerCerts(namespace, clusterID)

	s.Error(err)
}

func (s *localScannerSuite) TestServiceIssueLocalScannerCerts() {
	s.envIsolator.Setenv(features.LocalImageScanning.EnvVar(), "true")
	if !features.LocalImageScanning.Enabled() {
		s.T().Skip()
	}
	testCases := map[string]struct {
		namespace  string
		shouldFail bool
	}{
		"no parameter missing": {namespace, false},
		"namespace missing":    {"", true},
	}
	for tcName, tc := range testCases {
		s.Run(tcName, func() {
			certs, err := IssueLocalScannerCerts(tc.namespace, clusterID)
			if tc.shouldFail {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)
			for _, certs := range []*central.LocalScannerCertificates{
				certs.GetScannerCerts(),
				certs.GetScannerDbCerts(),
			} {
				s.Require().NotNil(certs)
				s.NotEmpty(certs.GetCa())
				s.NotEmpty(certs.GetCert())
				s.NotEmpty(certs.GetKey())
			}
		})
	}
}
