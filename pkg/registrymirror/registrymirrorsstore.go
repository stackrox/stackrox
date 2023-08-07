package registrymirror

import (
	"bytes"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/containers/image/v5/pkg/sysregistriesv2"
	ciTypes "github.com/containers/image/v5/types"
	"github.com/docker/distribution/reference"
	configV1 "github.com/openshift/api/config/v1"
	operatorV1Alpha1 "github.com/openshift/api/operator/v1alpha1"
	"github.com/openshift/runtime-utils/pkg/registries"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/maputil"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// default path to store registries config
	defaultRegistriesPath = "/var/cache/stackrox/mirrors/registries.conf"

	// default delay before writing the updated registries config
	defaultDelay = time.Millisecond * 300
)

var (
	log = logging.LoggerForModule()
)

// Store defines an interface for interacting with registry mirrors.
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
	// Source image must be fully qualified (include the registry hostname).
	PullSources(srcImage string) ([]string, error)
}

// FileStore stores/reads the consolidated registries config from a filesystem.
type FileStore struct {
	configPath    string
	systemContext *ciTypes.SystemContext
	updateDelay   time.Duration

	// holds the mirror sets used for managing registry mirrors.
	icspRules   map[types.UID]*operatorV1Alpha1.ImageContentSourcePolicy
	idmsRules   map[types.UID]*configV1.ImageDigestMirrorSet
	itmsRules   map[types.UID]*configV1.ImageTagMirrorSet
	ruleRWMutex sync.RWMutex

	timer      *time.Timer
	timerMutex sync.Mutex
}

var _ Store = (*FileStore)(nil)

type fileStoreOption func(*FileStore)

// WithConfigPath sets the path to read/write the consolidated registries config.
func WithConfigPath(path string) fileStoreOption {
	return func(s *FileStore) { s.configPath = path }
}

// WithDelay sets a delay before the updated config is written to disk.
func WithDelay(delay time.Duration) fileStoreOption {
	return func(s *FileStore) { s.updateDelay = delay }
}

// NewFileStore creates a new FileStore.
func NewFileStore(opts ...fileStoreOption) *FileStore {
	s := &FileStore{
		configPath:  defaultRegistriesPath,
		updateDelay: defaultDelay,
		icspRules:   make(map[types.UID]*operatorV1Alpha1.ImageContentSourcePolicy),
		idmsRules:   make(map[types.UID]*configV1.ImageDigestMirrorSet),
		itmsRules:   make(map[types.UID]*configV1.ImageTagMirrorSet),
	}

	for _, opt := range opts {
		opt(s)
	}

	s.systemContext = &ciTypes.SystemContext{SystemRegistriesConfPath: s.configPath}
	return s
}

// Cleanup resets the store which includes in-memory and disk resources.
func (s *FileStore) Cleanup() {
	s.ruleRWMutex.Lock()
	defer s.ruleRWMutex.Unlock()

	s.icspRules = make(map[types.UID]*operatorV1Alpha1.ImageContentSourcePolicy)
	s.idmsRules = make(map[types.UID]*configV1.ImageDigestMirrorSet)
	s.itmsRules = make(map[types.UID]*configV1.ImageTagMirrorSet)

	if err := os.Remove(s.configPath); err != nil && !os.IsNotExist(err) {
		log.Warnf("Failed to cleanup registry mirror config at %q: %v", s.configPath, err)
	}
	sysregistriesv2.InvalidateCache()
}

func (s *FileStore) updateConfig() error {
	if s.updateDelay == 0 {
		return s.updateConfigNow()
	}

	return s.updateConfigDelayed()
}

func (s *FileStore) updateConfigDelayed() error {
	// Using a different mutex to protect timer vs. rules to avoid deadlock.
	s.timerMutex.Lock()
	defer s.timerMutex.Unlock()

	if s.timer != nil {
		s.timer.Stop()
	}

	s.timer = time.AfterFunc(s.updateDelay, func() {
		if err := s.updateConfigNow(); err != nil {
			log.Errorf("Failed to update the registry mirror config: %v", err)
		}
	})

	return nil
}

func (s *FileStore) updateConfigNow() error {
	// Populate config.
	icspRules, idmsRules, itmsRules := s.getAllMirrorSets()

	config := new(sysregistriesv2.V2RegistriesConf)
	err := registries.EditRegistriesConfig(config, nil, nil, icspRules, idmsRules, itmsRules)
	if err != nil {
		return errors.Wrap(err, "could not create registries config")
	}

	// Encode config.
	var newData bytes.Buffer
	encoder := toml.NewEncoder(&newData)
	if err := encoder.Encode(config); err != nil {
		return errors.Wrap(err, "could not encode registries config")
	}

	// Ensure output dir exists.
	err = os.MkdirAll(filepath.Dir(s.configPath), 0755)
	if err != nil {
		return errors.Wrap(err, "could not create directories")
	}

	// Open handle to output file.
	f, err := os.Create(s.configPath)
	if err != nil {
		return errors.Wrap(err, "could not create/open file")
	}
	defer utils.IgnoreError(f.Close)

	// Write encoded config to output file.
	_, err = f.Write(newData.Bytes())
	if err != nil {
		return errors.Wrap(err, "could not write bytes to file")
	}

	if err := f.Close(); err != nil {
		return errors.Wrap(err, "could not close file")
	}

	// Invalidate the registries cache so that the updated file will be read on next invocation of PullSources.
	sysregistriesv2.InvalidateCache()

	log.Debugf("Successfully updated the registry mirror config at %q with %v icspRules, %v idmsRules, %v itmsRules",
		s.configPath, len(icspRules), len(idmsRules), len(itmsRules))

	return nil
}

