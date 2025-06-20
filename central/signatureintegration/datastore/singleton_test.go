package datastore

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/signatureintegration/store"
	mockSIStore "github.com/stackrox/rox/central/signatureintegration/store/mocks"
	"github.com/stackrox/rox/generated/storage"
	"go.uber.org/mock/gomock"
)

func TestInitializeIntegrations(t *testing.T) {
	t.Run("should not add default integrations when any already exist", func(t *testing.T) {
		s := mockSIStore.NewMockSignatureIntegrationStore(gomock.NewController(t))
		s.EXPECT().Walk(gomock.Any(), gomock.Any()).DoAndReturn(
			func(_ context.Context, fn func(*storage.SignatureIntegration) error) error {
				return fn(&storage.SignatureIntegration{Id: "existing-integration"})
			})

		initializeIntegrations(s)
	})

	t.Run("should add default integrations when none exist", func(t *testing.T) {
		s := mockSIStore.NewMockSignatureIntegrationStore(gomock.NewController(t))
		s.EXPECT().Walk(gomock.Any(), gomock.Any()).Return(nil)
		s.EXPECT().UpsertMany(gomock.Any(), store.DefaultSignatureIntegrations).Return(nil)

		initializeIntegrations(s)
	})
}
