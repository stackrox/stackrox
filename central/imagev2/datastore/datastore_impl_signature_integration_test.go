package datastore

import (
	"context"
	"testing"

	mockStore "github.com/stackrox/rox/central/imagev2/datastore/store/mocks"
	"github.com/stackrox/rox/central/ranking"
	mockRisks "github.com/stackrox/rox/central/risk/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/signatureintegration"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type mockSignatureIntegrationGetter struct {
	integrations map[string]*storage.SignatureIntegration
}

func (m *mockSignatureIntegrationGetter) GetSignatureIntegration(ctx context.Context, id string) (*storage.SignatureIntegration, bool, error) {
	integration, found := m.integrations[id]
	return integration, found, nil
}

func TestSignatureIntegrationInjectionV2(t *testing.T) {
	suite.Run(t, new(SignatureIntegrationV2TestSuite))
}

type SignatureIntegrationV2TestSuite struct {
	suite.Suite

	ctx        context.Context
	mockCtrl   *gomock.Controller
	mockStore  *mockStore.MockStore
	mockRisk   *mockRisks.MockDataStore
	datastore  DataStore
	mockGetter *mockSignatureIntegrationGetter
}

func (s *SignatureIntegrationV2TestSuite) SetupTest() {
	s.ctx = sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Image),
		),
	)

	s.mockCtrl = gomock.NewController(s.T())
	s.mockStore = mockStore.NewMockStore(s.mockCtrl)
	s.mockRisk = mockRisks.NewMockDataStore(s.mockCtrl)

	// Mock the initializeRankers call that happens in a goroutine
	s.mockStore.EXPECT().GetImagesRiskView(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

	s.datastore = NewWithPostgres(s.mockStore, s.mockRisk, ranking.NewRanker(), ranking.NewRanker())

	s.mockGetter = &mockSignatureIntegrationGetter{
		integrations: map[string]*storage.SignatureIntegration{
			"integration-1": {
				Id:   "integration-1",
				Name: "Test Integration 1",
			},
			"integration-2": {
				Id:   "integration-2",
				Name: "Test Integration 2",
			},
		},
	}
}

func (s *SignatureIntegrationV2TestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *SignatureIntegrationV2TestSuite) TestInjectSignatureIntegrationName_WithoutGetter() {
	// Test that method gracefully handles case where getter is not set
	image := s.createTestImageWithSignatures()

	// Don't set the getter function - should log warning and not crash
	result := []*storage.ImageV2{image}

	// Call the datastore method that internally calls injectSignatureIntegrationName
	s.mockStore.EXPECT().WalkByQuery(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, q interface{}, fn func(*storage.ImageV2) error) error {
			return fn(image)
		},
	)

	err := s.datastore.WalkByQuery(s.ctx, nil, func(img *storage.ImageV2) error {
		// Verify that verifier names are still empty since getter wasn't set
		for _, result := range img.GetSignatureVerificationData().GetResults() {
			s.Empty(result.GetVerifierName(), "VerifierName should be empty when getter is not set")
		}
		return nil
	})
	s.NoError(err)

	// Verify original verifier names are still empty
	for _, result := range result[0].GetSignatureVerificationData().GetResults() {
		s.Empty(result.GetVerifierName(), "VerifierName should remain empty without getter")
	}
}

func (s *SignatureIntegrationV2TestSuite) TestInjectSignatureIntegrationName_WithGetter() {
	// Set the getter function
	s.datastore.SetSignatureIntegrationGetterFunc(func() signatureintegration.Getter {
		return s.mockGetter
	})

	image := s.createTestImageWithSignatures()

	s.mockStore.EXPECT().WalkByQuery(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, q interface{}, fn func(*storage.ImageV2) error) error {
			return fn(image)
		},
	)

	err := s.datastore.WalkByQuery(s.ctx, nil, func(img *storage.ImageV2) error {
		results := img.GetSignatureVerificationData().GetResults()
		s.Require().Len(results, 2, "Should have 2 signature verification results")

		// Verify that verifier names were injected correctly
		s.Equal("Test Integration 1", results[0].GetVerifierName(), "First verifier name should be injected")
		s.Equal("Test Integration 2", results[1].GetVerifierName(), "Second verifier name should be injected")
		return nil
	})
	s.NoError(err)
}

func (s *SignatureIntegrationV2TestSuite) TestInjectSignatureIntegrationName_UnknownIntegration() {
	// Set the getter function
	s.datastore.SetSignatureIntegrationGetterFunc(func() signatureintegration.Getter {
		return s.mockGetter
	})

	image := s.createTestImageWithUnknownSignature()

	s.mockStore.EXPECT().WalkByQuery(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, q interface{}, fn func(*storage.ImageV2) error) error {
			return fn(image)
		},
	)

	err := s.datastore.WalkByQuery(s.ctx, nil, func(img *storage.ImageV2) error {
		results := img.GetSignatureVerificationData().GetResults()
		s.Require().Len(results, 1, "Should have 1 signature verification result")

		// Verify that unknown integration ID results in empty verifier name
		s.Empty(results[0].GetVerifierName(), "VerifierName should be empty for unknown integration")
		return nil
	})
	s.NoError(err)
}

