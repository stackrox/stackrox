package reposcan

import (
	"errors"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/registries/types/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// mockRegistrySet implements RegistrySet for testing.
type mockRegistrySet struct {
	all       []types.ImageRegistry
	allUnique []types.ImageRegistry
}

func (m *mockRegistrySet) GetAll() []types.ImageRegistry {
	return m.all
}

func (m *mockRegistrySet) GetAllUnique() []types.ImageRegistry {
	return m.allUnique
}

// mockRegistryStore implements RegistryStore for testing.
type mockRegistryStore struct {
	centralRegistries []types.ImageRegistry
	globalRegistries  []types.ImageRegistry
	globalError       error
}

func (m *mockRegistryStore) GetCentralRegistries(_ *storage.ImageName) []types.ImageRegistry {
	return m.centralRegistries
}

func (m *mockRegistryStore) GetGlobalRegistries(_ *storage.ImageName) ([]types.ImageRegistry, error) {
	if m.globalError != nil {
		return nil, m.globalError
	}
	return m.globalRegistries, nil
}

func TestNewRegistryMatcher_UsesGetAll_WhenDedupeDisabled(t *testing.T) {
	t.Setenv("ROX_DEDUPE_IMAGE_INTEGRATIONS", "false")

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	imgName := &storage.ImageName{
		Registry: "docker.io",
		Remote:   "library/nginx",
		Tag:      "latest",
	}

	// Create mock registries
	matchingReg := mocks.NewMockImageRegistry(ctrl)
	matchingReg.EXPECT().Match(imgName).Return(true)

	nonMatchingReg := mocks.NewMockImageRegistry(ctrl)
	nonMatchingReg.EXPECT().Match(imgName).Return(false)

	set := &mockRegistrySet{
		all: []types.ImageRegistry{nonMatchingReg, matchingReg},
		// allUnique should NOT be called when dedupe is disabled
		allUnique: nil,
	}

	matcher := NewRegistryMatcher(set)
	result := matcher(imgName)

	assert.Equal(t, matchingReg, result, "Should return first matching registry from GetAll()")
}

func TestNewRegistryMatcher_UsesGetAllUnique_WhenDedupeEnabled(t *testing.T) {
	t.Setenv("ROX_DEDUPE_IMAGE_INTEGRATIONS", "true")

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	imgName := &storage.ImageName{
		Registry: "quay.io",
		Remote:   "prometheus/prometheus",
		Tag:      "v2.45.0",
	}

	// Create mock registries
	matchingReg := mocks.NewMockImageRegistry(ctrl)
	matchingReg.EXPECT().Match(imgName).Return(true)

	nonMatchingReg := mocks.NewMockImageRegistry(ctrl)
	nonMatchingReg.EXPECT().Match(imgName).Return(false)

	set := &mockRegistrySet{
		// all should NOT be called when dedupe is enabled
		all:       nil,
		allUnique: []types.ImageRegistry{nonMatchingReg, matchingReg},
	}

	matcher := NewRegistryMatcher(set)
	result := matcher(imgName)

	assert.Equal(t, matchingReg, result, "Should return first matching registry from GetAllUnique()")
}

func TestNewRegistryMatcher_NoMatch(t *testing.T) {
	t.Setenv("ROX_DEDUPE_IMAGE_INTEGRATIONS", "false")

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	imgName := &storage.ImageName{
		Registry: "quay.io",
		Remote:   "prometheus/prometheus",
	}

	nonMatchingReg := mocks.NewMockImageRegistry(ctrl)
	nonMatchingReg.EXPECT().Match(imgName).Return(false)

	set := &mockRegistrySet{
		all: []types.ImageRegistry{nonMatchingReg},
	}

	matcher := NewRegistryMatcher(set)
	result := matcher(imgName)

	assert.Nil(t, result, "Should return nil when no registry matches")
}

func TestNewRegistryMatcher_EmptySet(t *testing.T) {
	t.Setenv("ROX_DEDUPE_IMAGE_INTEGRATIONS", "false")

	imgName := &storage.ImageName{
		Registry: "docker.io",
		Remote:   "library/alpine",
	}

	set := &mockRegistrySet{
		all: []types.ImageRegistry{},
	}

	matcher := NewRegistryMatcher(set)
	result := matcher(imgName)

	assert.Nil(t, result, "Should return nil when registry set is empty")
}

func TestNewRegistryMatcher_ReturnsFirstMatch(t *testing.T) {
	t.Setenv("ROX_DEDUPE_IMAGE_INTEGRATIONS", "false")

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	imgName := &storage.ImageName{
		Registry: "docker.io",
		Remote:   "library/nginx",
	}

	firstMatch := mocks.NewMockImageRegistry(ctrl)
	firstMatch.EXPECT().Match(imgName).Return(true)

	secondMatch := mocks.NewMockImageRegistry(ctrl)
	// Should not be called since first already matched

	set := &mockRegistrySet{
		all: []types.ImageRegistry{firstMatch, secondMatch},
	}

	matcher := NewRegistryMatcher(set)
	result := matcher(imgName)

	assert.Equal(t, firstMatch, result, "Should return first matching registry")
}

func TestNewRegistryMatcherFromStore_CentralRegistries(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	imgName := &storage.ImageName{
		Registry: "docker.io",
		Remote:   "library/nginx",
	}

	centralReg := mocks.NewMockImageRegistry(ctrl)
	centralReg.EXPECT().Match(imgName).Return(true)

	store := &mockRegistryStore{
		centralRegistries: []types.ImageRegistry{centralReg},
	}

	matcher := NewRegistryMatcherFromStore(store)
	result := matcher(imgName)

	assert.Equal(t, centralReg, result, "Should return matching Central registry")
}

func TestNewRegistryMatcherFromStore_FallbackToGlobal(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	imgName := &storage.ImageName{
		Registry: "quay.io",
		Remote:   "prometheus/prometheus",
	}

	centralReg := mocks.NewMockImageRegistry(ctrl)
	centralReg.EXPECT().Match(imgName).Return(false)

	globalReg := mocks.NewMockImageRegistry(ctrl)
	globalReg.EXPECT().Match(imgName).Return(true)

	store := &mockRegistryStore{
		centralRegistries: []types.ImageRegistry{centralReg},
		globalRegistries:  []types.ImageRegistry{globalReg},
	}

	matcher := NewRegistryMatcherFromStore(store)
	result := matcher(imgName)

	assert.Equal(t, globalReg, result, "Should fallback to global registry when Central doesn't match")
}

func TestNewRegistryMatcherFromStore_CentralMatchSkipsGlobal(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	imgName := &storage.ImageName{
		Registry: "docker.io",
		Remote:   "library/nginx",
	}

	centralReg := mocks.NewMockImageRegistry(ctrl)
	centralReg.EXPECT().Match(imgName).Return(true)

	globalReg := mocks.NewMockImageRegistry(ctrl)
	// Should not be called since Central already matched

	store := &mockRegistryStore{
		centralRegistries: []types.ImageRegistry{centralReg},
		globalRegistries:  []types.ImageRegistry{globalReg},
	}

	matcher := NewRegistryMatcherFromStore(store)
	result := matcher(imgName)

	assert.Equal(t, centralReg, result, "Should not check global registries if Central matches")
}

func TestNewRegistryMatcherFromStore_NoCentralRegistries(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	imgName := &storage.ImageName{
		Registry: "quay.io",
		Remote:   "prometheus/prometheus",
	}

	globalReg := mocks.NewMockImageRegistry(ctrl)
	globalReg.EXPECT().Match(imgName).Return(true)

	store := &mockRegistryStore{
		centralRegistries: []types.ImageRegistry{}, // Empty
		globalRegistries:  []types.ImageRegistry{globalReg},
	}

	matcher := NewRegistryMatcherFromStore(store)
	result := matcher(imgName)

	assert.Equal(t, globalReg, result, "Should check global registries when Central list is empty")
}

func TestNewRegistryMatcherFromStore_GlobalError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	imgName := &storage.ImageName{
		Registry: "quay.io",
		Remote:   "prometheus/prometheus",
	}

	centralReg := mocks.NewMockImageRegistry(ctrl)
	centralReg.EXPECT().Match(imgName).Return(false)

	store := &mockRegistryStore{
		centralRegistries: []types.ImageRegistry{centralReg},
		globalError:       errors.New("failed to get global registries"),
	}

	matcher := NewRegistryMatcherFromStore(store)
	result := matcher(imgName)

	assert.Nil(t, result, "Should return nil when global registries return error")
}

