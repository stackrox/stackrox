package registrymirror

import (
	"maps"
	"slices"
	"strings"
	"time"

	configV1 "github.com/openshift/api/config/v1"
	operatorV1Alpha1 "github.com/openshift/api/operator/v1alpha1"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"k8s.io/apimachinery/pkg/types"
)

var log = logging.LoggerForModule()

// Store defines an interface for interacting with registry mirrors.
//
//go:generate mockgen-wrapper
type Store interface {
	Cleanup()

	UpsertImageContentSourcePolicy(icsp *operatorV1Alpha1.ImageContentSourcePolicy) error
	DeleteImageContentSourcePolicy(uid types.UID) error

	UpsertImageDigestMirrorSet(idms *configV1.ImageDigestMirrorSet) error
	DeleteImageDigestMirrorSet(uid types.UID) error

	UpsertImageTagMirrorSet(itms *configV1.ImageTagMirrorSet) error
	DeleteImageTagMirrorSet(uid types.UID) error

	// PullSources will return image references in the order they should be attempted when pulling the image.
	// The returned slice may (or may not) include the source image based on the registries config.
	// Source image must include the registry hostname for mirrors to be matched, for example use:
	// "quay.io/stackrox-io/main:latest" instead of "stackrox-io/main:latest".
	PullSources(srcImage string) ([]string, error)
}

// registryMirror represents a source registry with its mirrors.
type registryMirror struct {
	source          string
	mirrors         []string
	blocked         bool // NeverContactSource
	mirrorByDigest  bool // mirrors apply only to digest-based pulls
	mirrorByTagOnly bool // mirrors apply only to tag-based pulls
}

// FileStore stores registry mirror configuration in memory, built from
// OpenShift ICSP/IDMS/ITMS resources.
//
// This replaces the previous implementation that wrote a TOML registries.conf
// file and used containers/image/v5/pkg/sysregistriesv2 to parse it back,
// which pulled in containers/storage (9 packages), containers/image (9),
// openshift/runtime-utils, and logrus — 20+ transitive dependencies for
// what amounts to prefix matching on image names.
type FileStore struct {
	// holds the mirror sets used for managing registry mirrors.
	icspRules   map[types.UID]*operatorV1Alpha1.ImageContentSourcePolicy
	idmsRules   map[types.UID]*configV1.ImageDigestMirrorSet
	itmsRules   map[types.UID]*configV1.ImageTagMirrorSet
	ruleRWMutex sync.RWMutex

	// compiled is the resolved list of registry mirrors, rebuilt on every rule change.
	compiled    []registryMirror
	compiledMtx sync.RWMutex

	updateDelay  time.Duration
	cancelUpdate concurrency.Signal
}

// fileStoreOption is used to provide functional options to the fileStore.
type fileStoreOption func(*FileStore)

// WithConfigPath is retained for API compatibility but no longer used.
// Mirror configuration is resolved in-memory.
func WithConfigPath(_ string) fileStoreOption {
	return func(_ *FileStore) {}
}

// WithDelay sets a delay before the compiled config is rebuilt.
func WithDelay(delay time.Duration) fileStoreOption {
	return func(s *FileStore) { s.updateDelay = delay }
}

