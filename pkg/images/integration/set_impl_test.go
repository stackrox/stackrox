package integration

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/integrationhealth/mocks"
	"github.com/stackrox/rox/pkg/registries"
	registriesMocks "github.com/stackrox/rox/pkg/registries/mocks"
	registriesTypesMocks "github.com/stackrox/rox/pkg/registries/types/mocks"
	"github.com/stackrox/rox/pkg/scanners"
	scannersMocks "github.com/stackrox/rox/pkg/scanners/mocks"
	scannersTypesMocks "github.com/stackrox/rox/pkg/scanners/types/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// TestCategoryRemove ensures that when a category is removed from
// and integration, the underlying sets are updated.
func TestCategoryRemove(t *testing.T) {
	ctrl := gomock.NewController(t)

	registryFactory := registriesMocks.NewMockFactory(ctrl)
	registrySet := registries.NewSet(registryFactory)
	registry := registriesTypesMocks.NewMockImageRegistry(ctrl)
	registryFactory.EXPECT().CreateRegistry(gomock.Any()).AnyTimes().Return(registry, nil)

	scannerFactory := scannersMocks.NewMockFactory(ctrl)
	scannerSet := scanners.NewSet(scannerFactory)
	scanner := scannersTypesMocks.NewMockImageScannerWithDataSource(ctrl)
	scannerFactory.EXPECT().CreateScanner(gomock.Any()).AnyTimes().Return(scanner, nil)

	reporter := mocks.NewMockReporter(ctrl)
	reporter.EXPECT().Register(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	s := setImpl{
		registryFactory: registryFactory,
		registrySet:     registrySet,

		scannerFactory: scannerFactory,
		scannerSet:     scannerSet,
		reporter:       reporter,
	}

	integration := &storage.ImageIntegration{
		Id:   "fake-id",
		Name: "fake-name",
		Type: "fake-type",
		Categories: []storage.ImageIntegrationCategory{
			storage.ImageIntegrationCategory_SCANNER,
			storage.ImageIntegrationCategory_REGISTRY,
		},
	}

	err := s.UpdateImageIntegration(integration)
	require.NoError(t, err)
	assert.False(t, scannerSet.IsEmpty())
	assert.False(t, registrySet.IsEmpty())

	// Remove the Scanner category
	integration.Categories = []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY}
	err = s.UpdateImageIntegration(integration)
	require.NoError(t, err)
	assert.True(t, scannerSet.IsEmpty())
	assert.False(t, registrySet.IsEmpty())

	// Remove the Registry category
	integration.Categories = []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_SCANNER}
	err = s.UpdateImageIntegration(integration)
	require.NoError(t, err)
	assert.False(t, scannerSet.IsEmpty())
	assert.True(t, registrySet.IsEmpty())
}
