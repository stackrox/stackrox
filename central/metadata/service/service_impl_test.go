//go:build sql_integration

package service

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/cloudflare/cfssl/csr"
	"github.com/cloudflare/cfssl/initca"
	cTLS "github.com/google/certificate-transparency-go/tls"
	systemInfoStorage "github.com/stackrox/rox/central/systeminfo/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn"
	mockIdentity "github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	"github.com/stackrox/rox/pkg/mtls"
	testutilsMTLS "github.com/stackrox/rox/pkg/mtls/testutils"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	//#nosec G101 -- This is a false positive
	validChallengeToken   = "h83_PGhSqS8OAvplb8asYMfPHy1JhVVMKcajYyKmrIU="
	invalidChallengeToken = "invalid"
)

func TestServiceImpl(t *testing.T) {
	suite.Run(t, new(serviceImplTestSuite))
}

func TestAuthz(t *testing.T) {
	testutils.AssertAuthzWorks(t, New().(*serviceImpl))
}

type serviceImplTestSuite struct {
	suite.Suite

	mockCtrl *gomock.Controller
}

func (s *serviceImplTestSuite) SetupTest() {
	wd, err := os.Getwd()
	s.Require().NoError(err)

	testdata := filepath.Join(wd, "testdata")
	s.T().Setenv("ROX_MTLS_ADDITIONAL_CA_DIR", path.Join(testdata, "additional-ca"))

	err = testutilsMTLS.LoadTestMTLSCerts(s.T())
	s.Require().NoError(err)

	s.mockCtrl = gomock.NewController(s.T())
}

func (s *serviceImplTestSuite) TestTLSChallenge() {
	service := New().(*serviceImpl)
	req := &v1.TLSChallengeRequest{
		ChallengeToken: validChallengeToken,
	}

	resp, err := service.TLSChallenge(context.TODO(), req)
	s.Require().NoError(err)

	trustInfo := &v1.TrustInfo{}
	err = trustInfo.UnmarshalVTUnsafe(resp.GetTrustInfoSerialized())
	s.Require().NoError(err)

	// Verify that additional CAs were received
	s.Require().Len(trustInfo.GetAdditionalCas(), 2)
	additionalCACert, err := x509.ParseCertificate(trustInfo.GetAdditionalCas()[0])
	s.Require().NoError(err)
	s.Equal("nginx LoadBalancer", additionalCACert.Subject.CommonName)

	// Verify signature
	s.Require().Len(trustInfo.GetCertChain(), 2)
	cert, err := x509.ParseCertificate(trustInfo.GetCertChain()[0])
	s.Require().NoError(err)

	err = verifySignature(cert, resp)
	s.Require().NoError(err, "Could not verify central auth signature")

	s.Contains(cert.DNSNames, "central.stackrox", "Expected leaf certificate.")
}

func (s *serviceImplTestSuite) TestTLSChallenge_VerifySignatureWithCACert_ShouldFail() {
	service := New().(*serviceImpl)
	req := &v1.TLSChallengeRequest{
		ChallengeToken: validChallengeToken,
	}

	resp, err := service.TLSChallenge(context.TODO(), req)
	s.Require().NoError(err)

	trustInfo := &v1.TrustInfo{}
	err = trustInfo.UnmarshalVTUnsafe(resp.GetTrustInfoSerialized())
	s.Require().NoError(err)

	// Read root CA from response
	caCert := trustInfo.GetCertChain()[len(trustInfo.GetCertChain())-1]
	cert, err := x509.ParseCertificate(caCert)
	s.Require().NoError(err)
	s.Equal(cert.Subject.CommonName, "StackRox Certificate Authority", "Not a root CA?")
	s.True(cert.IsCA)

	err = verifySignature(cert, resp)
	s.Error(err)
	s.Equal("failed to verify rsa signature: crypto/rsa: verification error", err.Error())
}

func (s *serviceImplTestSuite) TestTLSChallenge_ShouldFailWithoutChallenge() {
	service := New().(*serviceImpl)
	req := &v1.TLSChallengeRequest{}

	resp, err := service.TLSChallenge(context.TODO(), req)
	s.Require().Error(err)
	s.ErrorIs(err, errox.InvalidArgs)
	s.Nil(resp)
}

func (s *serviceImplTestSuite) TestTLSChallenge_ShouldFailWithInvalidToken() {
	service := New().(*serviceImpl)
	req := &v1.TLSChallengeRequest{
		ChallengeToken: invalidChallengeToken,
	}

	resp, err := service.TLSChallenge(context.TODO(), req)
	s.Require().Error(err)
	s.ErrorIs(err, errox.InvalidArgs)
	s.Nil(resp)
}

func verifySignature(cert *x509.Certificate, resp *v1.TLSChallengeResponse) error {
	return cTLS.VerifySignature(cert.PublicKey, resp.GetTrustInfoSerialized(), cTLS.DigitallySigned{
		Signature: resp.GetSignature(),
		Algorithm: cTLS.SignatureAndHashAlgorithm{
			Hash:      cTLS.SHA256,
			Signature: cTLS.SignatureAlgorithmFromPubKey(cert.PublicKey),
		},
	})
}