func (s *SignatureIntegrationV2TestSuite) TestInjectSignatureIntegrationName_EmptyVerifierId() {
	// Set the getter function
	s.datastore.SetSignatureIntegrationGetterFunc(func() signatureintegration.Getter {
		return s.mockGetter
	})

	image := s.createTestImageWithEmptyVerifierId()

	s.mockStore.EXPECT().WalkByQuery(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, q interface{}, fn func(*storage.ImageV2) error) error {
			return fn(image)
		},
	)

	err := s.datastore.WalkByQuery(s.ctx, nil, func(img *storage.ImageV2) error {
		results := img.GetSignatureVerificationData().GetResults()
		s.Require().Len(results, 1, "Should have 1 signature verification result")

		// Verify that empty verifier ID results in empty verifier name
		s.Empty(results[0].GetVerifierName(), "VerifierName should be empty for empty verifier ID")
		return nil
	})
	s.NoError(err)
}

func (s *SignatureIntegrationV2TestSuite) TestInjectSignatureIntegrationName_GetImageMetadata() {
	// Test injection through GetImageMetadata method
	s.datastore.SetSignatureIntegrationGetterFunc(func() signatureintegration.Getter {
		return s.mockGetter
	})

	image := s.createTestImageWithSignatures()
	imageID := "test-image-id"

	s.mockStore.EXPECT().GetImageMetadata(gomock.Any(), imageID).Return(image, true, nil)

	result, found, err := s.datastore.GetImageMetadata(s.ctx, imageID)
	s.NoError(err)
	s.True(found)
	s.NotNil(result)

	// Verify verifier names were injected
	results := result.GetSignatureVerificationData().GetResults()
	s.Require().Len(results, 2, "Should have 2 signature verification results")
	s.Equal("Test Integration 1", results[0].GetVerifierName())
	s.Equal("Test Integration 2", results[1].GetVerifierName())
}

func (s *SignatureIntegrationV2TestSuite) TestInjectSignatureIntegrationName_GetManyImageMetadata() {
	// Test injection through GetManyImageMetadata method
	s.datastore.SetSignatureIntegrationGetterFunc(func() signatureintegration.Getter {
		return s.mockGetter
	})

	image1 := s.createTestImageWithSignatures()
	image2 := s.createTestImageWithSignatures()
	imageIDs := []string{"image1", "image2"}

	s.mockStore.EXPECT().GetManyImageMetadata(gomock.Any(), imageIDs).Return([]*storage.ImageV2{image1, image2}, nil)

	results, err := s.datastore.GetManyImageMetadata(s.ctx, imageIDs)
	s.NoError(err)
	s.Len(results, 2)

	// Verify both images have verifier names injected
	for _, img := range results {
		verificationResults := img.GetSignatureVerificationData().GetResults()
		s.Require().Len(verificationResults, 2, "Should have 2 signature verification results")
		s.Equal("Test Integration 1", verificationResults[0].GetVerifierName())
		s.Equal("Test Integration 2", verificationResults[1].GetVerifierName())
	}
}

func (s *SignatureIntegrationV2TestSuite) TestSetSignatureIntegrationGetterFunc_ThreadSafety() {
	// Test that the setter is thread-safe
	done := make(chan bool, 2)

	// Simulate concurrent setter calls
	go func() {
		s.datastore.SetSignatureIntegrationGetterFunc(func() signatureintegration.Getter {
			return s.mockGetter
		})
		done <- true
	}()

	go func() {
		s.datastore.SetSignatureIntegrationGetterFunc(func() signatureintegration.Getter {
			return &mockSignatureIntegrationGetter{integrations: make(map[string]*storage.SignatureIntegration)}
		})
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Verify the datastore is still functional
	image := s.createTestImageWithSignatures()
	s.mockStore.EXPECT().GetImageMetadata(gomock.Any(), "test").Return(image, true, nil)

	_, found, err := s.datastore.GetImageMetadata(s.ctx, "test")
	s.NoError(err)
	s.True(found)
}

// Helper methods

func (s *SignatureIntegrationV2TestSuite) createTestImageWithSignatures() *storage.ImageV2 {
	return &storage.ImageV2{
		Id: "test-image-id",
		SignatureVerificationData: &storage.ImageSignatureVerificationData{
			Results: []*storage.ImageSignatureVerificationResult{
				{
					VerifierId:   "integration-1",
					Status:       storage.ImageSignatureVerificationResult_VERIFIED,
					Description:  "Successfully verified with integration 1",
					VerifierName: "", // Should be populated by injection
				},
				{
					VerifierId:   "integration-2",
					Status:       storage.ImageSignatureVerificationResult_VERIFIED,
					Description:  "Successfully verified with integration 2",
					VerifierName: "", // Should be populated by injection
				},
			},
		},
	}
}

func (s *SignatureIntegrationV2TestSuite) createTestImageWithUnknownSignature() *storage.ImageV2 {
	return &storage.ImageV2{
		Id: "test-image-id",
		SignatureVerificationData: &storage.ImageSignatureVerificationData{
			Results: []*storage.ImageSignatureVerificationResult{
				{
					VerifierId:   "unknown-integration",
					Status:       storage.ImageSignatureVerificationResult_FAILED_VERIFICATION,
					Description:  "Unknown integration",
					VerifierName: "", // Should remain empty
				},
			},
		},
	}
}

func (s *SignatureIntegrationV2TestSuite) createTestImageWithEmptyVerifierId() *storage.ImageV2 {
	return &storage.ImageV2{
		Id: "test-image-id",
		SignatureVerificationData: &storage.ImageSignatureVerificationData{
			Results: []*storage.ImageSignatureVerificationResult{
				{
					VerifierId:   "", // Empty verifier ID
					Status:       storage.ImageSignatureVerificationResult_FAILED_VERIFICATION,
					Description:  "Empty verifier ID",
					VerifierName: "", // Should remain empty
				},
			},
		},
	}
}
