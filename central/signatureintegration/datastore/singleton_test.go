package datastore

import (
	"testing"

	mockSIStore "github.com/stackrox/rox/central/signatureintegration/store/mocks"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/signatures"
	"github.com/stackrox/rox/pkg/testutils"
	"go.uber.org/mock/gomock"
)

func TestSetupDefaultRedHatSignatureIntegration(t *testing.T) {
	t.Run("should upsert integration when RedHatImagesSignedPolicy feature is enabled", func(t *testing.T) {
		testutils.MustUpdateFeature(t, features.RedHatImagesSignedPolicy, true)

		s := mockSIStore.NewMockSignatureIntegrationStore(gomock.NewController(t))
		s.EXPECT().Upsert(gomock.Any(), signatures.DefaultRedHatSignatureIntegration).Return(nil)

		setupDefaultRedHatSignatureIntegration(s)
	})

	t.Run("should delete integration when RedHatImagesSignedPolicy feature is disabled", func(t *testing.T) {
		testutils.MustUpdateFeature(t, features.RedHatImagesSignedPolicy, false)

		s := mockSIStore.NewMockSignatureIntegrationStore(gomock.NewController(t))
		s.EXPECT().Delete(gomock.Any(), signatures.DefaultRedHatSignatureIntegration.GetId()).Return(nil)

		setupDefaultRedHatSignatureIntegration(s)
	})

	t.Run("should handle NotFound error gracefully when deleting", func(t *testing.T) {
		testutils.MustUpdateFeature(t, features.RedHatImagesSignedPolicy, false)

		s := mockSIStore.NewMockSignatureIntegrationStore(gomock.NewController(t))
		notFoundErr := errox.NotFound.New("integration not found")
		s.EXPECT().Delete(gomock.Any(), signatures.DefaultRedHatSignatureIntegration.GetId()).Return(notFoundErr)

		setupDefaultRedHatSignatureIntegration(s)
	})
}
