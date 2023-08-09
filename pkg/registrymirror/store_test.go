package registrymirror

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	configV1 "github.com/openshift/api/config/v1"
	operatorV1Alpha1 "github.com/openshift/api/operator/v1alpha1"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	icspA = &operatorV1Alpha1.ImageContentSourcePolicy{
		ObjectMeta: v1.ObjectMeta{Name: "icspA", UID: "UIDicspA"},
		Spec: operatorV1Alpha1.ImageContentSourcePolicySpec{
			RepositoryDigestMirrors: []operatorV1Alpha1.RepositoryDigestMirrors{
				{Source: "icsp.registry.com", Mirrors: []string{"icsp.mirror1.com", "icsp.mirror2.com"}},
			},
		},
	}

	idmsA = &configV1.ImageDigestMirrorSet{
		ObjectMeta: v1.ObjectMeta{Name: "idmsA", UID: "UIDidmsA"},
		Spec: configV1.ImageDigestMirrorSetSpec{
			ImageDigestMirrors: []configV1.ImageDigestMirrors{
				{Source: "idms.registry.com", Mirrors: []configV1.ImageMirror{"idms.mirror1.com", "idms.mirror2.com"}},
			},
		},
	}

	itmsA = &configV1.ImageTagMirrorSet{
		ObjectMeta: v1.ObjectMeta{Name: "itmsA", UID: "UIDitmsA"},
		Spec: configV1.ImageTagMirrorSetSpec{
			ImageTagMirrors: []configV1.ImageTagMirrors{
				{Source: "itms.registry.com", Mirrors: []configV1.ImageMirror{"itms.mirror1.com", "itms.mirror2.com"}},
			},
		},
	}
)

func fileContains(t *testing.T, path, text string) bool {
	b, err := os.ReadFile(path)
	require.NoError(t, err)

	return strings.Contains(string(b), text)
}

func TestUpsertDelete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registries.conf")

	s := NewFileStore(WithConfigPath(path), WithDelay(0))
	require.Len(t, s.icspRules, 0)
	require.NoFileExists(t, path)

	t.Run("config should not be updated when no resource deleted", func(t *testing.T) {
		uid := types.UID("fake")
		err := s.DeleteImageContentSourcePolicy(uid)
		assert.NoError(t, err)
		require.NoFileExists(t, path)

		err = s.DeleteImageDigestMirrorSet(uid)
		assert.NoError(t, err)
		require.NoFileExists(t, path)

		err = s.DeleteImageTagMirrorSet(uid)
		assert.NoError(t, err)
		require.NoFileExists(t, path)
	})

	t.Run("ICSP", func(t *testing.T) {
		source := icspA.Spec.RepositoryDigestMirrors[0].Source
		err := s.UpsertImageContentSourcePolicy(icspA)
		assert.NoError(t, err)
		assert.Len(t, s.icspRules, 1)
		assert.True(t, fileContains(t, path, source), "config missing registry")

		err = s.DeleteImageContentSourcePolicy(icspA.UID)
		assert.NoError(t, err)
		assert.Len(t, s.icspRules, 0)
		assert.False(t, fileContains(t, path, source), "config has registry but shouldn't")
	})

	t.Run("IDMS", func(t *testing.T) {
		source := idmsA.Spec.ImageDigestMirrors[0].Source
		err := s.UpsertImageDigestMirrorSet(idmsA)
		assert.NoError(t, err)
		assert.Len(t, s.idmsRules, 1)
		assert.True(t, fileContains(t, path, source), "config missing registry")

		err = s.DeleteImageDigestMirrorSet(idmsA.UID)
		assert.NoError(t, err)
		assert.Len(t, s.idmsRules, 0)
		assert.False(t, fileContains(t, path, source), "config has registry but shouldn't")
	})

	t.Run("ITMS", func(t *testing.T) {
		source := itmsA.Spec.ImageTagMirrors[0].Source
		err := s.UpsertImageTagMirrorSet(itmsA)
		assert.NoError(t, err)
		assert.Len(t, s.itmsRules, 1)
		assert.True(t, fileContains(t, path, source), "config missing registry")

		err = s.DeleteImageTagMirrorSet(itmsA.UID)
		assert.NoError(t, err)
		assert.Len(t, s.itmsRules, 0)
		assert.False(t, fileContains(t, path, source), "config has registry but shouldn't")
	})
}