func (s *serviceImplTestSuite) TestDatabaseStatus() {
	// Need to fake being logged in
	mockID := mockIdentity.NewMockIdentity(s.mockCtrl)
	ctx := authn.ContextWithIdentity(sac.WithAllAccess(context.Background()), mockID, s.T())

	tp := pgtest.ForT(s.T())
	service := serviceImpl{db: tp.DB}

	dbStatus, err := service.GetDatabaseStatus(ctx, nil)
	s.NoError(err)
	s.True(dbStatus.GetDatabaseAvailable())
	s.Equal(v1.DatabaseStatus_PostgresDB, dbStatus.GetDatabaseType())
	s.NotEqual("", dbStatus.GetDatabaseVersion())

	dbStatus, err = service.GetDatabaseStatus(context.Background(), nil)
	s.NoError(err)
	s.True(dbStatus.GetDatabaseAvailable())
	s.Equal(v1.DatabaseStatus_Hidden, dbStatus.GetDatabaseType())
	s.Equal("", dbStatus.GetDatabaseVersion())

	tp.Close()
	dbStatus, err = service.GetDatabaseStatus(context.Background(), nil)
	s.NoError(err)
	s.False(dbStatus.GetDatabaseAvailable())
	s.Equal(v1.DatabaseStatus_Hidden, dbStatus.GetDatabaseType())
	s.Equal("", dbStatus.GetDatabaseVersion())
}

func (s *serviceImplTestSuite) TestDatabaseBackupStatus() {
	tp := pgtest.ForT(s.T())

	srv := &serviceImpl{
		db:              tp.DB,
		systemInfoStore: systemInfoStorage.New(tp.DB),
	}
	ctx := sac.WithAllAccess(context.Background())
	expected := &storage.SystemInfo{
		BackupInfo: &storage.BackupInfo{
			Status:          storage.OperationStatus_PASS,
			BackupLastRunAt: protocompat.TimestampNow(),
		},
	}
	err := srv.systemInfoStore.Upsert(ctx, expected)
	s.NoError(err)
	actual, err := srv.GetDatabaseBackupStatus(ctx, &v1.Empty{})
	s.NoError(err)
	protoassert.Equal(s.T(), expected.GetBackupInfo(), actual.GetBackupInfo())
}

func (s *serviceImplTestSuite) TestGetCentralCapabilities() {
	ctx := context.Background()

	s.Run("when managed central", func() {
		s.T().Setenv("ROX_MANAGED_CENTRAL", "true")

		caps, err := New().GetCentralCapabilities(ctx, nil)

		s.NoError(err)
		s.Equal(v1.CentralServicesCapabilities_CapabilityDisabled, caps.GetCentralScanningCanUseContainerIamRoleForEcr())
		s.Equal(v1.CentralServicesCapabilities_CapabilityDisabled, caps.GetCentralCanUseCloudBackupIntegrations())
		s.Equal(v1.CentralServicesCapabilities_CapabilityDisabled, caps.GetCentralCanDisplayDeclarativeConfigHealth())
		s.Equal(v1.CentralServicesCapabilities_CapabilityDisabled, caps.GetCentralCanUpdateCert())
		s.Equal(v1.CentralServicesCapabilities_CapabilityAvailable, caps.GetCentralCanUseAcscsEmailIntegration())
	})

	cases := map[string]string{"false": "false", "<empty>": ""}

	for name, val := range cases {
		s.Run(fmt.Sprintf("when not managed central (%s)", name), func() {
			s.T().Setenv("ROX_MANAGED_CENTRAL", val)

			caps, err := New().GetCentralCapabilities(ctx, nil)

			s.NoError(err)
			s.Equal(v1.CentralServicesCapabilities_CapabilityAvailable, caps.GetCentralScanningCanUseContainerIamRoleForEcr())
			s.Equal(v1.CentralServicesCapabilities_CapabilityAvailable, caps.GetCentralCanUseCloudBackupIntegrations())
			s.Equal(v1.CentralServicesCapabilities_CapabilityAvailable, caps.GetCentralCanDisplayDeclarativeConfigHealth())
			s.Equal(v1.CentralServicesCapabilities_CapabilityAvailable, caps.GetCentralCanUpdateCert())
		})
	}
}

func (s *serviceImplTestSuite) TestIssueSecondaryCALeafCert() {
	secondaryCA := s.createTestCA("Test Secondary CA", "17520h")

	s.Run("successful certificate generation", func() {
		mockProvider := &testCertificateProvider{
			secondaryCA:         secondaryCA,
			shouldFailSecondary: false,
		}

		cert, err := issueSecondaryCALeafCert(mockProvider)
		s.Require().NoError(err)
		s.Require().NotNil(cert.PrivateKey)
		s.Require().Len(cert.Certificate, 1)

		parsedCert, err := x509.ParseCertificate(cert.Certificate[0])
		s.Require().NoError(err)
		s.Equal(mtls.CentralSubject.CN(), parsedCert.Subject.CommonName)
	})

	s.Run("secondary CA loading failure", func() {
		mockProvider := &testCertificateProvider{
			secondaryCA:         secondaryCA,
			shouldFailSecondary: true,
			failSecondaryErr:    errors.New("secondary CA file not found"),
		}

		_, err := issueSecondaryCALeafCert(mockProvider)
		s.Require().Error(err)
		s.Contains(err.Error(), "failed to load secondary CA for signing")
		s.Contains(err.Error(), "secondary CA file not found")
	})
}

