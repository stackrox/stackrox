package datastore

import (
	"testing"

	mockSIStore "github.com/stackrox/rox/central/signatureintegration/store/mocks"
	"github.com/stackrox/rox/pkg/signatures"
	"go.uber.org/mock/gomock"
)

func TestCreateDefaultRedHatSignatureIntegration(t *testing.T) {
	t.Run("should not add default integration when it already exists", func(t *testing.T) {
		s := mockSIStore.NewMockSignatureIntegrationStore(gomock.NewController(t))
		s.EXPECT().Get(gomock.Any(), gomock.Any()).Return(
			signatures.DefaultRedHatSignatureIntegration, true, nil)

		createDefaultRedHatSignatureIntegration(s)
	})

	t.Run("should add default integration when it doesn't exist", func(t *testing.T) {
		s := mockSIStore.NewMockSignatureIntegrationStore(gomock.NewController(t))
		s.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, false, nil)
		s.EXPECT().Upsert(gomock.Any(), signatures.DefaultRedHatSignatureIntegration).Return(nil)

		createDefaultRedHatSignatureIntegration(s)
	})
}