func TestDelayedUpdate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registries.conf")

	s := NewFileStore(WithConfigPath(path), WithDelay(200*time.Millisecond))
	assert.Len(t, s.icspRules, 0)
	assert.NoFileExists(t, path)

	err := s.UpsertImageContentSourcePolicy(icspA)
	assert.NoError(t, err)
	assert.Len(t, s.icspRules, 1)
	assert.NoFileExists(t, path)

	waitFor := 1 * time.Second
	checkEvery := 250 * time.Millisecond
	conditionFn := func() bool { return fileContains(t, path, icspA.Spec.RepositoryDigestMirrors[0].Source) }
	assert.Eventually(t, conditionFn, waitFor, checkEvery, "config missing registry")
}

func TestDataRaceAtCleanup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registries.conf")
	s := NewFileStore(WithConfigPath(path), WithDelay(30*time.Millisecond))

	wg := sync.WaitGroup{}
	doneSignal := concurrency.NewSignal()
	// Spawning two goroutines to attempt to trigger data race in updateConfigDelayed
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-doneSignal.Done():
				return
			default:
				_ = s.UpsertImageDigestMirrorSet(idmsA)
				time.Sleep(50 * time.Millisecond)
				_ = s.DeleteImageDigestMirrorSet(idmsA.UID)
			}
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-doneSignal.Done():
				return
			default:
				_ = s.UpsertImageTagMirrorSet(itmsA)
				time.Sleep(50 * time.Millisecond)
				_ = s.DeleteImageTagMirrorSet(itmsA.UID)
			}
		}
	}()
	time.Sleep(100 * time.Millisecond)
	s.Cleanup()
	doneSignal.Signal()
	wg.Wait()
}

func TestCleanup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registries.conf")

	s := NewFileStore(WithConfigPath(path), WithDelay(0))

	_ = s.UpsertImageDigestMirrorSet(idmsA)
	_ = s.UpsertImageTagMirrorSet(itmsA)

	require.FileExists(t, path)
	assert.Len(t, s.icspRules, 0)
	assert.Len(t, s.idmsRules, 1)
	assert.Len(t, s.itmsRules, 1)

	s.Cleanup()

	require.NoFileExists(t, path)
	assert.Len(t, s.icspRules, 0)
	assert.Len(t, s.idmsRules, 0)
	assert.Len(t, s.itmsRules, 0)
}

