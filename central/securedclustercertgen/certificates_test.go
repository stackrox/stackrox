package securedclustercertgen

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cloudflare/cfssl/csr"
	"github.com/cloudflare/cfssl/helpers"
	"github.com/cloudflare/cfssl/initca"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/certgen"
	"github.com/stackrox/rox/pkg/cryptoutils"
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
	suite.Run(t, new(securedClusterCertGenSuite))
	suite.Run(t, new(securedClusterCARotationSuite))
}

type securedClusterCertGenSuite struct {
	suite.Suite
}

func (s *securedClusterCertGenSuite) SetupTest() {
	err := testutilsMTLS.LoadTestMTLSCerts(s.T())
	s.Require().NoError(err)
}

func (s *securedClusterCertGenSuite) TestCertMapContainsExpectedFiles() {
	testCases := []struct {
		service     storage.ServiceType
		expectError bool
	}{
		{storage.ServiceType_SCANNER_SERVICE, false},
		{storage.ServiceType_SCANNER_DB_SERVICE, false},
		{storage.ServiceType_SENSOR_SERVICE, true},
	}

	ca, err := mtls.CAForSigning()
	s.Require().NoError(err)

	certIssuer := certIssuerImpl{
		serviceTypes:             localScannerServiceTypes,
		signingCA:                ca,
		sensorSupportsCARotation: false,
	}
	for _, tc := range testCases {
		s.Run(tc.service.String(), func() {
			certMap, err := certIssuer.generateServiceCertMap(tc.service, namespace, clusterID)
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

func (s *securedClusterCertGenSuite) TestValidateServiceCertificate() {
	testCases := []storage.ServiceType{
		storage.ServiceType_SCANNER_SERVICE,
		storage.ServiceType_SCANNER_DB_SERVICE,
	}

	ca, err := mtls.CAForSigning()
	s.Require().NoError(err)

	certIssuer := certIssuerImpl{
		serviceTypes:             localScannerServiceTypes,
		signingCA:                ca,
		sensorSupportsCARotation: false,
	}
	for _, serviceType := range testCases {
		s.Run(serviceType.String(), func() {
			certMap, err := certIssuer.generateServiceCertMap(serviceType, namespace, clusterID)
			s.Require().NoError(err)
			validatingCA, err := mtls.LoadCAForValidation(certMap["ca.pem"])
			s.Require().NoError(err)
			s.NoError(certgen.VerifyServiceCertAndKey(certMap, "", validatingCA, serviceType, nil))
		})
	}
}

func (s *securedClusterCertGenSuite) TestLocalScannerCertificateGeneration() {
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

	ca, err := mtls.CAForSigning()
	s.Require().NoError(err)

	certIssuer := certIssuerImpl{
		serviceTypes:             localScannerServiceTypes,
		signingCA:                ca,
		sensorSupportsCARotation: false,
	}
	for _, tc := range testCases {
		s.Run(tc.service.String(), func() {
			certMap, err := certIssuer.generateServiceCertMap(tc.service, namespace, clusterID)
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

func (s *securedClusterCertGenSuite) TestSecuredClusterCertificateGeneration() {
	testCases := []struct {
		service                  storage.ServiceType
		expectOU                 string
		expectedAlternativeNames []string
	}{
		{storage.ServiceType_SENSOR_SERVICE, "SENSOR_SERVICE",
			[]string{"sensor.stackrox", "sensor.stackrox.svc", "sensor-webhook.stackrox.svc", "sensor.namespace", "sensor.namespace.svc", "sensor-webhook.namespace.svc"}},
		{storage.ServiceType_COLLECTOR_SERVICE, "COLLECTOR_SERVICE",
			[]string{"collector.stackrox", "collector.stackrox.svc", "collector.namespace", "collector.namespace.svc"}},
		{storage.ServiceType_ADMISSION_CONTROL_SERVICE, "ADMISSION_CONTROL_SERVICE",
			[]string{"admission-control.stackrox", "admission-control.stackrox.svc", "admission-control.namespace", "admission-control.namespace.svc"}},
	}

	ca, err := mtls.CAForSigning()
	s.Require().NoError(err)

	certIssuer := certIssuerImpl{
		serviceTypes:             securedClusterServiceTypes,
		signingCA:                ca,
		sensorSupportsCARotation: false,
	}
	for _, tc := range testCases {
		s.Run(tc.service.String(), func() {
			certMap, err := certIssuer.generateServiceCertMap(tc.service, namespace, clusterID)
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

func (s *securedClusterCertGenSuite) TestServiceIssueLocalScannerCerts() {
	getServiceTypes := func() set.FrozenSet[string] {
		serviceTypes := scannerV2ServiceTypes
		if features.ScannerV4.Enabled() {
			serviceTypes = localScannerServiceTypes
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
			for _, cert := range certs.GetServiceCerts() {
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

// TestServiceIssueSecuredClusterCerts checks the issuance of certificates for secured clusters.
func (s *securedClusterCertGenSuite) TestServiceIssueSecuredClusterCerts() {
	testCases := map[string]struct {
		namespace  string
		clusterID  string
		shouldFail bool
	}{
		"valid parameters": {
			namespace:  namespace,
			clusterID:  clusterID,
			shouldFail: false,
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

	for tcName, tc := range testCases {
		s.Run(tcName, func() {
			certs, err := IssueSecuredClusterCerts(tc.namespace, tc.clusterID, false, "")
			if tc.shouldFail {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)
			s.Require().NotNil(certs)
			s.Require().NotNil(certs.GetCaPem())
			s.Require().NotEmpty(certs.GetServiceCerts())

			expectedServiceTypes := allSupportedServiceTypes.Unfreeze()
			for _, cert := range certs.GetServiceCerts() {
				certService := cert.GetServiceType()
				s.Contains(expectedServiceTypes, certService, "[%s] unexpected certificate service type %q", tcName, certService)
				expectedServiceTypes.Remove(certService)
				s.NotEmpty(cert.GetCert().GetCertPem())
				s.NotEmpty(cert.GetCert().GetKeyPem())
			}
			s.Empty(expectedServiceTypes.AsSlice(), "[%s] not all expected certificates were returned by IssueSecuredClusterCerts", tcName)
		})
	}
}

type securedClusterCARotationSuite struct {
	suite.Suite
	primaryCA   mtls.CA
	secondaryCA mtls.CA
}

func (s *securedClusterCARotationSuite) SetupTest() {
	// Load the standard test CA as primary
	err := testutilsMTLS.LoadTestMTLSCerts(s.T())
	s.Require().NoError(err)

	s.primaryCA, err = mtls.CAForSigning()
	s.Require().NoError(err)

	// Create a test secondary CA with a future expiration
	s.secondaryCA = s.createTestCA("Test Secondary CA", "87600h") // 10 years
}

func (s *securedClusterCARotationSuite) createTestCA(commonName, expiry string) mtls.CA {
	caCert, _, caKey, err := initca.New(&csr.CertificateRequest{
		CN:         commonName,
		KeyRequest: csr.NewKeyRequest(),
		CA: &csr.CAConfig{
			Expiry: expiry,
		},
	})
	s.Require().NoError(err)

	ca, err := mtls.LoadCAForSigning(caCert, caKey)
	s.Require().NoError(err)
	return ca
}

func (s *securedClusterCARotationSuite) verifyServiceCertsSignedByCA(caCertPEM []byte, certs *storage.TypedServiceCertificateSet) {
	// Verify that the signing CA certificate matches the primary CA
	s.Equal(caCertPEM, certs.GetCaPem())

	// Verify that all service certs are actually signed by the expected CA
	caCert, err := helpers.ParseCertificatePEM(caCertPEM)
	s.Require().NoError(err)
	for _, serviceCert := range certs.GetServiceCerts() {
		certPEM := serviceCert.GetCert().GetCertPem()
		cert, err := helpers.ParseCertificatePEM(certPEM)
		s.Require().NoError(err)
		s.Equal(caCert.Subject, cert.Issuer, "Service cert not signed by expected CA")
	}
}

func (s *securedClusterCARotationSuite) TestIssueSecuredClusterCertsWithCAs() {
	// Create an older secondary CA that expires before the primary CA
	olderSecondaryCA := s.createTestCA("Older Secondary CA", "1h") // 1 hour

	testCases := []struct {
		name                     string
		sensorSupportsCARotation bool
		primaryCA                mtls.CA
		secondaryCA              mtls.CA
		sensorCAFingerprint      string
		expectedSigningCA        mtls.CA
		expectedBundleCertCount  int
		shouldHaveBundle         bool
	}{
		{
			name:                     "no CA rotation support",
			sensorSupportsCARotation: false,
			primaryCA:                s.primaryCA,
			secondaryCA:              s.secondaryCA,
			expectedSigningCA:        s.primaryCA,
			expectedBundleCertCount:  0, // No bundle when rotation disabled
			shouldHaveBundle:         false,
		},
		{
			name:                     "primary CA only",
			sensorSupportsCARotation: true,
			primaryCA:                s.primaryCA,
			secondaryCA:              nil,
			expectedSigningCA:        s.primaryCA,
			expectedBundleCertCount:  1,
			shouldHaveBundle:         true,
		},
		{
			name:                     "secondary CA newer than primary",
			sensorSupportsCARotation: true,
			primaryCA:                s.primaryCA,
			secondaryCA:              s.secondaryCA,
			expectedSigningCA:        s.secondaryCA,
			expectedBundleCertCount:  2,
			shouldHaveBundle:         true,
		},
		{
			name:                     "secondary CA older than primary",
			sensorSupportsCARotation: true,
			primaryCA:                s.primaryCA,
			secondaryCA:              olderSecondaryCA,
			expectedSigningCA:        s.primaryCA,
			expectedBundleCertCount:  2,
			shouldHaveBundle:         true,
		},
		{
			name:                     "secondary CA newer than primary - fingerprint prefers primary",
			sensorSupportsCARotation: false,
			primaryCA:                s.primaryCA,
			secondaryCA:              s.secondaryCA,
			sensorCAFingerprint:      cryptoutils.CertFingerprint(s.primaryCA.Certificate()),
			expectedSigningCA:        s.primaryCA,
			expectedBundleCertCount:  0,
			shouldHaveBundle:         false,
		},
		{
			name:                     "secondary CA newer than primary - fingerprint prefers secondary",
			sensorSupportsCARotation: false,
			primaryCA:                s.primaryCA,
			secondaryCA:              s.secondaryCA,
			sensorCAFingerprint:      cryptoutils.CertFingerprint(s.secondaryCA.Certificate()),
			expectedSigningCA:        s.secondaryCA,
			expectedBundleCertCount:  0,
			shouldHaveBundle:         false,
		},
		{
			name:                     "secondary CA newer than primary - random fingerprint",
			sensorSupportsCARotation: false,
			primaryCA:                s.primaryCA,
			secondaryCA:              s.secondaryCA,
			sensorCAFingerprint:      "random_fingerprint_that_matches_nothing",
			expectedSigningCA:        s.primaryCA,
			expectedBundleCertCount:  0,
			shouldHaveBundle:         false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			certs, err := IssueSecuredClusterCertsWithCAs(namespace, clusterID, tc.sensorSupportsCARotation, tc.primaryCA, tc.secondaryCA, tc.sensorCAFingerprint)
			s.Require().NoError(err)
			s.Require().NotNil(certs)
			s.Require().NotEmpty(certs.GetCaPem())
			s.Require().NotEmpty(certs.GetServiceCerts())

			// Verify the expected CA was used for signing
			s.verifyServiceCertsSignedByCA(tc.expectedSigningCA.CertPEM(), certs)

			// Verify CA bundle content
			if tc.shouldHaveBundle {
				s.Require().NotEmpty(certs.GetCaBundlePem())
				caBundleString := string(certs.GetCaBundlePem())
				s.Contains(caBundleString, "BEGIN CERTIFICATE")
				s.Contains(caBundleString, "END CERTIFICATE")

				certCount := strings.Count(caBundleString, "BEGIN CERTIFICATE")
				s.Equal(tc.expectedBundleCertCount, certCount)

				if tc.expectedBundleCertCount >= 1 {
					// Primary CA should always be in the bundle
					s.Contains(caBundleString, string(tc.primaryCA.CertPEM()))
				}
				if tc.expectedBundleCertCount >= 2 && tc.secondaryCA != nil {
					// Secondary CA should be in the bundle when present
					s.Contains(caBundleString, string(tc.secondaryCA.CertPEM()))
				}
			} else {
				s.Empty(certs.GetCaBundlePem())
			}
		})
	}
}

func (s *securedClusterCARotationSuite) TestBuildCABundle() {
	s.Run("primary CA only", func() {
		issuer := certIssuerImpl{
			serviceTypes:             allSupportedServiceTypes,
			signingCA:                s.primaryCA,
			secondaryCA:              nil,
			sensorSupportsCARotation: true,
		}

		bundle, err := issuer.buildCABundle()
		s.Require().NoError(err)
		s.Require().NotEmpty(bundle)

		bundleString := string(bundle)
		s.Contains(bundleString, "BEGIN CERTIFICATE")
		s.Contains(bundleString, "END CERTIFICATE")

		certCount := strings.Count(bundleString, "BEGIN CERTIFICATE")
		s.Equal(1, certCount)

		s.Contains(bundleString, string(s.primaryCA.CertPEM()))
	})

	s.Run("primary and secondary CA", func() {
		issuer := certIssuerImpl{
			serviceTypes:             allSupportedServiceTypes,
			signingCA:                s.primaryCA,
			secondaryCA:              s.secondaryCA,
			sensorSupportsCARotation: true,
		}

		bundle, err := issuer.buildCABundle()
		s.Require().NoError(err)
		s.Require().NotEmpty(bundle)

		bundleString := string(bundle)
		s.Contains(bundleString, "BEGIN CERTIFICATE")
		s.Contains(bundleString, "END CERTIFICATE")

		certCount := strings.Count(bundleString, "BEGIN CERTIFICATE")
		s.Equal(2, certCount)

		s.Contains(bundleString, string(s.primaryCA.CertPEM()))
		s.Contains(bundleString, string(s.secondaryCA.CertPEM()))
	})
}

func (s *securedClusterCARotationSuite) TestServiceCertificateGeneration() {
	certs, err := IssueSecuredClusterCertsWithCAs(namespace, clusterID, true, s.primaryCA, s.secondaryCA, "")
	s.Require().NoError(err)
	s.Require().NotNil(certs)

	// Verify all expected service types are present
	expectedServiceTypes := allSupportedServiceTypes.Unfreeze()
	for _, cert := range certs.GetServiceCerts() {
		certService := cert.GetServiceType()
		s.Contains(expectedServiceTypes, certService)
		expectedServiceTypes.Remove(certService)

		// Verify certificate has required fields
		s.NotEmpty(cert.GetCert().GetCertPem())
		s.NotEmpty(cert.GetCert().GetKeyPem())

		// Verify the certificate can be parsed
		parsedCert, err := helpers.ParseCertificatePEM(cert.GetCert().GetCertPem())
		s.Require().NoError(err)
		s.NotNil(parsedCert)
	}

	// Verify all expected service types were generated
	s.Empty(expectedServiceTypes.AsSlice())
}

func (s *securedClusterCARotationSuite) TestErrorHandling() {
	s.Run("empty namespace", func() {
		_, err := IssueSecuredClusterCertsWithCAs("", clusterID, true, s.primaryCA, s.secondaryCA, "")
		s.Error(err)
		s.Contains(err.Error(), "namespace is required")
	})

	s.Run("empty cluster ID", func() {
		_, err := IssueSecuredClusterCertsWithCAs(namespace, "", true, s.primaryCA, s.secondaryCA, "")
		s.Error(err)
	})

	s.Run("nil primary CA", func() {
		_, err := IssueSecuredClusterCertsWithCAs(namespace, clusterID, true, nil, s.secondaryCA, "")
		s.Error(err)
		s.Contains(err.Error(), "primary CA is required")
	})
}
