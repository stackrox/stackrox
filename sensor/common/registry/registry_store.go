package registry

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/docker/config"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/openshift"
	"github.com/stackrox/rox/pkg/registries"
	rhelFactory "github.com/stackrox/rox/pkg/registries/rhel"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/tlscheck"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/sensor/common/cloudproviders/gcp"
	"github.com/stackrox/rox/sensor/common/registry/metrics"
)

const defaultSA = "default"

var (
	log       = logging.LoggerForModule()
	bgContext = context.Background()
)

// namespaceToSecretName is an alias for a map of namespaces to another map keyed by secret name.
type namespaceToSecretName = map[string]secretNameToHostname

// secretNameToHostname is an alias for a map of secret names to another map keyed by registry hostname.
type secretNameToHostname = map[string]hostnameToRegistry

// hostnameToRegistry is an alias for a map of registry hostnames to image registries.
type hostnameToRegistry = map[string]types.ImageRegistry

// Store stores cluster-internal registries by namespace.
type Store struct {
	factory registries.Factory

	// storeByHost maps a namespace to registries accessible from within the namespace.
	// Only one of storeByHost or storeByName will be active at any given time.
	storeByHost map[string]registries.Set

	// storeByName maps a namespace to secret names to host names to a registry. This more
	// closely resembles how pull secrets are represented in k8s.  Only one of
	// storeByHost or storeByName will be active at any given time.
	storeByName namespaceToSecretName

	// storeMutux controls access to storeByHost or storeByName (whichever is active).
	storeMutux sync.RWMutex

	// clusterLocalRegistryHosts contains hosts (names and/or IPs) for registries that are local
	// to this cluster (ie: the OCP internal registry).
	clusterLocalRegistryHosts      set.StringSet
	clusterLocalRegistryHostsMutex sync.RWMutex

	// globalRegistries holds registries that are not bound to a namespace and can be used
	// for processing images from any namespace, example: the OCP Global Pull Secret.
	globalRegistries registries.Set

	// delegatedRegistryConfig is used to determine if scanning images from a registry
	// should be done via local scanner or sent to central.
	delegatedRegistryConfig      *central.DelegatedRegistryConfig
	delegatedRegistryConfigMutex sync.RWMutex

	// centralRegistryIntegration holds registry integrations sync'd from Central.
	centralRegistryIntegrations registries.Set

	tlsCheckCache *tlsCheckCacheImpl
}

// CheckTLS defines a function which checks if the given address is using TLS.
// An example implementation of this is tlscheck.CheckTLS.
type CheckTLS func(ctx context.Context, origAddr string) (bool, error)

// NewRegistryStore creates a new registry store.
// The passed-in CheckTLS is used to check if a registry uses TLS.
// If checkTLS is nil, tlscheck.CheckTLS is used by default.
func NewRegistryStore(checkTLSFunc CheckTLS) *Store {
	if checkTLSFunc == nil {
		checkTLSFunc = tlscheck.CheckTLS
	}
	tlsCheckCache := newTLSCheckCache(checkTLSFunc)

	defaultFactory := registries.NewFactory(registries.FactoryOptions{
		CreatorFuncs: registries.AllCreatorFuncsWithoutRepoList,
	})

	factory := newLazyFactory(tlsCheckCache)

	store := &Store{
		factory:     factory,
		storeByHost: make(map[string]registries.Set),
		storeByName: make(namespaceToSecretName),
		globalRegistries: registries.NewSet(
			factory,
			types.WithMetricsHandler(metrics.Singleton()),
			types.WithGCPTokenManager(gcp.Singleton()),
		),
		centralRegistryIntegrations: registries.NewSet(
			defaultFactory,
			types.WithMetricsHandler(metrics.Singleton()),
			types.WithGCPTokenManager(gcp.Singleton()),
		),
		clusterLocalRegistryHosts: set.NewStringSet(),
		tlsCheckCache:             tlsCheckCache,
	}

	return store
}

// Cleanup deletes all entries from store that are derived from k8s informers/listeners.
// The lifecycle of other data in this store will be handled separately, such as the delegated
// registry config and image integrations synced from Central.
func (rs *Store) Cleanup() {
	// Separate cleanup methods are used to ensure only one lock is obtained at a time
	// to avoid accidental deadlock.
	rs.cleanupRegistries()
	rs.cleanupClusterLocalRegistryHosts()
	rs.tlsCheckCache.Cleanup()

	metrics.ResetRegistryMetrics()

	log.Info("Registry store cleared.")
}

