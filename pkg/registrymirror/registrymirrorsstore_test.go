package registrymirror

import (
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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	if err != nil {
		t.Error(err)
		return false
	}

	return strings.Contains(string(b), text)
}

func TestUpsertDelete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registries.conf")

	s := NewFileStore(WithConfigPath(path), WithDelay(0))
	assert.Len(t, s.icspRules, 0)
	assert.NoFileExists(t, path)

	t.Run("ICSP", func(t *testing.T) {
		source := icspA.Spec.RepositoryDigestMirrors[0].Source
		s.UpsertImageContentSourcePolicy(icspA)
		assert.Len(t, s.icspRules, 1)
		assert.True(t, fileContains(t, path, source), "config missing registry")

		s.DeleteImageContentSourcePolicy(icspA.UID)
		assert.Len(t, s.icspRules, 0)
		assert.False(t, fileContains(t, path, source), "config has registry but shouldn't")
	})

	t.Run("IDMS", func(t *testing.T) {
		source := idmsA.Spec.ImageDigestMirrors[0].Source
		s.UpsertImageDigestMirrorSet(idmsA)
		assert.Len(t, s.idmsRules, 1)
		assert.True(t, fileContains(t, path, source), "config missing registry")

		s.DeleteImageDigestMirrorSet(idmsA.UID)
		assert.Len(t, s.idmsRules, 0)
		assert.False(t, fileContains(t, path, source), "config has registry but shouldn't")
	})

	t.Run("ITMS", func(t *testing.T) {
		source := itmsA.Spec.ImageTagMirrors[0].Source
		s.UpsertImageTagMirrorSet(itmsA)
		assert.Len(t, s.itmsRules, 1)
		assert.True(t, fileContains(t, path, source), "config missing registry")

		s.DeleteImageTagMirrorSet(itmsA.UID)
		assert.Len(t, s.itmsRules, 0)
		assert.False(t, fileContains(t, path, source), "config has registry but shouldn't")
	})
}

func TestDelayedUpdate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registries.conf")

	s := NewFileStore(WithConfigPath(path), WithDelay(200*time.Millisecond))
	assert.Len(t, s.icspRules, 0)
	assert.NoFileExists(t, path)

	s.UpsertImageContentSourcePolicy(icspA)
	assert.Len(t, s.icspRules, 1)
	assert.NoFileExists(t, path)

	time.Sleep(300 * time.Millisecond)
	assert.True(t, fileContains(t, path, icspA.Spec.RepositoryDigestMirrors[0].Source), "config missing registry")
}

func TestDataRaceAtCleanup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "registries.conf")

	s := NewFileStore(WithConfigPath(path), WithDelay(30*time.Millisecond))

	wg := sync.WaitGroup{}
	doneSignal := concurrency.NewSignal()
	// Spawning two goroutines to trigger data race in updateConfigDelayed
	// (comment out s.timerMutex.Lock/Unlock to see data race trigger)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-doneSignal.Done():
				return
			default:
				s.UpsertImageDigestMirrorSet(idmsA)
				time.Sleep(50 * time.Millisecond)
				s.DeleteImageDigestMirrorSet(idmsA.UID)
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
				s.UpsertImageTagMirrorSet(itmsA)
				time.Sleep(50 * time.Millisecond)
				s.DeleteImageTagMirrorSet(itmsA.UID)
			}
		}
	}()
	time.Sleep(100 * time.Millisecond)
	s.Cleanup()
	doneSignal.Signal()
	wg.Wait()
}

func TestPullSources(t *testing.T) {
	// TODO
}