// PullSources will return image references in the order they should be attempted when pulling the image.
// The returned slice may (or may not) include the source image based on the registries config.
// Source image must be fully qualified (include the registry hostname).
func (s *FileStore) PullSources(srcImage string) ([]string, error) {
	// FindRegistry will parse the registries config file and cache it for future usage.
	reg, err := sysregistriesv2.FindRegistry(s.systemContext, srcImage)
	// Using errors.Is instead of os.IsNotExist because the return from FindRegistry
	// wraps the error in a type that os.IsNotExist ignores.
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, errors.Wrap(err, "failed finding registry")
	}

	if reg == nil {
		return []string{srcImage}, nil
	}

	// ParseNamed assumes srcImage is a fully qualified references (contains a hostname). If not then
	// reference.ParseNormalizedNamed should be used instead.
	ref, err := reference.ParseNamed(srcImage)
	if err != nil {
		return nil, errors.Wrap(err, "could not create image reference")
	}

	pullSrcs, err := reg.PullSourcesFromReference(ref)
	if err != nil {
		return nil, errors.Wrap(err, "could not pull sources from reference")
	}

	srcs := make([]string, 0, len(pullSrcs))
	for _, src := range pullSrcs {
		ref := src.Reference.String()
		if ref == srcImage && reg.Blocked {
			// The registries config states the src registry should not be contacted.
			continue
		}
		srcs = append(srcs, ref)
	}

	return srcs, nil
}

// UpsertImageContentSourcePolicy will store a new/updated ImageContentSourcePolicy.
func (s *FileStore) UpsertImageContentSourcePolicy(icsp *operatorV1Alpha1.ImageContentSourcePolicy) error {
	s.upsertImageContentSourcePolicyWithLock(icsp)
	return s.updateConfig()
}
func (s *FileStore) upsertImageContentSourcePolicyWithLock(icsp *operatorV1Alpha1.ImageContentSourcePolicy) {
	s.ruleRWMutex.Lock()
	defer s.ruleRWMutex.Unlock()

	s.icspRules[icsp.GetUID()] = icsp
}

// DeleteImageContentSourcePolicy will delete an ImageContentSourcePolicy from the store if it exists.
func (s *FileStore) DeleteImageContentSourcePolicy(uid types.UID) error {
	s.deleteImageContentSourcePolicyWithLock(uid)
	return s.updateConfig()
}

func (s *FileStore) deleteImageContentSourcePolicyWithLock(uid types.UID) {
	s.ruleRWMutex.Lock()
	defer s.ruleRWMutex.Unlock()

	delete(s.icspRules, uid)
}

// UpsertImageDigestMirrorSet will store a new/updated ImageDigestMirrorSet.
func (s *FileStore) UpsertImageDigestMirrorSet(idms *configV1.ImageDigestMirrorSet) error {
	s.upsertImageDigestMirrorSetWithLock(idms)
	return s.updateConfig()
}

func (s *FileStore) upsertImageDigestMirrorSetWithLock(idms *configV1.ImageDigestMirrorSet) {
	s.ruleRWMutex.Lock()
	defer s.ruleRWMutex.Unlock()

	s.idmsRules[idms.GetUID()] = idms
}

// DeleteImageDigestMirrorSet will delete an ImageDigestMirrorSet from the store if it exists.
func (s *FileStore) DeleteImageDigestMirrorSet(uid types.UID) error {
	s.deleteImageDigestMirrorSetWithLock(uid)
	return s.updateConfig()
}

func (s *FileStore) deleteImageDigestMirrorSetWithLock(uid types.UID) {
	s.ruleRWMutex.Lock()
	defer s.ruleRWMutex.Unlock()

	delete(s.idmsRules, uid)
}

// UpsertImageTagMirrorSet will store a new/updated ImageTagMirrorSet.
func (s *FileStore) UpsertImageTagMirrorSet(itms *configV1.ImageTagMirrorSet) error {
	s.upsertImageTagMirrorSetWithLock(itms)
	return s.updateConfig()
}

func (s *FileStore) upsertImageTagMirrorSetWithLock(itms *configV1.ImageTagMirrorSet) {
	s.ruleRWMutex.Lock()
	defer s.ruleRWMutex.Unlock()

	s.itmsRules[itms.GetUID()] = itms
}

// DeleteImageTagMirrorSet will delete an ImageTagMirrorSet from the store if it exists.
func (s *FileStore) DeleteImageTagMirrorSet(uid types.UID) error {
	s.deleteImageTagMirrorSetWithLock(uid)
	return s.updateConfig()
}

func (s *FileStore) deleteImageTagMirrorSetWithLock(uid types.UID) {
	s.ruleRWMutex.Lock()
	defer s.ruleRWMutex.Unlock()

	delete(s.itmsRules, uid)
}

// getAllMirrorSets returns slices of all the stored mirror sets as expected by openshift/runtime-utils.
//
// ref: https://github.com/openshift/runtime-utils/blob/5c488b20a19fc8c1fee9011c41ce70379bc8ca4d/pkg/registries/registries.go#L240
func (s *FileStore) getAllMirrorSets() ([]*operatorV1Alpha1.ImageContentSourcePolicy, []*configV1.ImageDigestMirrorSet, []*configV1.ImageTagMirrorSet) {
	s.ruleRWMutex.RLock()
	defer s.ruleRWMutex.RUnlock()

	return maputil.Values(s.icspRules), maputil.Values(s.idmsRules), maputil.Values(s.itmsRules)
}