func (rs *Store) cleanupRegistries() {
	// This set has an internal mutex for controlling access.
	rs.globalRegistries.Clear()

	rs.storeMutux.Lock()
	defer rs.storeMutux.Unlock()

	clear(rs.storeByHost)
	clear(rs.storeByName)
}

func (rs *Store) cleanupClusterLocalRegistryHosts() {
	rs.clusterLocalRegistryHostsMutex.Lock()
	defer rs.clusterLocalRegistryHostsMutex.Unlock()

	rs.clusterLocalRegistryHosts = set.NewStringSet()
}

func (rs *Store) getRegistries(namespace string) registries.Set {
	rs.storeMutux.Lock()
	defer rs.storeMutux.Unlock()

	regs := rs.storeByHost[namespace]
	if regs == nil {
		regs = registries.NewSet(rs.factory, types.WithGCPTokenManager(gcp.Singleton()))
		rs.storeByHost[namespace] = regs
	}

	return regs
}

func createImageIntegration(host string, dce config.DockerConfigEntry, name string) *storage.ImageIntegration {
	registryType := types.DockerType
	if rhelFactory.RedHatRegistryEndpoints.Contains(urlfmt.TrimHTTPPrefixes(host)) {
		registryType = types.RedHatType
	}

	return &storage.ImageIntegration{
		Id:         name,
		Name:       name,
		Type:       registryType,
		Categories: []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY},
		IntegrationConfig: &storage.ImageIntegration_Docker{
			Docker: &storage.DockerConfig{
				Endpoint: host,
				Username: dce.Username,
				Password: dce.Password,
			},
		},
	}
}

// genIntegrationName returns a string to use as an integration name. It's meant to aid in identifying where
// the registry came from.
func genIntegrationName(prefix, namespace, secretName, registry string) string {
	if namespace != "" {
		namespace = fmt.Sprintf("/ns:%s", namespace)
	}

	if secretName != "" {
		secretName = fmt.Sprintf("/name:%s", secretName)
	}

	if registry != "" {
		registry = fmt.Sprintf("/reg:%s", registry)
	}

	return fmt.Sprintf("%s%s%s%s", prefix, namespace, secretName, registry)
}

// upsertRegistry upserts the given registry with the given credentials in the given namespace into the store.
func (rs *Store) upsertRegistry(namespace, registry, host string, dce config.DockerConfigEntry) error {
	regs := rs.getRegistries(namespace)

	// remove http/https prefixes from registry, matching may fail otherwise, the created registry.url will have
	// the appropriate prefix
	registry = urlfmt.TrimHTTPPrefixes(registry)
	name := genIntegrationName(types.PullSecretNamePrefix, namespace, "", registry)

	ii := createImageIntegration(host, dce, name)
	inserted, err := regs.UpdateImageIntegration(ii)
	if err != nil {
		return errors.Wrapf(err, "updating registry store with registry %q", registry)
	}

	log.Debugf("Upserted registry %q for namespace %q into store", registry, namespace)

	if inserted {
		// A new entry was inserted (not updated).
		metrics.IncrementPullSecretEntriesCount(1)
		metrics.IncrementPullSecretEntriesSize(ii.SizeVT())
	}

	return nil
}

// getRegistriesInNamespace returns all the registries within a given namespace.
func (rs *Store) getRegistriesInNamespace(namespace string) registries.Set {
	rs.storeMutux.RLock()
	defer rs.storeMutux.RUnlock()

	return rs.storeByHost[namespace]
}

// getRegistryForImageInNamespace returns the stored registry that matches image.Registry
// and is associated with namespace.
//
// An error is returned if no registry found.
func (rs *Store) getRegistryForImageInNamespace(image *storage.ImageName, namespace string) (types.ImageRegistry, error) {
	reg := image.GetRegistry()
	regs := rs.getRegistriesInNamespace(namespace)
	if regs != nil {
		for _, r := range regs.GetAll() {
			if r.Config(bgContext).GetRegistryHostname() == reg {
				return r, nil
			}
		}
	}

	return nil, errors.Errorf("unknown image registry: %q", reg)
}

