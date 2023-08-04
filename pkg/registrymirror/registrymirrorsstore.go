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
	J
	// default delay before writing the updated registries config
	defaultDelay = time.Millisecond * 300
)

var (
	log = logging.LoggerForModule()
)

// TODO:
// - change PullSources to return ACS specific pull sources
// - Test PullSources when no registries.conf exists - that error does not mean failure in our scenario

// Store defines an interface for interacting with registry mirrors.
type Store interface {
	Cleanup()

	UpsertImageContentSourcePolicy(icsp *operatorV1Alpha1.ImageContentSourcePolicy)
	DeleteImageContentSourcePolicy(uid types.UID)

	UpsertImageDigestMirrorSet(idms *configV1.ImageDigestMirrorSet)
	DeleteImageDigestMirrorSet(uid types.UID)

	UpsertImageTagMirrorSet(itms *configV1.ImageTagMirrorSet)
	DeleteImageTagMirrorSet(uid types.UID)

	// PullSources will return image references in the order they should be attempted when pulling the image.
	// An empty slice indicates that there are no mirrors given the source image and config. The
	// returned slice may (or may not) include srcImage based on the mirroring rules. Source image must be
	// fully qualified (include the registry hostname).
	PullSources(srcImage string) ([]string, error)
}

// FileStore stores/reads the consolidated mirror config from a filesystem.
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

// WithConfigPath sets the path to read/write the consolidated mirroring config.
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

func (s *FileStore) Cleanup() {
	s.ruleRWMutex.Lock()
	defer s.ruleRWMutex.Unlock()

	s.icspRules = make(map[types.UID]*operatorV1Alpha1.ImageContentSourcePolicy)
	s.idmsRules = make(map[types.UID]*configV1.ImageDigestMirrorSet)
	s.itmsRules = make(map[types.UID]*configV1.ImageTagMirrorSet)
}

func (s *FileStore) updateConfig() error {
	if s.updateDelay == 0 {
		return s.updateConfigNow()
	}

	return s.updateConfigDelayed()
}

func (s *FileStore) updateConfigDelayed() error {
	// Using a different mutex to protect timer vs. rulesMutex to avoid deadlock.
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

func (s *FileStore) PullSources(srcImage string) ([]string, error) {
	// FindRegistry will parse the registries config file and cache it for future usage.
	reg, err := sysregistriesv2.FindRegistry(s.systemContext, srcImage)
	if err != nil {
		return nil, errors.Wrap(err, "could not find registry")
	}

	// This assumes srcImage is a fully qualified references (contains a hostname). If not then
	// reference.ParseNormalizedNamed should be used instead.
	ref, err := reference.ParseNamed(srcImage)
	if err != nil {
		return nil, errors.Wrap(err, "could not create reference")
	}

	pullSrcs, err := reg.PullSourcesFromReference(ref)
	if err != nil {
		return nil, errors.Wrap(err, "could not pull sources from reference")
	}

	srcs := make([]string, 0, len(pullSrcs))
	for _, src := range pullSrcs {
		srcs = append(srcs, src.Reference.String())
	}

	return srcs, nil
}

// UpsertImageContentSourcePolicy will store a new/updated ImageContentSourcePolicy.
func (s *FileStore) UpsertImageContentSourcePolicy(icsp *operatorV1Alpha1.ImageContentSourcePolicy) {
	s.upsertImageContentSourcePolicyWithLock(icsp)
	s.updateConfig()
}
func (s *FileStore) upsertImageContentSourcePolicyWithLock(icsp *operatorV1Alpha1.ImageContentSourcePolicy) {
	s.ruleRWMutex.Lock()
	defer s.ruleRWMutex.Unlock()

	s.icspRules[icsp.GetUID()] = icsp
}

// DeleteImageContentSourcePolicy will delete an ImageContentSourcePolicy from the store if it exists.
func (s *FileStore) DeleteImageContentSourcePolicy(uid types.UID) {
	s.deleteImageContentSourcePolicyWithLock(uid)
	s.updateConfig()
}

func (s *FileStore) deleteImageContentSourcePolicyWithLock(uid types.UID) {
	s.ruleRWMutex.Lock()
	defer s.ruleRWMutex.Unlock()

	delete(s.icspRules, uid)
}

// UpsertImageDigestMirrorSet will store a new/updated ImageDigestMirrorSet.
func (s *FileStore) UpsertImageDigestMirrorSet(idms *configV1.ImageDigestMirrorSet) {
	s.upsertImageDigestMirrorSetWithLock(idms)
	s.updateConfig()
}

func (s *FileStore) upsertImageDigestMirrorSetWithLock(idms *configV1.ImageDigestMirrorSet) {
	s.ruleRWMutex.Lock()
	defer s.ruleRWMutex.Unlock()

	s.idmsRules[idms.GetUID()] = idms
}

// DeleteImageDigestMirrorSet will delete an ImageDigestMirrorSet from the store if it exists.
func (s *FileStore) DeleteImageDigestMirrorSet(uid types.UID) {
	s.deleteImageDigestMirrorSetWithLock(uid)
	s.updateConfig()
}

func (s *FileStore) deleteImageDigestMirrorSetWithLock(uid types.UID) {
	s.ruleRWMutex.Lock()
	defer s.ruleRWMutex.Unlock()

	delete(s.idmsRules, uid)
}

// UpsertImageTagMirrorSet will store a new/updated ImageTagMirrorSet.
func (s *FileStore) UpsertImageTagMirrorSet(itms *configV1.ImageTagMirrorSet) {
	s.upsertImageTagMirrorSetWithLock(itms)
	s.updateConfig()
}

func (s *FileStore) upsertImageTagMirrorSetWithLock(itms *configV1.ImageTagMirrorSet) {
	s.ruleRWMutex.Lock()
	defer s.ruleRWMutex.Unlock()

	s.itmsRules[itms.GetUID()] = itms
}

// DeleteImageTagMirrorSet will delete an ImageTagMirrorSet from the store if it exists.
func (s *FileStore) DeleteImageTagMirrorSet(uid types.UID) {
	s.deleteImageTagMirrorSetWithLock(uid)
	s.updateConfig()
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