// NewFileStore creates a new FileStore.
func NewFileStore(opts ...fileStoreOption) *FileStore {
	s := &FileStore{
		updateDelay: 100 * time.Millisecond,
		icspRules:   make(map[types.UID]*operatorV1Alpha1.ImageContentSourcePolicy),
		idmsRules:   make(map[types.UID]*configV1.ImageDigestMirrorSet),
		itmsRules:   make(map[types.UID]*configV1.ImageTagMirrorSet),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Cleanup resets the store.
func (s *FileStore) Cleanup() {
	s.ruleRWMutex.Lock()
	defer s.ruleRWMutex.Unlock()

	s.icspRules = make(map[types.UID]*operatorV1Alpha1.ImageContentSourcePolicy)
	s.idmsRules = make(map[types.UID]*configV1.ImageDigestMirrorSet)
	s.itmsRules = make(map[types.UID]*configV1.ImageTagMirrorSet)

	s.cancelUpdate.Signal()

	s.compiledMtx.Lock()
	s.compiled = nil
	s.compiledMtx.Unlock()
}

func (s *FileStore) rebuildConfig() {
	s.ruleRWMutex.RLock()
	icspRules := slices.Collect(maps.Values(s.icspRules))
	idmsRules := slices.Collect(maps.Values(s.idmsRules))
	itmsRules := slices.Collect(maps.Values(s.itmsRules))
	s.ruleRWMutex.RUnlock()

	var mirrors []registryMirror

	// ICSP rules: digest-only mirrors.
	for _, icsp := range icspRules {
		for _, rdm := range icsp.Spec.RepositoryDigestMirrors {
			m := registryMirror{
				source:         rdm.Source,
				mirrorByDigest: true,
			}
			for _, mirror := range rdm.Mirrors {
				m.mirrors = append(m.mirrors, mirror)
			}
			mirrors = mergeOrAppend(mirrors, m)
		}
	}

	// IDMS rules: digest-only mirrors.
	for _, idms := range idmsRules {
		for _, idm := range idms.Spec.ImageDigestMirrors {
			m := registryMirror{
				source:         idm.Source,
				blocked:        idm.MirrorSourcePolicy == configV1.NeverContactSource,
				mirrorByDigest: true,
			}
			for _, mirror := range idm.Mirrors {
				m.mirrors = append(m.mirrors, string(mirror))
			}
			mirrors = mergeOrAppend(mirrors, m)
		}
	}

	// ITMS rules: tag-only mirrors.
	for _, itms := range itmsRules {
		for _, itm := range itms.Spec.ImageTagMirrors {
			m := registryMirror{
				source:          itm.Source,
				blocked:         itm.MirrorSourcePolicy == configV1.NeverContactSource,
				mirrorByTagOnly: true,
			}
			for _, mirror := range itm.Mirrors {
				m.mirrors = append(m.mirrors, string(mirror))
			}
			mirrors = mergeOrAppend(mirrors, m)
		}
	}

	s.compiledMtx.Lock()
	s.compiled = mirrors
	s.compiledMtx.Unlock()

	log.Debugf("Rebuilt registry mirror config: %d entries from %d ICSP, %d IDMS, %d ITMS rules",
		len(mirrors), len(icspRules), len(idmsRules), len(itmsRules))
}

// mergeOrAppend adds a mirror entry, merging with an existing entry for the same source.
func mergeOrAppend(mirrors []registryMirror, m registryMirror) []registryMirror {
	for i := range mirrors {
		if mirrors[i].source == m.source {
			mirrors[i].mirrors = append(mirrors[i].mirrors, m.mirrors...)
			if m.blocked {
				mirrors[i].blocked = true
			}
			return mirrors
		}
	}
	return append(mirrors, m)
}

func (s *FileStore) updateConfig() error {
	if s.updateDelay == 0 {
		s.rebuildConfig()
		return nil
	}

	return s.updateConfigDelayed()
}

func (s *FileStore) updateConfigDelayed() error {
	s.cancelUpdate.Signal()
	s.cancelUpdate.Reset()

	concurrency.AfterFunc(s.updateDelay, func() {
		s.rebuildConfig()
	}, &s.cancelUpdate)

	return nil
}

// PullSources — refer to Store interface for details.
func (s *FileStore) PullSources(srcImage string) ([]string, error) {
	s.compiledMtx.RLock()
	compiled := s.compiled
	s.compiledMtx.RUnlock()

	if len(compiled) == 0 {
		return nil, nil
	}

	// Find the registry with the longest matching prefix.
	var best *registryMirror
	for i := range compiled {
		prefix := compiled[i].source
		if !strings.HasPrefix(srcImage, prefix) {
			continue
		}
		// Prefix must match at a boundary (exact, or followed by / : @).
		if len(srcImage) > len(prefix) {
			next := srcImage[len(prefix)]
			if next != '/' && next != ':' && next != '@' {
				continue
			}
		}
		if best == nil || len(prefix) > len(best.source) {
			best = &compiled[i]
		}
	}

	if best == nil {
		return nil, nil
	}

	// Determine if this is a digest-based or tag-based pull.
	isDigest := strings.Contains(srcImage, "@sha256:")

	// Check if mirrors apply to this pull type.
	mirrorsApply := true
	if best.mirrorByDigest && !isDigest {
		mirrorsApply = false
	}
	if best.mirrorByTagOnly && isDigest {
		mirrorsApply = false
	}

	var srcs []string
	if mirrorsApply {
		for _, mirror := range best.mirrors {
			// Replace the source prefix with the mirror location.
			mirrored := mirror + srcImage[len(best.source):]
			srcs = append(srcs, mirrored)
		}
	}

	if !best.blocked {
		srcs = append(srcs, srcImage)
	}

	if len(srcs) == 0 {
		return nil, nil
	}

	return srcs, nil
}

// UpsertImageContentSourcePolicy will store a new/updated ImageContentSourcePolicy.
func (s *FileStore) UpsertImageContentSourcePolicy(icsp *operatorV1Alpha1.ImageContentSourcePolicy) error {
	s.ruleRWMutex.Lock()
	s.icspRules[icsp.GetUID()] = icsp
	s.ruleRWMutex.Unlock()
	return s.updateConfig()
}

// DeleteImageContentSourcePolicy will delete an ImageContentSourcePolicy from the store if it exists.
func (s *FileStore) DeleteImageContentSourcePolicy(uid types.UID) error {
	s.ruleRWMutex.Lock()
	_, ok := s.icspRules[uid]
	if ok {
		delete(s.icspRules, uid)
	}
	s.ruleRWMutex.Unlock()
	if !ok {
		return nil
	}
	return s.updateConfig()
}

// UpsertImageDigestMirrorSet will store a new/updated ImageDigestMirrorSet.
func (s *FileStore) UpsertImageDigestMirrorSet(idms *configV1.ImageDigestMirrorSet) error {
	s.ruleRWMutex.Lock()
	s.idmsRules[idms.GetUID()] = idms
	s.ruleRWMutex.Unlock()
	return s.updateConfig()
}

// DeleteImageDigestMirrorSet will delete an ImageDigestMirrorSet from the store if it exists.
func (s *FileStore) DeleteImageDigestMirrorSet(uid types.UID) error {
	s.ruleRWMutex.Lock()
	_, ok := s.idmsRules[uid]
	if ok {
		delete(s.idmsRules, uid)
	}
	s.ruleRWMutex.Unlock()
	if !ok {
		return nil
	}
	return s.updateConfig()
}

// UpsertImageTagMirrorSet will store a new/updated ImageTagMirrorSet.
func (s *FileStore) UpsertImageTagMirrorSet(itms *configV1.ImageTagMirrorSet) error {
	s.ruleRWMutex.Lock()
	s.itmsRules[itms.GetUID()] = itms
	s.ruleRWMutex.Unlock()
	return s.updateConfig()
}

// DeleteImageTagMirrorSet will delete an ImageTagMirrorSet from the store if it exists.
func (s *FileStore) DeleteImageTagMirrorSet(uid types.UID) error {
	s.ruleRWMutex.Lock()
	_, ok := s.itmsRules[uid]
	if ok {
		delete(s.itmsRules, uid)
	}
	s.ruleRWMutex.Unlock()
	if !ok {
		return nil
	}
	return s.updateConfig()
}
