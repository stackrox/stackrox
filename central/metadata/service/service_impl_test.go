//go:build sql_integration

package service

import (
	"context"
	"crypto/x509"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/gogo/protobuf/proto"
	timestamp "github.com/gogo/protobuf/types"
	cTLS "github.com/google/certificate-transparency-go/tls"
	systemInfoStorage "github.com/stackrox/rox/central/systeminfo/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn"
	mockIdentity "github.com/stackrox/rox/pkg/grpc/authn/mocks"
	"github.com/stackrox/rox/pkg/grpc/testutils"
	testutilsMTLS "github.com/stackrox/rox/pkg/mtls/testutils"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
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
	testutils.AssertAuthzWorks(t, &serviceImpl{})
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
	service := serviceImpl{}
	req := &v1.TLSChallengeRequest{
		ChallengeToken: validChallengeToken,
	}

	resp, err := service.TLSChallenge(context.TODO(), req)
	s.Require().NoError(err)

	trustInfo := &v1.TrustInfo{}
	err = proto.Unmarshal(resp.GetTrustInfoSerialized(), trustInfo)
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
	service := serviceImpl{}
	req := &v1.TLSChallengeRequest{
		ChallengeToken: validChallengeToken,
	}

	resp, err := service.TLSChallenge(context.TODO(), req)
	s.Require().NoError(err)

	trustInfo := &v1.TrustInfo{}
	err = proto.Unmarshal(resp.GetTrustInfoSerialized(), trustInfo)
	s.Require().NoError(err)

	// Read root CA from response
	caCert := trustInfo.CertChain[len(trustInfo.CertChain)-1]
	cert, err := x509.ParseCertificate(caCert)
	s.Require().NoError(err)
	s.Equal(cert.Subject.CommonName, "StackRox Certificate Authority", "Not a root CA?")
	s.True(cert.IsCA)

	err = verifySignature(cert, resp)
	s.Error(err)
	s.Equal("failed to verify rsa signature: crypto/rsa: verification error", err.Error())
}

func (s *serviceImplTestSuite) TestTLSChallenge_ShouldFailWithoutChallenge() {
	service := serviceImpl{}
	req := &v1.TLSChallengeRequest{}

	resp, err := service.TLSChallenge(context.TODO(), req)
	s.Require().Error(err)
	s.ErrorIs(err, errox.InvalidArgs)
	s.Nil(resp)
}

func (s *serviceImplTestSuite) TestTLSChallenge_ShouldFailWithInvalidToken() {
	service := serviceImpl{}
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
	s.True(dbStatus.DatabaseAvailable)
	s.Equal(v1.DatabaseStatus_PostgresDB, dbStatus.DatabaseType)
	s.NotEqual("", dbStatus.DatabaseVersion)

	dbStatus, err = service.GetDatabaseStatus(context.Background(), nil)
	s.NoError(err)
	s.True(dbStatus.DatabaseAvailable)
	s.Equal(v1.DatabaseStatus_Hidden, dbStatus.DatabaseType)
	s.Equal("", dbStatus.DatabaseVersion)

	tp.Close()
	dbStatus, err = service.GetDatabaseStatus(context.Background(), nil)
	s.NoError(err)
	s.False(dbStatus.DatabaseAvailable)
	s.Equal(v1.DatabaseStatus_Hidden, dbStatus.DatabaseType)
	s.Equal("", dbStatus.DatabaseVersion)
}

func (s *serviceImplTestSuite) TestDatabaseBackupStatus() {
	tp := pgtest.ForT(s.T())
	defer tp.Teardown(s.T())

	srv := &serviceImpl{
		db:              tp.DB,
		systemInfoStore: systemInfoStorage.New(tp.DB),
	}
	ctx := sac.WithAllAccess(context.Background())
	expected := &storage.SystemInfo{
		BackupInfo: &storage.BackupInfo{
			Status:          storage.OperationStatus_PASS,
			BackupLastRunAt: timestamp.TimestampNow(),
		},
	}
	err := srv.systemInfoStore.Upsert(ctx, expected)
	s.NoError(err)
	actual, err := srv.GetDatabaseBackupStatus(ctx, &v1.Empty{})
	s.NoError(err)
	s.EqualValues(expected, actual)
}

func (s *serviceImplTestSuite) TestGetCentralCapabilities() {
	mockID := mockIdentity.NewMockIdentity(s.mockCtrl)
	ctx := authn.ContextWithIdentity(sac.WithNoAccess(context.Background()), mockID, s.T())

	s.Run("when managed central", func() {
		s.T().Setenv("ROX_MANAGED_CENTRAL", "true")

		caps, err := (&serviceImpl{}).GetCentralCapabilities(ctx, nil)

		s.NoError(err)
		s.Equal(v1.CentralServicesCapabilities_CapabilityDisabled, caps.GetCentralScanningCanUseContainerIamRoleForEcr())
		s.Equal(v1.CentralServicesCapabilities_CapabilityDisabled, caps.GetCentralCanUseCloudBackupIntegrations())
		s.Equal(v1.CentralServicesCapabilities_CapabilityDisabled, caps.GetCentralCanDisplayDeclarativeConfigHealth())
	})

	cases := map[string]string{"false": "false", "<empty>": ""}

	for name, val := range cases {
		s.Run(fmt.Sprintf("when not managed central (%s)", name), func() {
			s.T().Setenv("ROX_MANAGED_CENTRAL", val)

			caps, err := (&serviceImpl{}).GetCentralCapabilities(ctx, nil)

			s.NoError(err)
			s.Equal(v1.CentralServicesCapabilities_CapabilityAvailable, caps.CentralScanningCanUseContainerIamRoleForEcr)
			s.Equal(v1.CentralServicesCapabilities_CapabilityAvailable, caps.CentralCanUseCloudBackupIntegrations)
			s.Equal(v1.CentralServicesCapabilities_CapabilityAvailable, caps.CentralCanDisplayDeclarativeConfigHealth)
		})
	}
}
