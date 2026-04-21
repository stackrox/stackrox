package manager

import (
	"context"
	"testing"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/coalescer"
	"github.com/stackrox/rox/pkg/sizeboundedcache"
	"github.com/stackrox/rox/sensor/admission-control/resources"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ManagerImplSuite struct {
	suite.Suite
	mgr     *manager
	nsStore *resources.NamespaceStore
}

func TestManagerImplSuite(t *testing.T) {
	suite.Run(t, new(ManagerImplSuite))
}

func (s *ManagerImplSuite) SetupSuite() {
	cache, err := sizeboundedcache.New(1024*1024, 512*1024, func(key string, value imageCacheEntry) int64 {
		return 1024
	})
	require.NoError(s.T(), err)

	nameCache, err := lru.New[string, string](imageNameCacheSize)
	require.NoError(s.T(), err)

	s.mgr = &manager{
		imageCache:               cache,
		imageNameToImageCacheKey: nameCache,
		imageFetchGroup:          coalescer.New[*storage.Image](),
		imageCacheGen:            newImageGenTracker(),
	}
}

func (s *ManagerImplSuite) SetupTest() {
	s.mgr.imageCache.Purge()
	s.mgr.imageNameToImageCacheKey.Purge()
	s.mgr.imageCacheGen.Clear(s.T())
	s.mgr.clusterLabels.Store(nil)

	depStore := resources.NewDeploymentStore(nil)
	podStore := resources.NewPodStore()
	s.nsStore = resources.NewNamespaceStore(depStore, podStore)
	s.mgr.namespaces = s.nsStore
}

func (s *ManagerImplSuite) addToImageCache(key string) {
	s.mgr.imageCache.Add(key, imageCacheEntry{
		Image:     &storage.Image{Id: key},
		timestamp: time.Now(),
	})
}

func (s *ManagerImplSuite) addNameMapping(name, key string) {
	s.mgr.imageNameToImageCacheKey.Add(name, key)
}

func (s *ManagerImplSuite) assertCached(key string, expected bool, msgAndArgs ...interface{}) {
	_, ok := s.mgr.imageCache.Get(key)
	s.Equal(expected, ok, msgAndArgs...)
}

func (s *ManagerImplSuite) assertNameMapped(name string, expected bool, msgAndArgs ...interface{}) {
	_, ok := s.mgr.imageNameToImageCacheKey.Get(name)
	s.Equal(expected, ok, msgAndArgs...)
}

func (s *ManagerImplSuite) invalidate(keys ...*central.ImageKey) {
	s.mgr.processImageCacheInvalidation(&sensor.AdmCtrlImageCacheInvalidation{
		ImageKeys: keys,
	})
}

// --- Cluster labels ---

func (s *ManagerImplSuite) TestGetClusterLabelsNil() {
	labels, err := s.mgr.GetClusterLabels(context.Background(), "cluster-id")
	s.NoError(err)
	s.Nil(labels)
}

func (s *ManagerImplSuite) TestGetClusterLabelsReturnsStored() {
	clusterLabels := map[string]string{
		"env":    "prod",
		"region": "us-east-1",
	}
	s.mgr.clusterLabels.Store(&clusterLabels)

	labels, err := s.mgr.GetClusterLabels(context.Background(), "cluster-id")
	s.NoError(err)
	s.Equal(map[string]string{
		"env":    "prod",
		"region": "us-east-1",
	}, labels)
}

// --- Namespace labels ---

func (s *ManagerImplSuite) TestGetNamespaceLabelsReturnsStored() {
	s.nsStore.ProcessEvent(central.ResourceAction_CREATE_RESOURCE, &storage.NamespaceMetadata{
		Name: "test-namespace",
		Labels: map[string]string{
			"team": "backend",
			"tier": "app",
		},
	})

	labels, err := s.mgr.GetNamespaceLabels(context.Background(), "cluster-id", "test-namespace")
	s.NoError(err)
	s.Equal(map[string]string{
		"team": "backend",
		"tier": "app",
	}, labels)
}

func (s *ManagerImplSuite) TestGetNamespaceLabelsNonExistent() {
	labels, err := s.mgr.GetNamespaceLabels(context.Background(), "cluster-id", "nonexistent")
	s.NoError(err)
	s.Nil(labels)
}

// --- Image cache invalidation ---

func (s *ManagerImplSuite) TestInvalidateByImageID() {
	s.mgr.state.Store(createTestState(false))
	s.addToImageCache("sha256:abc")
	s.addNameMapping("nginx:latest", "sha256:abc")

	s.invalidate(&central.ImageKey{
		ImageId:       "sha256:abc",
		ImageFullName: "nginx:latest",
	})

	s.assertCached("sha256:abc", false, "image cache entry should be removed")
	s.assertNameMapped("nginx:latest", false, "name-to-key mapping should be removed")
}

func (s *ManagerImplSuite) TestInvalidateByV2IDWhenFlattenEnabled() {
	s.mgr.state.Store(createTestState(true))
	s.addToImageCache("v2-uuid")
	s.addNameMapping("nginx:latest", "v2-uuid")

	s.invalidate(&central.ImageKey{
		ImageId:       "sha256:abc",
		ImageIdV2:     "v2-uuid",
		ImageFullName: "nginx:latest",
	})

	s.assertCached("v2-uuid", false, "image cache entry should be removed by V2 key")
	s.assertCached("sha256:abc", false, "V1 key should not be used when flatten is enabled and V2 key is present")
	s.assertNameMapped("nginx:latest", false, "name-to-key mapping should be removed")
}

func (s *ManagerImplSuite) TestFlattenEnabledV2EmptyFallsBack() {
	s.mgr.state.Store(createTestState(true))
	s.addToImageCache("sha256:abc")

	s.invalidate(&central.ImageKey{
		ImageId:       "sha256:abc",
		ImageIdV2:     "",
		ImageFullName: "nginx:latest",
	})

	s.assertCached("sha256:abc", false, "image cache entry should be removed using V1 key as fallback")
}

func (s *ManagerImplSuite) TestOnlyFullNameRemovesNameMappingOnly() {
	s.mgr.state.Store(createTestState(false))
	s.addToImageCache("sha256:abc")
	s.addNameMapping("nginx:latest", "sha256:abc")

	s.invalidate(&central.ImageKey{
		ImageFullName: "nginx:latest",
	})

	s.assertCached("sha256:abc", true, "image cache entry should NOT be removed when only fullName is provided")
	s.assertNameMapped("nginx:latest", false, "name-to-key mapping should be removed")
}

func (s *ManagerImplSuite) TestOnlyImageIDRemovesCacheEntryOnly() {
	s.mgr.state.Store(createTestState(false))
	s.addToImageCache("sha256:abc")
	s.addNameMapping("nginx:latest", "sha256:abc")

	s.invalidate(&central.ImageKey{
		ImageId: "sha256:abc",
	})

	s.assertCached("sha256:abc", false, "image cache entry should be removed")
	s.assertNameMapped("nginx:latest", true, "name-to-key mapping should NOT be removed when fullName is empty")
}

func (s *ManagerImplSuite) TestMultipleKeysInOneMessage() {
	s.mgr.state.Store(createTestState(false))
	s.addToImageCache("sha256:abc")
	s.addToImageCache("sha256:def")
	s.addNameMapping("nginx:latest", "sha256:abc")
	s.addNameMapping("redis:7", "sha256:def")

	s.invalidate(
		&central.ImageKey{ImageId: "sha256:abc", ImageFullName: "nginx:latest"},
		&central.ImageKey{ImageId: "sha256:def", ImageFullName: "redis:7"},
	)

	s.assertCached("sha256:abc", false)
	s.assertCached("sha256:def", false)
	s.assertNameMapped("nginx:latest", false)
	s.assertNameMapped("redis:7", false)
}

func (s *ManagerImplSuite) TestNilStateDefaultsFlattenToFalse() {
	s.addToImageCache("sha256:abc")

	s.invalidate(&central.ImageKey{
		ImageId:   "sha256:abc",
		ImageIdV2: "v2-uuid",
	})

	s.assertCached("sha256:abc", false, "should use ImageId when state is nil (flatten defaults to false)")
}
