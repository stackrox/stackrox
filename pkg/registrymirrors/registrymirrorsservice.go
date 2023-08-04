package registrymirrors

import (
	"bytes"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/containers/image/v5/pkg/sysregistriesv2"
	"github.com/containers/image/v5/types"
	"github.com/docker/distribution/reference"
	configV1 "github.com/openshift/api/config/v1"
	operatorV1Alpha1 "github.com/openshift/api/operator/v1alpha1"
	"github.com/openshift/runtime-utils/pkg/registries"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	log = logging.LoggerForModule()
)

// TODO: Should this be changed to implement Cleanup()? Add the cleanup method and change this to a store?

// Service defines an interface for interacting with registry mirrors.
type Service interface {
	// UpdateConfig updates the consolidated registry mirror config using the provided rules.
	UpdateConfig(icspRules []*operatorV1Alpha1.ImageContentSourcePolicy, idmsRules []*configV1.ImageDigestMirrorSet, itmsRules []*configV1.ImageTagMirrorSet) error

	// PullSources will return image references in the order they should be attempted when pulling the image.
	// An empty slice indicates that there are no mirrors given the source image and config. The
	// returned slice may (or may not) include srcImage based on the mirroring rules. Source image must be
	// fully qualified (include the registry hostname).
	PullSources(srcImage string) ([]string, error)
	// TODO: change PullSources to return ACS specific pull sources
	// TODO: Test PullSources when no registries.conf exists - that error does not mean failure in our scenario
}

// FileService stores/reads the consolidated mirror config from a filesystem.
type FileService struct {
	configPath    string
	systemContext *types.SystemContext
}

var _ Service = (*FileService)(nil)

// NewFileMirrorService creates a new FileRegistryMirrorsService.
func NewFileService(configPath string) *FileService {
	return &FileService{
		configPath:    configPath,
		systemContext: &types.SystemContext{SystemRegistriesConfPath: configPath},
	}
}

func (s *FileService) UpdateConfig(icspRules []*operatorV1Alpha1.ImageContentSourcePolicy, idmsRules []*configV1.ImageDigestMirrorSet, itmsRules []*configV1.ImageTagMirrorSet) error {
	// Populate config.
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

func (s *FileService) PullSources(srcImage string) ([]string, error) {
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

// DelayedService wraps Service with a delay added to UpdateConfig.
type DelayedService struct {
	svc         Service
	mutex       sync.Mutex
	updateDelay time.Duration
	timer       *time.Timer
}

var _ Service = (*DelayedService)(nil)

func NewDelayedService(base Service) *DelayedService {
	return &DelayedService{
		svc:         base,
		updateDelay: time.Millisecond * 200,
	}
}

// PullSources implements Service.
func (s *DelayedService) PullSources(srcImage string) ([]string, error) {
	return s.svc.PullSources(srcImage)
}

// UpdateConfig will update the registry mirror config after a delay. If the previous
// invocation has not yet completed it will be canceled. Any error encountered will
// be logged but NOT returned due to the async update.
func (s *DelayedService) UpdateConfig(icspRules []*operatorV1Alpha1.ImageContentSourcePolicy, idmsRules []*configV1.ImageDigestMirrorSet, itmsRules []*configV1.ImageTagMirrorSet) error {
	// Use a mutex to protect access to s.timer.
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.timer != nil {
		s.timer.Stop()
	}

	s.timer = time.AfterFunc(s.updateDelay, func() {
		if err := s.svc.UpdateConfig(icspRules, idmsRules, itmsRules); err != nil {
			log.Errorf("Failed to update the registry mirror config: %v", err)
		}
	})

	return nil
}
