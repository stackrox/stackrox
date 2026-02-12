//go:build sql_integration

package service

import (
	"context"
	"errors"
	"io"
	"testing"

	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	policyDataStoreMock "github.com/stackrox/rox/central/policy/datastore/mocks"
	signatureIntegrationDS "github.com/stackrox/rox/central/signatureintegration/datastore"
	signatureIntegrationPostgres "github.com/stackrox/rox/central/signatureintegration/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	pkgGRPC "github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
)

func TestImageServicePostgres(t *testing.T) {
	suite.Run(t, new(imageServicePostgresTestSuite))
}

type imageServicePostgresTestSuite struct {
	suite.Suite

	pool             *pgtest.TestPostgres
	imageDS          imageDataStore.DataStore
	sigIntegrationDS signatureIntegrationDS.DataStore
	service          Service

	ctx context.Context
}

func (s *imageServicePostgresTestSuite) SetupTest() {
	s.pool = pgtest.ForT(s.T())

	s.imageDS = imageDataStore.GetTestPostgresDataStore(s.T(), s.pool)

	sigStore := signatureIntegrationPostgres.New(s.pool)
	policyMock := policyDataStoreMock.NewMockDataStore(gomock.NewController(s.T()))
	s.sigIntegrationDS = signatureIntegrationDS.New(sigStore, policyMock)

	s.service = New(s.imageDS, nil, s.imageDS, nil, nil, nil, nil, nil, nil, nil, nil, nil, s.sigIntegrationDS)

	s.ctx = sac.WithAllAccess(context.Background())
}

func (s *imageServicePostgresTestSuite) TestGetImageEnrichesVerifierName() {
	// Create a signature integration with a known name.
	integration := newTestSignatureIntegration("my-cosign-verifier")
	savedIntegration, err := s.sigIntegrationDS.AddSignatureIntegration(s.ctx, integration)
	s.Require().NoError(err)
	s.Require().NotNil(savedIntegration)

	// Create an image with a verification result referencing this integration.
	image := newTestImageWithVerificationResults("sha256:aaa111", []*storage.ImageSignatureVerificationResult{
		{
			VerifierId: savedIntegration.GetId(),
			Status:     storage.ImageSignatureVerificationResult_VERIFIED,
		},
	})
	s.Require().NoError(s.imageDS.UpsertImage(s.ctx, image))

	// Call GetImage and verify that VerifierName is populated.
	resp, err := s.service.GetImage(s.ctx, &v1.GetImageRequest{Id: image.GetId()})
	s.Require().NoError(err)
	s.Require().NotNil(resp)

	results := resp.GetSignatureVerificationData().GetResults()
	s.Require().Len(results, 1)
	s.Equal("my-cosign-verifier", results[0].GetVerifierName())
}

func (s *imageServicePostgresTestSuite) TestGetImageWithUnknownVerifierId() {
	// Create an image with a verification result pointing to a non-existent integration.
	image := newTestImageWithVerificationResults("sha256:bbb222", []*storage.ImageSignatureVerificationResult{
		{
			VerifierId: "io.stackrox.signatureintegration.non-existent-id",
			Status:     storage.ImageSignatureVerificationResult_VERIFIED,
		},
	})
	s.Require().NoError(s.imageDS.UpsertImage(s.ctx, image))

	resp, err := s.service.GetImage(s.ctx, &v1.GetImageRequest{Id: image.GetId()})
	s.Require().NoError(err)
	s.Require().NotNil(resp)

	results := resp.GetSignatureVerificationData().GetResults()
	s.Require().Len(results, 1)
	s.Empty(results[0].GetVerifierName(), "VerifierName should be empty for unknown integration ID")
}

func (s *imageServicePostgresTestSuite) TestGetImageWithEmptyVerifierId() {
	// Create an image with a verification result where VerifierId is empty.
	image := newTestImageWithVerificationResults("sha256:ccc333", []*storage.ImageSignatureVerificationResult{
		{
			VerifierId: "",
			Status:     storage.ImageSignatureVerificationResult_VERIFIED,
		},
	})
	s.Require().NoError(s.imageDS.UpsertImage(s.ctx, image))

	resp, err := s.service.GetImage(s.ctx, &v1.GetImageRequest{Id: image.GetId()})
	s.Require().NoError(err)
	s.Require().NotNil(resp)

	results := resp.GetSignatureVerificationData().GetResults()
	s.Require().Len(results, 1)
	s.Empty(results[0].GetVerifierName(), "VerifierName should be empty when VerifierId is empty")
}

