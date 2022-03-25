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
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/tlscheck"
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
	store := &Store{
		factory: registries.NewFactory(registries.FactoryOptions{
			CreatorFuncs: []registries.CreatorWrapper{dockerFactory.Creator},
		}),
		store: make(map[string]registries.Set),

		checkTLS: tlscheck.CheckTLS,
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

// UpsertRegistry upserts the given registry with the given credentials in the given namespace into the store.
func (rs *Store) UpsertRegistry(ctx context.Context, namespace, registry string, dce config.DockerConfigEntry) error {
	secure, err := rs.checkTLS(ctx, registry)
	if err != nil {
		return errors.Wrapf(err, "unable to check TLS for registry %q", registry)
	}

	regs := rs.getRegistries(namespace)
	err = regs.UpdateImageIntegration(&storage.ImageIntegration{
		Id:         registry,
		Name:       registry,
		Type:       "docker",
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
		IntegrationConfig: &storage.ImageIntegration_Docker{
			Docker: &storage.DockerConfig{
				Endpoint: registry,
				Username: dce.Username,
				Password: dce.Password,
				Insecure: !secure,
			},
		},
	})
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
// An error is returned if the registry is unknown.
func (rs *Store) GetRegistryForImage(image *storage.ImageName) (registryTypes.Registry, error) {
	reg := image.GetRegistry()

	ns := utils.ExtractOpenShiftProject(image)
	regs := rs.getRegistriesInNamespace(ns)
	if regs != nil {
		for _, r := range regs.GetAll() {
			if r.Name() == reg {
				return r, nil
			}
		}
	}

	return nil, errors.Errorf("Unknown image registry: %q", reg)
}
