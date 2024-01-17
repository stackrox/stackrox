package localscanner

import (
	"fmt"
	"testing"
	"time"

	"github.com/cloudflare/cfssl/helpers"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/mtls"
	testutilsMTLS "github.com/stackrox/rox/pkg/mtls/testutils"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/testutils"
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
}

func (s *localScannerSuite) SetupTest() {
	err := testutilsMTLS.LoadTestMTLSCerts(s.T())
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
		s.Run(tc.service.String(), func() {
			certMap, err := generateServiceCertMap(tc.service, namespace, clusterID)
			if tc.expectError {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)
			expectedFiles := []string{"ca.pem", "cert.pem", "key.pem"}
			s.Len(certMap, len(expectedFiles))
			for _, key := range expectedFiles {
				s.Contains(certMap, key)
			}
		})
	}
}

func (s *localScannerSuite) TestValidateServiceCertificate() {
	testCases := []storage.ServiceType{
		storage.ServiceType_SCANNER_SERVICE,
		storage.ServiceType_SCANNER_DB_SERVICE,
	}

	for _, serviceType := range testCases {
		s.Run(serviceType.String(), func() {
			certMap, err := generateServiceCertMap(serviceType, namespace, clusterID)
			s.Require().NoError(err)
			validatingCA, err := mtls.LoadCAForValidation(certMap["ca.pem"])
			s.Require().NoError(err)
			s.NoError(certgen.VerifyServiceCertAndKey(certMap, "", validatingCA, serviceType))
		})
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
		{storage.ServiceType_SCANNER_V4_INDEXER_SERVICE, "SCANNER_V4_INDEXER_SERVICE",
			[]string{"scanner-v4-indexer.stackrox", "scanner-v4-indexer.stackrox.svc", "scanner-v4-indexer.namespace", "scanner-v4-indexer.namespace.svc"}},
		{storage.ServiceType_SCANNER_V4_DB_SERVICE, "SCANNER_V4_DB_SERVICE",
			[]string{"scanner-v4-db.stackrox", "scanner-v4-db.stackrox.svc", "scanner-v4-db.namespace", "scanner-v4-db.namespace.svc"}},
	}

	for _, tc := range testCases {
		s.Run(tc.service.String(), func() {
			certMap, err := generateServiceCertMap(tc.service, namespace, clusterID)
			s.Require().NoError(err)
			cert, err := helpers.ParseCertificatePEM(certMap["cert.pem"])
			s.Require().NoError(err)

			subject := cert.Subject
			certOUs := subject.OrganizationalUnit
			s.Require().Len(certOUs, 1)
			s.Equal(tc.expectOU, certOUs[0])

			s.Equal(fmt.Sprintf("%s: %s", tc.expectOU, clusterID), subject.CommonName)

			certAlternativeNames := cert.DNSNames
			s.ElementsMatch(tc.expectedAlternativeNames, certAlternativeNames)
			s.Equal(cert.NotBefore.Add((365*24+1)*time.Hour), cert.NotAfter)
		})
	}
}

func (s *localScannerSuite) TestServiceIssueLocalScannerCerts() {
	getServiceTypes := func() set.FrozenSet[string] {
		serviceTypes := v2ServiceTypes
		if features.ScannerV4.Enabled() {
			serviceTypes = allSupportedServiceTypes
		}
		serviceTypeNames := make([]string, 0, serviceTypes.Cardinality())
		for _, serviceType := range serviceTypes.AsSlice() {
			serviceTypeNames = append(serviceTypeNames, serviceType.String())
		}
		return set.NewFrozenSet(serviceTypeNames...)
	}
	testCases := map[string]struct {
		namespace        string
		clusterID        string
		shouldFail       bool
		scannerV4Enabled bool
	}{
		"no parameter missing": {
			namespace:  namespace,
			clusterID:  clusterID,
			shouldFail: false,
		},
		"no parameter missing, scanner v4 enabled": {
			namespace:        namespace,
			clusterID:        clusterID,
			shouldFail:       false,
			scannerV4Enabled: true,
		},
		"namespace missing": {
			namespace:  "",
			clusterID:  clusterID,
			shouldFail: true,
		},
		"clusterID missing": {
			namespace:  namespace,
			clusterID:  "",
			shouldFail: true,
		},
	}
	scannerV4Enabled := features.ScannerV4.Enabled()
	for tcName, tc := range testCases {
		s.Run(tcName, func() {
			testutils.MustUpdateFeature(s.T(), features.ScannerV4, tc.scannerV4Enabled)
			certs, err := IssueLocalScannerCerts(tc.namespace, tc.clusterID)
			if tc.shouldFail {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)
			s.Require().NotNil(certs.GetCaPem())
			s.Require().NotEmpty(certs.GetServiceCerts())
			expectedServiceTypes := getServiceTypes().Unfreeze()
			for _, cert := range certs.ServiceCerts {
				certService := cert.GetServiceType().String()
				// Verifies that the service types of the returned certificates are supported Scanner service types.
				s.Contains(expectedServiceTypes, certService, "[%s] unexpected certificate service type %q", tcName, certService)
				expectedServiceTypes.Remove(certService)
				s.NotEmpty(cert.GetCert().GetCertPem())
				s.NotEmpty(cert.GetCert().GetKeyPem())
			}
			// Verify that certificates for all expected service types have been returned.
			s.Empty(expectedServiceTypes.AsSlice(), "[%s] not all expected certificates were returned by IssueLocalScannerCerts", tcName)
		})
	}
	testutils.MustUpdateFeature(s.T(), features.ScannerV4, scannerV4Enabled)
}