func TestNewRegistryMatcherFromStore_NoMatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	imgName := &storage.ImageName{
		Registry: "gcr.io",
		Remote:   "my-project/my-image",
	}

	centralReg := mocks.NewMockImageRegistry(ctrl)
	centralReg.EXPECT().Match(imgName).Return(false)

	globalReg := mocks.NewMockImageRegistry(ctrl)
	globalReg.EXPECT().Match(imgName).Return(false)

	store := &mockRegistryStore{
		centralRegistries: []types.ImageRegistry{centralReg},
		globalRegistries:  []types.ImageRegistry{globalReg},
	}

	matcher := NewRegistryMatcherFromStore(store)
	result := matcher(imgName)

	assert.Nil(t, result, "Should return nil when no registries match")
}

func TestNewRegistryMatcherFromStore_EmptyStore(t *testing.T) {
	imgName := &storage.ImageName{
		Registry: "docker.io",
		Remote:   "library/alpine",
	}

	store := &mockRegistryStore{
		centralRegistries: []types.ImageRegistry{},
		globalRegistries:  []types.ImageRegistry{},
	}

	matcher := NewRegistryMatcherFromStore(store)
	result := matcher(imgName)

	assert.Nil(t, result, "Should return nil when store has no registries")
}

func TestNewRegistryMatcherFromStore_MultipleCentralFirstWins(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	imgName := &storage.ImageName{
		Registry: "docker.io",
		Remote:   "library/nginx",
	}

	firstCentral := mocks.NewMockImageRegistry(ctrl)
	firstCentral.EXPECT().Match(imgName).Return(true)

	secondCentral := mocks.NewMockImageRegistry(ctrl)
	// Should not be called since first already matched

	store := &mockRegistryStore{
		centralRegistries: []types.ImageRegistry{firstCentral, secondCentral},
	}

	matcher := NewRegistryMatcherFromStore(store)
	result := matcher(imgName)

	assert.Equal(t, firstCentral, result, "Should return first matching Central registry")
}

func TestNewRegistryMatcherFromStore_MultipleGlobalFirstWins(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	imgName := &storage.ImageName{
		Registry: "quay.io",
		Remote:   "prometheus/prometheus",
	}

	centralReg := mocks.NewMockImageRegistry(ctrl)
	centralReg.EXPECT().Match(imgName).Return(false)

	firstGlobal := mocks.NewMockImageRegistry(ctrl)
	firstGlobal.EXPECT().Match(imgName).Return(true)

	secondGlobal := mocks.NewMockImageRegistry(ctrl)
	// Should not be called since first already matched

	store := &mockRegistryStore{
		centralRegistries: []types.ImageRegistry{centralReg},
		globalRegistries:  []types.ImageRegistry{firstGlobal, secondGlobal},
	}

	matcher := NewRegistryMatcherFromStore(store)
	result := matcher(imgName)

	assert.Equal(t, firstGlobal, result, "Should return first matching global registry")
}
