package registry

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/docker/config"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/registries"
	dockerFactory "github.com/stackrox/rox/pkg/registries/docker"
	rhelFactory "github.com/stackrox/rox/pkg/registries/rhel"
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/tlscheck"
	"github.com/stackrox/rox/pkg/urlfmt"
)

const (
	pullSecretNamePrefix = "PullSec"
	globalRegNamePrefix  = "Global"
)

var (
	log = logging.LoggerForModule()
)

// Store stores cluster-internal registries by namespace.
type Store struct {
	factory registries.Factory
	// store maps a namespace to the names of registries accessible from within the namespace.
	store map[string]registries.Set

	// clusterLocalRegistryHosts contains hosts (names and/or IPs) for registries that are local
	// to this cluster (ie: the OCP internal registry).
	clusterLocalRegistryHosts      set.StringSet
	clusterLocalRegistryHostsMutex sync.RWMutex

	// globalRegistries holds registries that are not bound to a namespace and can be used
	// for processing images from any namespace, example: the OCP Global Pull Secret.
	globalRegistries registries.Set

	mutex sync.RWMutex

	checkTLS CheckTLS

	// delegatedRegistryConfig is used to determine if scanning images from a registry
	// should be done via local scanner or sent to central.
	delegatedRegistryConfig      *central.DelegatedRegistryConfig
	delegatedRegistryConfigMutex sync.RWMutex

	// centralRegistryIntegration holds registry integrations sync'd from Central.
	centralRegistryIntegrations registries.Set
}

// ReconcileDelete is called after Sensor reconnects with Central and receives its state hashes.
// Reconciliacion ensures that Sensor and Central have the same state by checking whether a given resource
// shall be deleted from Central.
func (rs *Store) ReconcileDelete(resType, resID string, resHash uint64) (string, error) {
	_, _, _ = resType, resID, resHash
	// TODO implement me
	panic("implement me")
}

// CheckTLS defines a function which checks if the given address is using TLS.
// An example implementation of this is tlscheck.CheckTLS.
type CheckTLS func(ctx context.Context, origAddr string) (bool, error)

// NewRegistryStore creates a new registry store.
// The passed-in CheckTLS is used to check if a registry uses TLS.
// If checkTLS is nil, tlscheck.CheckTLS is used by default.
func NewRegistryStore(checkTLS CheckTLS) *Store {
	regFactory := registries.NewFactory(registries.FactoryOptions{
		CreatorFuncs: registries.AllCreatorFuncsWithoutRepoList,
	})

	store := &Store{
		factory:                     regFactory,
		store:                       make(map[string]registries.Set),
		checkTLS:                    tlscheck.CheckTLS,
		globalRegistries:            registries.NewSet(regFactory),
		centralRegistryIntegrations: registries.NewSet(regFactory),
		clusterLocalRegistryHosts:   set.NewStringSet(),
	}

	if checkTLS != nil {
		store.checkTLS = checkTLS
	}

	return store
}

// Cleanup deletes all entries from store.
func (rs *Store) Cleanup() {
	// Separate cleanup methods are used to ensure only one lock is obtained at a time
	// to avoid accidental deadlock.
	rs.cleanupRegistries()
	rs.cleanupClusterLocalRegistryHosts()
	rs.cleanupDelegatedRegistryConfig()
}

func (rs *Store) cleanupRegistries() {
	// These Sets have an internal mutex for controlling access.
	rs.centralRegistryIntegrations.Clear()
	rs.globalRegistries.Clear()

	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	rs.store = make(map[string]registries.Set)
}

func (rs *Store) cleanupClusterLocalRegistryHosts() {
	rs.clusterLocalRegistryHostsMutex.Lock()
	defer rs.clusterLocalRegistryHostsMutex.Unlock()

	rs.clusterLocalRegistryHosts = set.NewStringSet()
}