// upsertGlobalRegistry will store a new registry with the given credentials into the global registry store.
func (rs *Store) upsertGlobalRegistry(registry, host string, dce config.DockerConfigEntry) error {
	name := genIntegrationName(types.GlobalRegNamePrefix, "", "", registry)
	_, err := rs.globalRegistries.UpdateImageIntegration(createImageIntegration(host, dce, name))
	if err != nil {
		return errors.Wrapf(err, "updating registry store with registry %q", registry)
	}

	log.Debugf("Upserted global registry %q into store", registry)

	metrics.SetGlobalSecretEntriesCount(rs.globalRegistries.Len())

	return nil
}

// GetGlobalRegistries returns the relevant global registry for image.
//
// An error is returned if the registry is unknown.
func (rs *Store) GetGlobalRegistries(image *storage.ImageName) ([]types.ImageRegistry, error) {
	reg := image.GetRegistry()
	matchedRegs := []types.ImageRegistry{}
	if rs.globalRegistries != nil {
		for _, r := range rs.globalRegistries.GetAll() {
			if r.Config(bgContext).GetRegistryHostname() == reg {
				matchedRegs = append(matchedRegs, r)
			}
		}
	}
	if len(matchedRegs) > 0 {
		return matchedRegs, nil
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
		// This host is always cluster local regardless of the DelegatedRegistryConfig (ie: OCP internal registry).
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

// addClusterLocalRegistryHost adds host to an internal set of hosts representing
// registries that are only accessible from this cluster. These hosts will be factored
// into IsLocal decisions. Is OK to call repeatedly for the same host.
func (rs *Store) addClusterLocalRegistryHost(host string) {
	trimmed := urlfmt.TrimHTTPPrefixes(host)

	rs.clusterLocalRegistryHostsMutex.Lock()
	defer rs.clusterLocalRegistryHostsMutex.Unlock()

	if rs.clusterLocalRegistryHosts.Add(trimmed) {
		log.Infof("Added cluster local registry host %q", trimmed)

		metrics.SetClusterLocalHostsCount(len(rs.clusterLocalRegistryHosts))
	}
}

func (rs *Store) hasClusterLocalRegistryHost(host string) bool {
	trimmed := urlfmt.TrimHTTPPrefixes(host)

	rs.clusterLocalRegistryHostsMutex.RLock()
	defer rs.clusterLocalRegistryHostsMutex.RUnlock()

	return rs.clusterLocalRegistryHosts.Contains(trimmed)
}

// UpsertCentralRegistryIntegrations upserts registry integrations from Central into the store.
func (rs *Store) UpsertCentralRegistryIntegrations(iis []*storage.ImageIntegration, refresh bool) {
	if refresh {
		// On refresh any existing integrations should be replaced.
		rs.centralRegistryIntegrations.Clear()
	}

	for _, ii := range iis {
		_, err := rs.centralRegistryIntegrations.UpdateImageIntegration(ii)
		if err != nil {
			log.Errorf("Failed to upsert registry integration %q: %v", ii.GetId(), err)
		} else {
			log.Debugf("Upserted registry integration %q (%q)", ii.GetName(), ii.GetId())
		}
	}

	metrics.SetCentralIntegrationCount(rs.centralRegistryIntegrations.Len())
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

	metrics.SetCentralIntegrationCount(rs.centralRegistryIntegrations.Len())
}

// GetCentralRegistries returns registry integrations sync'd from Central that match the
// provided image name.
func (rs *Store) GetCentralRegistries(imgName *storage.ImageName) []types.ImageRegistry {
	var regs []types.ImageRegistry
	for _, ii := range rs.centralRegistryIntegrations.GetAll() {
		if ii.Match(imgName) {
			regs = append(regs, ii)
		}
	}

	return regs
}

// UpsertSecret upserts a pull secret into the store.
func (rs *Store) UpsertSecret(namespace, secretName string, dockerConfig config.DockerConfig, serviceAcctName string) {
	if !features.SensorPullSecretsByName.Enabled() {
		rs.upsertSecretByHost(namespace, secretName, dockerConfig, serviceAcctName)
		return
	}

	rs.upsertSecretByName(namespace, secretName, dockerConfig, serviceAcctName)
}

func (rs *Store) upsertSecretByHost(namespace, secretName string, dockerConfig config.DockerConfig, serviceAcctName string) {
	isGlobalPullSecret := openshift.GlobalPullSecret(namespace, secretName)

	// In Kubernetes, the `default` service account always exists in each namespace (it is recreated upon deletion).
	// The default service account always contains an API token.
	// In OpenShift, the default service account also contains credentials for the
	// OpenShift Container Registry, which is an internal image registry.
	fromDefaultSA := serviceAcctName == defaultSA

	for registryAddress, dce := range dockerConfig {
		registryAddr := strings.TrimSpace(registryAddress)
		host := urlfmt.GetServerFromURL(
			urlfmt.FormatURL(registryAddr, urlfmt.HTTPS, urlfmt.NoTrailingSlash),
		)

		if fromDefaultSA {
			// Registries found in the `dockercfg` secret associated with the `default`
			// service account are assumed to reference the OCP internal registry.
			rs.addClusterLocalRegistryHost(host)
			if err := rs.upsertRegistry(namespace, registryAddr, host, dce); err != nil {
				log.Errorf("Unable to upsert registry %q into store: %v", registryAddr, err)
			}
			continue
		}

		if env.DelegatedScanningDisabled.BooleanSetting() {
			// If delegated scanning is disabled then we do not store additional secrets outside of those needed
			// for scanning images from the OCP internal registry.
			continue
		}

		if serviceAcctName != "" {
			// Ignore secrets for service accounts other than default so that the
			// default registry is not overwritten in the store.
			continue
		}

		var err error
		if isGlobalPullSecret {
			err = rs.upsertGlobalRegistry(registryAddr, host, dce)
		} else {
			err = rs.upsertRegistry(namespace, registryAddr, host, dce)
		}
		if err != nil {
			log.Errorf("unable to upsert registry %q into store: %v", registryAddr, err)
		}
	}
}

func (rs *Store) upsertSecretByName(namespace, secretName string, dockerConfig config.DockerConfig, serviceAcctName string) {
	isGlobalPullSecret := openshift.GlobalPullSecret(namespace, secretName)

	// hasBoundServiceAccount indicates that this secret is bound to a service account,
	// which means the secret is managed by OCP and its lifecycle is tied to that of
	// the associated service account (ie: if the service account is deleted so is the secret).
	hasBoundServiceAccount := serviceAcctName != ""

	// To avoid partial upserts - hold the lock until the entire secret upserted.
	rs.storeMutux.Lock()
	defer rs.storeMutux.Unlock()

	for registryAddress, dce := range dockerConfig {
		registryAddr := strings.TrimSpace(registryAddress)
		host := urlfmt.GetServerFromURL(
			urlfmt.FormatURL(registryAddr, urlfmt.HTTPS, urlfmt.NoTrailingSlash),
		)

		if hasBoundServiceAccount {
			// Registries found in any `dockercfg` secret bound a service account
			// are assumed to reference the OCP internal registry.
			rs.upsertPullSecretByNameNoLock(namespace, secretName, registryAddr, host, dce)
			rs.addClusterLocalRegistryHost(host)
			continue
		}

		if env.DelegatedScanningDisabled.BooleanSetting() {
			// If delegated scanning is disabled then we do not store additional secrets outside of those needed
			// for scanning images from the OCP internal registry.
			continue
		}

		if isGlobalPullSecret {
			if err := rs.upsertGlobalRegistry(registryAddr, host, dce); err != nil {
				log.Errorf("Unable to upsert global registry for pull secret %q, namespace %q, registry %q, host %q: %v", secretName, namespace, registryAddr, host, err)
			}
		}

		// Regardless if this secret is the global pull secret, we still store it
		// in case there is a workload that directly references it by name.
		rs.upsertPullSecretByNameNoLock(namespace, secretName, registryAddr, host, dce)
	}

	log.Debugf("Upserted %d entries from secret %q in namespace %q", len(dockerConfig), secretName, namespace)
}

func (rs *Store) upsertPullSecretByNameNoLock(namespace, secretName, registry, host string, dce config.DockerConfigEntry) {
	host = urlfmt.TrimHTTPPrefixes(host)

	name := genIntegrationName(types.PullSecretNamePrefix, namespace, secretName, registry)
	ii := createImageIntegration(host, dce, name)

	reg, err := rs.factory.CreateRegistry(ii, types.WithGCPTokenManager(gcp.Singleton()))
	if err != nil {
		log.Errorf("Creating registry for pull secret %q, namespace %q, registry %q, host %q: %v", secretName, namespace, registry, host, err)
		return
	}

	secretNameToHost, ok := rs.storeByName[namespace]
	if !ok {
		secretNameToHost = make(secretNameToHostname)
		rs.storeByName[namespace] = secretNameToHost
	}

	hostToRegistry, ok := secretNameToHost[secretName]
	if !ok {
		hostToRegistry = make(hostnameToRegistry)
		secretNameToHost[secretName] = hostToRegistry
	}

	oldreg, ok := hostToRegistry[registry]
	if !ok {
		metrics.IncrementPullSecretEntriesCount(1)
		metrics.IncrementPullSecretEntriesSize(reg.Source().SizeVT())
	} else {
		// Adjust the the size based on the diff between the old and the new entry.
		metrics.IncrementPullSecretEntriesSize(reg.Source().SizeVT() - oldreg.Source().SizeVT())
	}

	hostToRegistry[registry] = reg
}

// DeleteSecret returns true when a secret is deleted from the store, false otherwise.
func (rs *Store) DeleteSecret(namespace, secretName string) bool {
	if !features.SensorPullSecretsByName.Enabled() {
		// When storing secrets by host they cannot be deleted.
		return false
	}

	rs.storeMutux.Lock()
	defer rs.storeMutux.Unlock()

	secretNameToHost := rs.storeByName[namespace]
	if secretNameToHost == nil {
		return false
	}

	if hostToRegistry, ok := secretNameToHost[secretName]; ok {
		var deletedBytes int
		for _, reg := range hostToRegistry {
			deletedBytes += reg.Source().SizeVT()
		}

		delete(secretNameToHost, secretName)

		if len(secretNameToHost) == 0 {
			// If there are no more secrets for this namespace, delete the namespace entry as well.
			delete(rs.storeByName, namespace)
		}

		log.Debugf("Deleted secret %q from namespace %q", secretName, namespace)
		metrics.DecrementPullSecretEntriesCount(len(hostToRegistry))
		metrics.DecrementPullSecretEntriesSize(deletedBytes)
		return true
	}

	return false
}

// GetPullSecretRegistries returns the matching registries associated with the provided pull secrets found in namespace.
// If no pull secrets are provided, all matching registries from the namespace are returned.
func (rs *Store) GetPullSecretRegistries(image *storage.ImageName, namespace string, imagePullSecrets []string) ([]types.ImageRegistry, error) {
	if !features.SensorPullSecretsByName.Enabled() {
		reg, err := rs.getRegistryForImageInNamespace(image, namespace)
		if err != nil {
			return nil, err
		}

		return []types.ImageRegistry{reg}, nil
	}

	rs.storeMutux.RLock()
	defer rs.storeMutux.RUnlock()

	secretNameToHost, ok := rs.storeByName[namespace]
	if !ok {
		return nil, nil
	}

	if len(imagePullSecrets) > 0 {
		// Return matching registries referenced by the image pull secrets.
		return rs.getPullSecretRegistriesNoLock(secretNameToHost, image, imagePullSecrets), nil
	}

	// If no pull secrets were provided, we assume that all matching registries
	// from the namespace are desired (scan requests that originate from Central
	// will not have pull secrets, such as those executed via roxctl).
	return rs.getAllPullSecretRegistriesNoLock(secretNameToHost, image), nil
}

// getPullSecretRegistriesNoLock returns registries found within image pull secrets
// from a namespace that match image.
func (rs *Store) getPullSecretRegistriesNoLock(secretNameToHost secretNameToHostname, image *storage.ImageName, imagePullSecrets []string) []types.ImageRegistry {
	var regs []types.ImageRegistry
	registryHostname := image.GetRegistry()

	// Extract registries from the matching pull secrets.
	for _, secretName := range imagePullSecrets {
		for host, reg := range secretNameToHost[secretName] {
			if host == registryHostname {
				regs = append(regs, reg)
			}
		}
	}

	return regs
}

// getAllPullSecretRegistriesNoLock returns all registries within a namespace that match image.
func (rs *Store) getAllPullSecretRegistriesNoLock(secretNameToHost secretNameToHostname, image *storage.ImageName) []types.ImageRegistry {
	secretNames := make([]string, 0, len(secretNameToHost))
	for secretName := range secretNameToHost {
		secretNames = append(secretNames, secretName)
	}

	// To make the output deterministic sort the secret names.
	slices.Sort(secretNames)

	return rs.getPullSecretRegistriesNoLock(secretNameToHost, image, secretNames)
}