func (s *imageServicePostgresTestSuite) TestGetImageWithMultipleVerificationResults() {
	// Create two signature integrations with different names.
	integration1 := newTestSignatureIntegration("verifier-alpha")
	saved1, err := s.sigIntegrationDS.AddSignatureIntegration(s.ctx, integration1)
	s.Require().NoError(err)

	integration2 := newTestSignatureIntegration("verifier-beta")
	saved2, err := s.sigIntegrationDS.AddSignatureIntegration(s.ctx, integration2)
	s.Require().NoError(err)

	// Create an image with multiple verification results.
	image := newTestImageWithVerificationResults("sha256:ddd444", []*storage.ImageSignatureVerificationResult{
		{
			VerifierId: saved1.GetId(),
			Status:     storage.ImageSignatureVerificationResult_VERIFIED,
		},
		{
			VerifierId: saved2.GetId(),
			Status:     storage.ImageSignatureVerificationResult_VERIFIED,
		},
	})
	s.Require().NoError(s.imageDS.UpsertImage(s.ctx, image))

	resp, err := s.service.GetImage(s.ctx, &v1.GetImageRequest{Id: image.GetId()})
	s.Require().NoError(err)
	s.Require().NotNil(resp)

	results := resp.GetSignatureVerificationData().GetResults()
	s.Require().Len(results, 2)

	// Build a map of verifier ID to verifier name for order-independent assertion.
	nameByID := make(map[string]string, len(results))
	for _, r := range results {
		nameByID[r.GetVerifierId()] = r.GetVerifierName()
	}
	s.Equal("verifier-alpha", nameByID[saved1.GetId()])
	s.Equal("verifier-beta", nameByID[saved2.GetId()])
}

func (s *imageServicePostgresTestSuite) TestExportImagesEnrichesVerifierName() {
	// Create a signature integration.
	integration := newTestSignatureIntegration("export-verifier")
	saved, err := s.sigIntegrationDS.AddSignatureIntegration(s.ctx, integration)
	s.Require().NoError(err)

	// Create an image with a verification result referencing this integration.
	image := newTestImageWithVerificationResults("sha256:eee555", []*storage.ImageSignatureVerificationResult{
		{
			VerifierId: saved.GetId(),
			Status:     storage.ImageSignatureVerificationResult_VERIFIED,
		},
	})
	s.Require().NoError(s.imageDS.UpsertImage(s.ctx, image))

	// Set up a gRPC streaming server.
	conn, closeFunc, err := pkgGRPC.CreateTestGRPCStreamingService(
		s.ctx,
		s.T(),
		func(registrar grpc.ServiceRegistrar) {
			v1.RegisterImageServiceServer(registrar, s.service)
		},
	)
	s.Require().NoError(err)
	defer closeFunc()

	client := v1.NewImageServiceClient(conn)
	stream, err := client.ExportImages(s.ctx, &v1.ExportImageRequest{Timeout: 60})
	s.Require().NoError(err)

	var exported []*storage.Image
	for {
		resp, recvErr := stream.Recv()
		if errors.Is(recvErr, io.EOF) {
			break
		}
		s.Require().NoError(recvErr)
		exported = append(exported, resp.GetImage())
	}

	s.Require().Len(exported, 1)
	results := exported[0].GetSignatureVerificationData().GetResults()
	s.Require().Len(results, 1)
	s.Equal("export-verifier", results[0].GetVerifierName())
}

// newTestSignatureIntegration creates a minimal SignatureIntegration for testing.
func newTestSignatureIntegration(name string) *storage.SignatureIntegration {
	return &storage.SignatureIntegration{
		Name: name,
		Cosign: &storage.CosignPublicKeyVerification{
			PublicKeys: []*storage.CosignPublicKeyVerification_PublicKey{
				{
					Name:            "key1",
					PublicKeyPemEnc: "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAryQICCl6NZ5gDKrnSztO\n3Hy8PEUcuyvg/ikC+VcIo2SFFSf18a3IMYldIugqqqZCs4/4uVW3sbdLs/6PfgdX\n7O9D22ZiFWHPYA2k2N744MNiCD1UE+tJyllUhSblK48bn+v1oZHCM0nYQ2NqUkvS\nj+hwUU3RiWl7x3D2s9wSdNt7XUtW05a/FXehsPSiJfKvHJJnGOX0BgTvkLnkAOTd\nOrUZ/wK69Dzu4IvrN4vs9Nes8vbwPa/ddZEzGR0cQMt0JBkhk9kU/qwqUseP1QRJ\n5I1jR4g8aYPL/ke9K35PxZWuDp3U0UPAZ3PjFAh+5T+fc7gzCs9dPzSHloruU+gl\nFQIDAQAB\n-----END PUBLIC KEY-----",
				},
			},
		},
	}
}

// newTestImageWithVerificationResults creates a minimal Image with the given
// signature verification results.
func newTestImageWithVerificationResults(id string, results []*storage.ImageSignatureVerificationResult) *storage.Image {
	return &storage.Image{
		Id: id,
		Name: &storage.ImageName{
			FullName: "docker.io/library/test:" + id,
		},
		SignatureVerificationData: &storage.ImageSignatureVerificationData{
			Results: results,
		},
	}
}