func (s *serviceImplTestSuite) createTestCA(commonName, expiry string) mtls.CA {
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

type testCertificateProvider struct {
	primaryCA           mtls.CA
	secondaryCA         mtls.CA
	primaryLeafCert     tls.Certificate
	shouldFailSecondary bool
	failSecondaryErr    error
}

func (m *testCertificateProvider) GetPrimaryCACert() (*x509.Certificate, []byte, error) {
	cert := m.primaryCA.Certificate()
	return cert, cert.Raw, nil
}

func (m *testCertificateProvider) GetPrimaryLeafCert() (tls.Certificate, error) {
	return m.primaryLeafCert, nil
}

func (m *testCertificateProvider) GetSecondaryCAForSigning() (mtls.CA, error) {
	if m.shouldFailSecondary {
		if m.failSecondaryErr != nil {
			return nil, m.failSecondaryErr
		}
		return nil, errors.New("secondary CA not found")
	}
	return m.secondaryCA, nil
}

func (m *testCertificateProvider) GetSecondaryCACert() (*x509.Certificate, []byte, error) {
	if m.shouldFailSecondary {
		if m.failSecondaryErr != nil {
			return nil, nil, m.failSecondaryErr
		}
		return nil, nil, errors.New("secondary CA not found")
	}
	cert := m.secondaryCA.Certificate()
	return cert, cert.Raw, nil
}

func (m *testCertificateProvider) GetSecondaryLeafCert() (tls.Certificate, error) {
	return issueSecondaryCALeafCert(m)
}

func (s *serviceImplTestSuite) TestTLSChallengeWithSecondaryCA() {
	// Create test CAs
	primaryCA := s.createTestCA("Test Primary CA", "8760h")
	secondaryCA := s.createTestCA("Test Secondary CA", "17520h")

	// Create a primary leaf certificate
	issuedCert, err := primaryCA.IssueCertForSubject(mtls.CentralSubject)
	s.Require().NoError(err)
	primaryLeafCert, err := tls.X509KeyPair(issuedCert.CertPEM, issuedCert.KeyPEM)
	s.Require().NoError(err)

	s.Run("with both primary and secondary CA", func() {
		mockProvider := &testCertificateProvider{
			primaryCA:           primaryCA,
			secondaryCA:         secondaryCA,
			primaryLeafCert:     primaryLeafCert,
			shouldFailSecondary: false,
		}

		service := &serviceImpl{
			certProvider: mockProvider,
		}

		req := &v1.TLSChallengeRequest{
			ChallengeToken: validChallengeToken,
		}

		resp, err := service.TLSChallenge(context.TODO(), req)
		s.Require().NoError(err)
		s.Require().NotNil(resp)

		trustInfo := &v1.TrustInfo{}
		err = trustInfo.UnmarshalVTUnsafe(resp.GetTrustInfoSerialized())
		s.Require().NoError(err)

		// Verify primary certificate chain
		s.Require().Len(trustInfo.GetCertChain(), 2)
		s.Equal(validChallengeToken, trustInfo.GetSensorChallenge())
		s.NotEmpty(trustInfo.GetCentralChallenge())

		// Verify secondary certificate chain and signature are present
		s.Require().Len(trustInfo.GetSecondaryCertChain(), 2)
		s.NotEmpty(resp.GetSignatureSecondaryCa())
		s.NotEqual(trustInfo.GetCertChain()[0], trustInfo.GetSecondaryCertChain()[0])
	})

	s.Run("with broken secondary CA", func() {
		mockProvider := &testCertificateProvider{
			primaryCA:           primaryCA,
			secondaryCA:         secondaryCA,
			primaryLeafCert:     primaryLeafCert,
			shouldFailSecondary: true,
		}

		service := &serviceImpl{
			certProvider: mockProvider,
		}

		req := &v1.TLSChallengeRequest{
			ChallengeToken: validChallengeToken,
		}

		resp, err := service.TLSChallenge(context.TODO(), req)
		s.Require().NoError(err) // broken secondary CA should not fail the entire request
		s.Require().NotNil(resp)

		trustInfo := &v1.TrustInfo{}
		err = trustInfo.UnmarshalVTUnsafe(resp.GetTrustInfoSerialized())
		s.Require().NoError(err)

		s.Require().Len(trustInfo.GetCertChain(), 2)
		s.Empty(trustInfo.GetSecondaryCertChain())
		s.Empty(resp.GetSignatureSecondaryCa())
	})
}