func TestPullSources(t *testing.T) {
	fileName := "registries.conf"
	digest := "@sha256:0000000000000000000000000000000000000000000000000000000000000000"
	tag := ":latest"

	icspfmtStr := "icsp.registry.com/repo/path%s"
	icspImageWithDigest := fmt.Sprintf(icspfmtStr, digest)
	icspImageWithTag := fmt.Sprintf(icspfmtStr, tag)

	idmsfmtStr := "idms.registry.com/repo/path%s"
	idmsImageWithDigest := fmt.Sprintf(idmsfmtStr, digest)
	idmsImageWithTag := fmt.Sprintf(idmsfmtStr, tag)

	itmsfmtStr := "itms.registry.com/repo/path%s"
	itmsImageWithDigest := fmt.Sprintf(itmsfmtStr, digest)
	itmsImageWithTag := fmt.Sprintf(itmsfmtStr, tag)

	t.Run("return source image when config does not exist", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), fileName)

		s := NewFileStore(WithConfigPath(path))

		srcs, err := s.PullSources(icspImageWithDigest)
		assert.NoError(t, err)
		require.Len(t, srcs, 1)
		assert.Equal(t, icspImageWithDigest, srcs[0])
	})

	t.Run("return source image when config is empty", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), fileName)

		_, err := os.Create(path)
		require.NoError(t, err)

		s := NewFileStore(WithConfigPath(path))

		srcs, err := s.PullSources(icspImageWithDigest)
		assert.NoError(t, err)
		require.Len(t, srcs, 1)
		assert.Equal(t, icspImageWithDigest, srcs[0])
	})

	t.Run("return mirrors from ICSP", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), fileName)

		s := NewFileStore(WithConfigPath(path), WithDelay(0))

		err := s.UpsertImageContentSourcePolicy(icspA)
		require.NoError(t, err)

		srcs, err := s.PullSources(icspImageWithDigest)
		assert.NoError(t, err)
		require.Len(t, srcs, 3)
		assert.Contains(t, srcs[0], icspA.Spec.RepositoryDigestMirrors[0].Mirrors[0])
		assert.Contains(t, srcs[1], icspA.Spec.RepositoryDigestMirrors[0].Mirrors[1])
		assert.Contains(t, srcs[2], icspA.Spec.RepositoryDigestMirrors[0].Source)

		srcs, err = s.PullSources(icspImageWithTag)
		assert.NoError(t, err)
		require.Len(t, srcs, 1)
		assert.Contains(t, srcs[0], icspA.Spec.RepositoryDigestMirrors[0].Source)
	})

	t.Run("return mirrors from IDMS", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), fileName)

		s := NewFileStore(WithConfigPath(path), WithDelay(0))

		err := s.UpsertImageDigestMirrorSet(idmsA)
		require.NoError(t, err)

		srcs, err := s.PullSources(idmsImageWithDigest)
		assert.NoError(t, err)
		require.Len(t, srcs, 3)
		assert.Contains(t, srcs[0], idmsA.Spec.ImageDigestMirrors[0].Mirrors[0])
		assert.Contains(t, srcs[1], idmsA.Spec.ImageDigestMirrors[0].Mirrors[1])
		assert.Contains(t, srcs[2], idmsA.Spec.ImageDigestMirrors[0].Source)

		srcs, err = s.PullSources(idmsImageWithTag)
		assert.NoError(t, err)
		assert.Len(t, srcs, 1)
		assert.Contains(t, srcs[0], idmsA.Spec.ImageDigestMirrors[0].Source)

		// do not return source reference if it is blocked
		idmsANoSrc := idmsA.DeepCopy()
		idmsANoSrc.Spec.ImageDigestMirrors[0].MirrorSourcePolicy = "NeverContactSource"

		err = s.UpsertImageDigestMirrorSet(idmsANoSrc)
		require.NoError(t, err)

		srcs, err = s.PullSources(idmsImageWithDigest)
		assert.NoError(t, err)
		require.Len(t, srcs, 2)
		assert.Contains(t, srcs[0], idmsA.Spec.ImageDigestMirrors[0].Mirrors[0])
		assert.Contains(t, srcs[1], idmsA.Spec.ImageDigestMirrors[0].Mirrors[1])
	})

	t.Run("return mirrors from ITMS", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), fileName)

		s := NewFileStore(WithConfigPath(path), WithDelay(0))

		err := s.UpsertImageTagMirrorSet(itmsA)
		require.NoError(t, err)

		srcs, err := s.PullSources(itmsImageWithDigest)
		assert.NoError(t, err)
		require.Len(t, srcs, 1)
		assert.Contains(t, srcs[0], itmsA.Spec.ImageTagMirrors[0].Source)

		srcs, err = s.PullSources(itmsImageWithTag)
		assert.NoError(t, err)
		require.Len(t, srcs, 3)
		assert.Contains(t, srcs[0], itmsA.Spec.ImageTagMirrors[0].Mirrors[0])
		assert.Contains(t, srcs[1], itmsA.Spec.ImageTagMirrors[0].Mirrors[1])
		assert.Contains(t, srcs[2], itmsA.Spec.ImageTagMirrors[0].Source)

		// do not return source reference if it is blocked
		itmsANoSrc := itmsA.DeepCopy()
		itmsANoSrc.Spec.ImageTagMirrors[0].MirrorSourcePolicy = "NeverContactSource"

		err = s.UpsertImageTagMirrorSet(itmsANoSrc)
		require.NoError(t, err)

		srcs, err = s.PullSources(itmsImageWithTag)
		assert.NoError(t, err)
		require.Len(t, srcs, 2)
		assert.Contains(t, srcs[0], itmsA.Spec.ImageTagMirrors[0].Mirrors[0])
		assert.Contains(t, srcs[1], itmsA.Spec.ImageTagMirrors[0].Mirrors[1])
	})

	t.Run("error when CRs updated to bad state (sync)", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), fileName)

		s := NewFileStore(WithConfigPath(path), WithDelay(0))

		err := s.UpsertImageContentSourcePolicy(icspA)
		assert.NoError(t, err)

		err = s.UpsertImageDigestMirrorSet(idmsA)
		assert.Error(t, err)
	})

	t.Run("no error when CRs are updated to bad state (async)", func(t *testing.T) {
		// the config update will fail and the failure logged, however because the write
		// is async no error is returned.
		path := filepath.Join(t.TempDir(), fileName)

		s := NewFileStore(WithConfigPath(path), WithDelay(30*time.Millisecond))

		err := s.UpsertImageContentSourcePolicy(icspA)
		assert.NoError(t, err)

		err = s.UpsertImageDigestMirrorSet(idmsA)
		time.Sleep(40 * time.Millisecond)
		assert.NoError(t, err)
	})

	t.Run("unqualified hostname returns source image", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), fileName)

		s := NewFileStore(WithConfigPath(path), WithDelay(0))

		err := s.UpsertImageContentSourcePolicy(icspA)
		assert.NoError(t, err)

		src := "nginx:latest"
		srcs, err := s.PullSources(src)
		require.NoError(t, err)
		require.Len(t, srcs, 1)
		assert.Equal(t, src, srcs[0])
	})
}
