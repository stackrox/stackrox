package registry

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/docker/config"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/registries"
	dockerFactory "github.com/stackrox/rox/pkg/registries/docker"
	rhelFactory "github.com/stackrox/rox/pkg/registries/rhel"
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/tlscheck"
	"github.com/stackrox/rox/pkg/urlfmt"
)

var (
	log = logging.LoggerForModule()
)

// Store stores cluster-internal registries by namespace.
// It is assumed all the registries are Docker registries.
type Store struct {
	factory registries.Factory
	// store maps a namespace to the names of registries accessible from within the namespace.
	store map[string]registries.Set

	// globalRegistries holds registries that are not bound to a namespace and can be used
	// for processing images from any namespace, example: the OCP Global Pull Secret
	globalRegistries registries.Set

	mutex sync.RWMutex

	checkTLS CheckTLS
}

// CheckTLS defines a function which checks if the given address is using TLS.
// An example implementation of this is tlscheck.CheckTLS.
type CheckTLS func(ctx context.Context, origAddr string) (bool, error)

// NewRegistryStore creates a new registry store.
// The passed-in CheckTLS is used to check if a registry uses TLS.
// If checkTLS is nil, tlscheck.CheckTLS is used by default.
func NewRegistryStore(checkTLS CheckTLS) *Store {
	factory := registries.NewFactory(registries.FactoryOptions{
		CreatorFuncs: []registries.CreatorWrapper{
			dockerFactory.Creator,
			rhelFactory.Creator,
		},
	})

	store := &Store{
		factory:          factory,
		store:            make(map[string]registries.Set),
		checkTLS:         tlscheck.CheckTLS,
		globalRegistries: registries.NewSet(factory),
	}

	if checkTLS != nil {
		store.checkTLS = checkTLS
	}

	return store
}

func (rs *Store) getRegistries(namespace string) registries.Set {
	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	regs := rs.store[namespace]
	if regs == nil {
		regs = registries.NewSet(rs.factory)
		rs.store[namespace] = regs
	}

	return regs
}

func createImageIntegration(registry string, dce config.DockerConfigEntry, secure bool) *storage.ImageIntegration {
	registryType := dockerFactory.GenericDockerRegistryType
	if rhelFactory.RedHatRegistryEndpoints.Contains(urlfmt.TrimHTTPPrefixes(registry)) {
		registryType = rhelFactory.RedHatRegistryType
	}

	return &storage.ImageIntegration{
		Id:         registry,
		Name:       registry,
		Type:       registryType,
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
		IntegrationConfig: &storage.ImageIntegration_Docker{
			Docker: &storage.DockerConfig{
				Endpoint: registry,
				Username: dce.Username,
				Password: dce.Password,
				Insecure: !secure,
			},
		},
	}
}

// UpsertRegistry upserts the given registry with the given credentials in the given namespace into the store.
func (rs *Store) UpsertRegistry(ctx context.Context, namespace, registry string, dce config.DockerConfigEntry) error {
	secure, err := rs.checkTLS(ctx, registry)
	if err != nil {
		return errors.Wrapf(err, "unable to check TLS for registry %q", registry)
	}

	regs := rs.getRegistries(namespace)

	// remove http/https prefixes from registry, matching may fail otherwise, the created registry.url will have
	// the appropriate prefix
	registry = urlfmt.TrimHTTPPrefixes(registry)
	err = regs.UpdateImageIntegration(createImageIntegration(registry, dce, secure))
	if err != nil {
		return errors.Wrapf(err, "updating registry store with registry %q", registry)
	}

	log.Debugf("Upserted registry %q for namespace %q into store", registry, namespace)

	return nil
}

// getRegistriesInNamespace returns all the registries within a given namespace.
func (rs *Store) getRegistriesInNamespace(namespace string) registries.Set {
	rs.mutex.RLock()
	defer rs.mutex.RUnlock()

	return rs.store[namespace]
}

// GetRegistryForImage returns the relevant image registry for the given image.
//
// An error is returned if the registry is unknown.
//
// Assumes the image is from an OCP internal registry
func (rs *Store) GetRegistryForImage(image *storage.ImageName) (registryTypes.Registry, error) {
	ns := utils.ExtractOpenShiftProject(image)
	return rs.GetRegistryForImageInNamespace(image, ns)
}

// HasRegistryForImage returns true when the registry store has the registry
// for the given image.
//
// Assumes the image is from an OCP internal registry
func (rs *Store) HasRegistryForImage(image *storage.ImageName) bool {
	reg, err := rs.GetRegistryForImage(image)
	return reg != nil && err == nil
}

// GetRegistryForImageInNamespace returns the stored registry that matches image.Registry
// and is associated with namespace
//
// An error is returned if no registry found
func (rs *Store) GetRegistryForImageInNamespace(image *storage.ImageName, namespace string) (registryTypes.Registry, error) {
	reg := image.GetRegistry()
	regs := rs.getRegistriesInNamespace(namespace)
	if regs != nil {
		for _, r := range regs.GetAll() {
			if r.Name() == reg {
				return r, nil
			}
		}
	}

	return nil, errors.Errorf("unknown image registry: %q", reg)
}

// UpsertGlobalRegistry will store a new registry with the given credentials into the global registry store
func (rs *Store) UpsertGlobalRegistry(ctx context.Context, registry string, dce config.DockerConfigEntry) error {
	secure, err := rs.checkTLS(ctx, registry)
	if err != nil {
		return errors.Wrapf(err, "unable to check TLS for registry %q", registry)
	}

	err = rs.globalRegistries.UpdateImageIntegration(createImageIntegration(registry, dce, secure))
	if err != nil {
		return errors.Wrapf(err, "updating registry store with registry %q", registry)
	}

	log.Debugf("Upserted global registry %q into store", registry)

	return nil
}

// GetGlobalRegistryForImage returns the relevant global registry for image
//
// An error is returned if the registry is unknown
func (rs *Store) GetGlobalRegistryForImage(image *storage.ImageName) (registryTypes.Registry, error) {
	reg := image.GetRegistry()
	regs := rs.globalRegistries
	if regs != nil {
		for _, r := range regs.GetAll() {
			if r.Name() == reg {
				return r, nil
			}
		}
	}

	return nil, errors.Errorf("unknown image registry: %q", reg)
}