func (rs *Store) cleanupDelegatedRegistryConfig() {
	rs.delegatedRegistryConfigMutex.Lock()
	defer rs.delegatedRegistryConfigMutex.Unlock()

	rs.delegatedRegistryConfig = nil
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

func createImageIntegration(registry string, dce config.DockerConfigEntry, secure bool, name string) *storage.ImageIntegration {
	registryType := dockerFactory.GenericDockerRegistryType
	if rhelFactory.RedHatRegistryEndpoints.Contains(urlfmt.TrimHTTPPrefixes(registry)) {
		registryType = rhelFactory.RedHatRegistryType
	}

	return &storage.ImageIntegration{
		Id:         registry,
		Name:       name,
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

// genIntegrationName returns a string to use as an integration name. It's meant to aid in identifying where
// the registry came from.
func genIntegrationName(prefix string, namespace string, registry string) string {
	if namespace != "" {
		namespace = fmt.Sprintf("/ns:%s", namespace)
	}

	if registry != "" {
		registry = fmt.Sprintf("/reg:%s", registry)
	}

	return fmt.Sprintf("%v%v%v", prefix, namespace, registry)
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
	name := genIntegrationName(pullSecretNamePrefix, namespace, registry)
	err = regs.UpdateImageIntegration(createImageIntegration(registry, dce, secure, name))
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

// GetRegistryForImageInNamespace returns the stored registry that matches image.Registry
// and is associated with namespace.
//
// An error is returned if no registry found.
func (rs *Store) GetRegistryForImageInNamespace(image *storage.ImageName, namespace string) (registryTypes.ImageRegistry, error) {
	reg := image.GetRegistry()
	regs := rs.getRegistriesInNamespace(namespace)
	if regs != nil {
		for _, r := range regs.GetAll() {
			if r.Config().RegistryHostname == reg {
				return r, nil
			}
		}
	}

	return nil, errors.Errorf("unknown image registry: %q", reg)
}

// UpsertGlobalRegistry will store a new registry with the given credentials into the global registry store.
func (rs *Store) UpsertGlobalRegistry(ctx context.Context, registry string, dce config.DockerConfigEntry) error {
	secure, err := rs.checkTLS(ctx, registry)
	if err != nil {
		return errors.Wrapf(err, "unable to check TLS for registry %q", registry)
	}

	name := genIntegrationName(globalRegNamePrefix, "", registry)
	err = rs.globalRegistries.UpdateImageIntegration(createImageIntegration(registry, dce, secure, name))
	if err != nil {
		return errors.Wrapf(err, "updating registry store with registry %q", registry)
	}

	log.Debugf("Upserted global registry %q into store", registry)

	return nil
}

// GetGlobalRegistryForImage returns the relevant global registry for image.
//
// An error is returned if the registry is unknown.
func (rs *Store) GetGlobalRegistryForImage(image *storage.ImageName) (registryTypes.ImageRegistry, error) {
	reg := image.GetRegistry()
	regs := rs.globalRegistries
	if regs != nil {
		for _, r := range regs.GetAll() {
			if r.Config().RegistryHostname == reg {
				return r, nil
			}
		}
	}

	return nil, errors.Errorf("unknown image registry: %q", reg)
}

// SetDelegatedRegistryConfig sets a new delegated registry config for use in determining
// if a particular image is from a registry that should be accessed local to this cluster.
func (rs *Store) SetDelegatedRegistryConfig(config *central.DelegatedRegistryConfig) {
	rs.delegatedRegistryConfigMutex.Lock()
	defer rs.delegatedRegistryConfigMutex.Unlock()
	rs.delegatedRegistryConfig = config
}

// IsLocal determines if an image is from a registry that should be accessed
// local to this secured cluster.  Always returns true for image registries that have
// been added via AddClusterLocalRegistryHost.
func (rs *Store) IsLocal(image *storage.ImageName) bool {
	if image == nil {
		return false
	}

	if rs.hasClusterLocalRegistryHost(image.GetRegistry()) {
		// This host is always cluster local irregardless of the DelegatedRegistryConfig (ie: OCP internal registry).
		return true
	}

	imageFullName := urlfmt.TrimHTTPPrefixes(image.GetFullName())

	rs.delegatedRegistryConfigMutex.RLock()
	defer rs.delegatedRegistryConfigMutex.RUnlock()

	config := rs.delegatedRegistryConfig
	if config == nil || config.EnabledFor == central.DelegatedRegistryConfig_NONE {
		return false
	}

	if config.EnabledFor == central.DelegatedRegistryConfig_ALL {
		return true
	}

	// if image matches a delegated registry prefix, it is local
	for _, r := range config.Registries {
		regPath := urlfmt.TrimHTTPPrefixes(r.GetPath())
		if strings.HasPrefix(imageFullName, regPath) {
			return true
		}
	}

	return false
}

// AddClusterLocalRegistryHost adds host to an internal set of hosts representing
// registries that are only accessible from this cluster. These hosts will be factored
// into IsLocal decisions. Is OK to call repeatedly for the same host.
func (rs *Store) AddClusterLocalRegistryHost(host string) {
	trimmed := urlfmt.TrimHTTPPrefixes(host)

	rs.clusterLocalRegistryHostsMutex.Lock()
	defer rs.clusterLocalRegistryHostsMutex.Unlock()

	rs.clusterLocalRegistryHosts.Add(trimmed)

	log.Debugf("Added cluster local registry host %q", trimmed)
}

func (rs *Store) hasClusterLocalRegistryHost(host string) bool {
	trimmed := urlfmt.TrimHTTPPrefixes(host)

	rs.clusterLocalRegistryHostsMutex.RLock()
	defer rs.clusterLocalRegistryHostsMutex.RUnlock()

	return rs.clusterLocalRegistryHosts.Contains(trimmed)
}

// UpsertCentralRegistryIntegrations upserts registry integrations from Central into the store.
func (rs *Store) UpsertCentralRegistryIntegrations(iis []*storage.ImageIntegration) {
	for _, ii := range iis {
		err := rs.centralRegistryIntegrations.UpdateImageIntegration(ii)
		if err != nil {
			log.Errorf("Failed to upsert registry integration %q: %v", ii.GetId(), err)
		} else {
			log.Debugf("Upserted registry integration %q (%q)", ii.GetName(), ii.GetId())
		}
	}
}

// DeleteCentralRegistryIntegrations deletes registry integrations from the store.
func (rs *Store) DeleteCentralRegistryIntegrations(ids []string) {
	for _, id := range ids {
		err := rs.centralRegistryIntegrations.RemoveImageIntegration(id)
		if err != nil {
			log.Errorf("Failed to delete registry integration %q: %v", id, err)
		} else {
			log.Debugf("Deleted registry integration %q", id)
		}
	}
}

// GetMatchingCentralRegistryIntegrations returns registry integrations sync'd from Central that match the
// provided image name.
func (rs *Store) GetMatchingCentralRegistryIntegrations(imgName *storage.ImageName) []registryTypes.ImageRegistry {
	var regs []registryTypes.ImageRegistry
	for _, ii := range rs.centralRegistryIntegrations.GetAll() {
		if ii.Match(imgName) {
			regs = append(regs, ii)
		}
	}

	return regs
}
