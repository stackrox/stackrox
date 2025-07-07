package datastore

import (
	"context"
	"testing"

	mockSIStore "github.com/stackrox/rox/central/signatureintegration/store/mocks"
	"github.com/stackrox/rox/pkg/signatures"
	"go.uber.org/mock/gomock"
)

func TestCreateDefaultRedHatSignatureIntegration(t *testing.T) {
	ctx := context.Background()

	t.Run("should not add default integration when it already exists", func(t *testing.T) {
		s := mockSIStore.NewMockSignatureIntegrationStore(gomock.NewController(t))

		s.EXPECT().Get(gomock.Any(), signatures.DefaultRedHatSignatureIntegration.GetId()).Return(
			signatures.DefaultRedHatSignatureIntegration, true, nil)

		createDefaultRedHatSignatureIntegration(ctx, s)
	})

	t.Run("should add default integration when it doesn't exist", func(t *testing.T) {
		s := mockSIStore.NewMockSignatureIntegrationStore(gomock.NewController(t))

		s.EXPECT().Get(gomock.Any(), signatures.DefaultRedHatSignatureIntegration.GetId()).Return(
			nil, false, nil)
		s.EXPECT().Upsert(gomock.Any(), signatures.DefaultRedHatSignatureIntegration).Return(nil)

		createDefaultRedHatSignatureIntegration(ctx, s)
	})
}

func TestRemoveDefaultRedHatSignatureIntegration(t *testing.T) {
	ctx := context.Background()

	t.Run("should not delete default integration when it doesn't exist", func(t *testing.T) {
		s := mockSIStore.NewMockSignatureIntegrationStore(gomock.NewController(t))

		s.EXPECT().Get(gomock.Any(), signatures.DefaultRedHatSignatureIntegration.GetId()).Return(
			nil, false, nil)

		removeDefaultRedHatSignatureIntegration(ctx, s)
	})

	t.Run("should delete default integration when it exists", func(t *testing.T) {
		s := mockSIStore.NewMockSignatureIntegrationStore(gomock.NewController(t))

		s.EXPECT().Get(gomock.Any(), signatures.DefaultRedHatSignatureIntegration.GetId()).Return(
			signatures.DefaultRedHatSignatureIntegration, true, nil)
		s.EXPECT().Delete(gomock.Any(), signatures.DefaultRedHatSignatureIntegration.GetId()).Return(nil)

		removeDefaultRedHatSignatureIntegration(ctx, s)
	})
}
