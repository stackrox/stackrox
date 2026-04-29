package manager

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	pkgErr "github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyfields"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/coalescer"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/detection"
	"github.com/stackrox/rox/pkg/detection/deploytime"
	"github.com/stackrox/rox/pkg/detection/runtime"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/size"
	"github.com/stackrox/rox/pkg/sizeboundedcache"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/admission-control/errors"
	"github.com/stackrox/rox/sensor/admission-control/resources"
	"google.golang.org/grpc"
	admission "k8s.io/api/admission/v1"
)

var (
	log = logging.LoggerForModule()

	allowAlwaysUsers = set.NewFrozenStringSet(
		"system:kube-scheduler",
		"system:kube-controller-manager",
		"system:kube-proxy",
	)

	allowAlwaysGroups = set.NewFrozenStringSet(
		"system:nodes",
	)
)

type state struct {
	*sensor.AdmissionControlSettings

	// specOnlyDeployDetector evaluates deploy policies that only reference
	// deployment spec fields (privileged, capabilities, labels, etc.) and
	// can produce a review response without requiring image enrichment data.
	specOnlyDeployDetector deploytime.Detector

	// enrichmentRequiredDeployDetector evaluates deploy policies that require image
	// enrichment data (scan results, image metadata, signatures).
	enrichmentRequiredDeployDetector deploytime.Detector

	allK8sEventDetector    runtime.Detector
	deployFieldK8sDetector runtime.Detector
	eventOnlyK8sDetector   runtime.Detector

	bypassForUsers, bypassForGroups set.FrozenStringSet

	centralConn *grpc.ClientConn
}

func (s *state) clusterID() string {
	clusterID := s.GetClusterId()
	if clusterID == "" {
		clusterID = getClusterID()
	}
	return clusterID
}

func (s *state) admissionTimeoutCtx() (context.Context, context.CancelFunc) {
	timeout := s.GetClusterConfig().GetAdmissionControllerConfig().GetTimeoutSeconds()
	return context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
}

const (
	// imageNameCacheSize is the maximum number of entries in the image name-to-cache-key
	// LRU. Each entry maps a full image name (e.g. "docker.io/library/nginx:1.25") to its
	// resolved cache key in imageCache. 8192 covers large clusters with aggressive CI/CD
	// while bounding memory to ~1.6MB. Dead entries (old tags never referenced again) are
	// naturally evicted by LRU pressure from new entries.
	imageNameCacheSize = 8192
)

// imageGenTracker tracks per-key generation counters to prevent stale in-flight
// fetches from re-caching data after an invalidation or full purge.
type imageGenTracker struct {
	mu           sync.RWMutex
	gen          map[string]uint64
	cacheVersion string
}

func newImageGenTracker() *imageGenTracker {
	return &imageGenTracker{gen: make(map[string]uint64)}
}

// Snapshot captures the per-key generation and CacheVersion before a fetch.
func (t *imageGenTracker) Snapshot(key string) (gen uint64, cacheVersion string) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.gen[key], t.cacheVersion
}

// Changed returns true if the generation or CacheVersion moved since the snapshot.
func (t *imageGenTracker) Changed(key string, gen uint64, cacheVersion string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.gen[key] != gen || t.cacheVersion != cacheVersion
}

// Inc bumps the per-key generation counter.
func (t *imageGenTracker) Inc(key string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.gen[key]++
}

// CacheVersion returns the current cache version.
func (t *imageGenTracker) CacheVersion() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.cacheVersion
}

// UpdateCacheVersion sets a new CacheVersion and clears all per-key counters.
func (t *imageGenTracker) UpdateCacheVersion(v string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cacheVersion = v
	clear(t.gen)
}

// Get returns the per-key generation counter (for testing only).
func (t *imageGenTracker) Get(_ testing.TB, key string) uint64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.gen[key]
}

// Clear resets all per-key counters and the cache version (for testing only).
func (t *imageGenTracker) Clear(_ testing.TB) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.cacheVersion = ""
	clear(t.gen)
}

type manager struct {
	stopper concurrency.Stopper

	client        sensor.ImageServiceClient
	imageCache    sizeboundedcache.Cache[string, imageCacheEntry]
	imageCacheGen *imageGenTracker
	// imageNameToImageCacheKey resolves image full names (e.g. "docker.io/library/nginx:1.25")
	// to their cache keys in imageCache. This is needed because admission requests for CREATE/UPDATE
	// operations only contain image names (no digest/ID), so imageKey() returns the full name as the
	// cache key. After a scan, the cache stores the result under the image's resolved digest. Without this
	// map, subsequent requests for the same image by name would miss the cache and trigger redundant scans.
	// Bounded by imageNameCacheSize to prevent unbounded growth during long-lived Sensor sessions.
	imageNameToImageCacheKey *lru.Cache[string, string]
	imageNameCacheEnabled    bool
	imageCacheTTL            time.Duration
	imageFetchGroup          *coalescer.Coalescer[*storage.Image]

	depClient               sensor.DeploymentServiceClient
	resourceUpdatesC        chan *sensor.AdmCtrlUpdateResourceRequest
	imageCacheInvalidationC chan *sensor.AdmCtrlImageCacheInvalidation
	namespaces              *resources.NamespaceStore
	deployments             *resources.DeploymentStore
	pods                    *resources.PodStore
	initialSyncSig          concurrency.Signal

	settingsStream     *concurrency.ValueStream[*sensor.AdmissionControlSettings]
	settingsC          chan *sensor.AdmissionControlSettings
	lastSettingsUpdate *time.Time

	syncC chan *concurrency.Signal

	state         atomic.Pointer[state]
	clusterLabels atomic.Pointer[map[string]string]

	sensorConnStatus concurrency.Flag

	alertsC chan []*storage.Alert

	ownNamespace string
}

// NewManager creates a new manager
func NewManager(namespace string, maxImageCacheSize int64, imageNameCacheEnabled bool, imageServiceClient sensor.ImageServiceClient, deploymentServiceClient sensor.DeploymentServiceClient) *manager {
	cache, err := sizeboundedcache.New(maxImageCacheSize, 2*size.MB, func(key string, value imageCacheEntry) int64 {
		return int64(len(key) + value.SizeVT())
	})
	utils.CrashOnError(err)

	nameCache, err := lru.New[string, string](imageNameCacheSize)
	utils.CrashOnError(err)

	podStore := resources.NewPodStore()
	depStore := resources.NewDeploymentStore(podStore)
	nsStore := resources.NewNamespaceStore(depStore, podStore)
	return &manager{
		settingsStream: concurrency.NewValueStream[*sensor.AdmissionControlSettings](nil),
		settingsC:      make(chan *sensor.AdmissionControlSettings),
		stopper:        concurrency.NewStopper(),
		syncC:          make(chan *concurrency.Signal),

		client:                   imageServiceClient,
		imageCache:               cache,
		imageNameToImageCacheKey: nameCache,
		imageNameCacheEnabled:    imageNameCacheEnabled,
		imageCacheTTL:            env.AdmissionControlImageCacheTTL.DurationSetting(),
		imageFetchGroup:          coalescer.New[*storage.Image](),
		imageCacheGen:            newImageGenTracker(),

		alertsC: make(chan []*storage.Alert),

		namespaces:              nsStore,
		deployments:             depStore,
		pods:                    podStore,
		resourceUpdatesC:        make(chan *sensor.AdmCtrlUpdateResourceRequest),
		imageCacheInvalidationC: make(chan *sensor.AdmCtrlImageCacheInvalidation),
		initialSyncSig:          concurrency.NewSignal(),
		depClient:               deploymentServiceClient,

		ownNamespace: namespace,
	}
}

func (m *manager) currentState() *state {
	return m.state.Load()
}

func (m *manager) SettingsStream() concurrency.ReadOnlyValueStream[*sensor.AdmissionControlSettings] {
	return m.settingsStream
}

// GetClusterLabels implements scopecomp.ClusterLabelProvider interface.
func (m *manager) GetClusterLabels(_ context.Context, _ string) (map[string]string, error) {
	labels := m.clusterLabels.Load()
	if labels == nil {
		return nil, nil
	}
	return *labels, nil
}

// GetNamespaceLabels implements scopecomp.NamespaceLabelProvider interface.
func (m *manager) GetNamespaceLabels(ctx context.Context, clusterID string, namespaceName string) (map[string]string, error) {
	labels, err := m.namespaces.GetNamespaceLabels(ctx, clusterID, namespaceName)
	if err != nil {
		return nil, pkgErr.Wrapf(err, "getting namespace labels for %q", namespaceName)
	}
	return labels, nil
}

func (m *manager) IsReady() bool {
	return m.currentState() != nil
}

func (m *manager) Sync(ctx context.Context) error {
	syncSig := concurrency.NewSignal()
	select {
	case m.syncC <- &syncSig:
	case <-ctx.Done():
		return pkgErr.Wrap(ctx.Err(), "sync")
	case <-m.stopper.Client().Stopped().Done():
		return pkgErr.Wrap(
			m.stopper.Client().Stopped().ErrorWithDefault(pkgErr.New("manager was stopped")),
			"sync",
		)
	}

	select {
	case <-syncSig.Done():
		return nil
	case <-ctx.Done():
		return pkgErr.Wrap(ctx.Err(), "syncing")
	case <-m.stopper.Client().Stopped().Done():
		return pkgErr.Wrap(
			m.stopper.Client().Stopped().ErrorWithDefault(pkgErr.New("manager was stopped")),
			"sync",
		)
	}
}

func (m *manager) Start() {
	go m.run()
}

func (m *manager) Stop() {
	m.stopper.Client().Stop()
}

func (m *manager) Stopped() concurrency.ErrorWaitable {
	return m.stopper.Client().Stopped()
}

func (m *manager) SettingsUpdateC() chan<- *sensor.AdmissionControlSettings {
	return m.settingsC
}

func (m *manager) ResourceUpdatesC() chan<- *sensor.AdmCtrlUpdateResourceRequest {
	return m.resourceUpdatesC
}

func (m *manager) ImageCacheInvalidationC() chan<- *sensor.AdmCtrlImageCacheInvalidation {
	return m.imageCacheInvalidationC
}

func (m *manager) InitialResourceSyncSig() *concurrency.Signal {
	return &m.initialSyncSig
}

func (m *manager) run() {
	defer m.stopper.Flow().ReportStopped()
	defer log.Info("Stopping watcher")

	for {
		select {
		case <-m.stopper.Flow().StopRequested():
			m.ProcessNewSettings(nil)
			return
		case newSettings := <-m.settingsC:
			m.ProcessNewSettings(newSettings)
		case req := <-m.resourceUpdatesC:
			m.processUpdateResourceRequest(req)
		case inv := <-m.imageCacheInvalidationC:
			m.processImageCacheInvalidation(inv)
		default:
			// Select on syncC only if there is nothing to be read from the main
			// channels. The duplication of select branches is a bit ugly, but inevitable
			// without reflection.
			select {
			case <-m.stopper.Flow().StopRequested():
				m.ProcessNewSettings(nil)
				return
			case newSettings := <-m.settingsC:
				m.ProcessNewSettings(newSettings)
			case req := <-m.resourceUpdatesC:
				m.processUpdateResourceRequest(req)
			case inv := <-m.imageCacheInvalidationC:
				m.processImageCacheInvalidation(inv)
			case syncSig := <-m.syncC:
				syncSig.Signal()
			}
		}
	}
}

// ProcessNewSettings processes new settings
func (m *manager) ProcessNewSettings(newSettings *sensor.AdmissionControlSettings) {
	if newSettings == nil {
		log.Info("DISABLING admission control service (config map was deleted).")
		m.state.Store(nil)
		m.lastSettingsUpdate = nil
		m.settingsStream.Push(newSettings) // typed nil ptr, not nil!
		return
	}

	if m.lastSettingsUpdate != nil && protocompat.CompareTimestampToTime(newSettings.GetTimestamp(), m.lastSettingsUpdate) <= 0 {
		return // no update
	}

	// Manager implements both ClusterLabelProvider and NamespaceLabelProvider interfaces.
	allK8sEventPolicies := detection.NewPolicySet(m, m)
	deployFieldK8sEventPolicies, k8sEventOnlyPolicies := detection.NewPolicySet(m, m), detection.NewPolicySet(m, m)
	for _, policy := range newSettings.GetRuntimePolicies().GetPolicies() {
		if policyfields.AlertsOnMissingEnrichment(policy) && !newSettings.GetClusterConfig().GetAdmissionControllerConfig().GetScanInline() {
			log.Warn(errors.ImageScanUnavailableMsg(policy))
			continue
		}

		if err := allK8sEventPolicies.UpsertPolicy(policy); err != nil {
			log.Errorf("Unable to upsert policy %q (%s), will not be able to detect", policy.GetName(), policy.GetId())
		}

		if booleanpolicy.ContainsDeployTimeFields(policy) {
			if err := deployFieldK8sEventPolicies.UpsertPolicy(policy); err != nil {
				log.Errorf("Unable to upsert policy %q (%s), will not be able to detect", policy.GetName(), policy.GetId())
			}
		} else {
			if err := k8sEventOnlyPolicies.UpsertPolicy(policy); err != nil {
				log.Errorf("Unable to upsert policy %q (%s), will not be able to detect", policy.GetName(), policy.GetId())
			}
		}
		log.Debugf("Upserted policy %q (%s)", policy.GetName(), policy.GetId())
	}

	// Since 4.9, enforcement is controlled by a single "enforce" toggle that
	// sets both Enabled and EnforceOnUpdates identically. The webhook is only
	// registered when enforcement is on, and always for both CREATE and UPDATE.
	// We use GetEnabled() as the canonical gate; GetEnforceOnUpdates() is
	// no longer checked separately.
	enforce := newSettings.GetClusterConfig().GetAdmissionControllerConfig().GetEnabled()

	// Manager implements both ClusterLabelProvider and NamespaceLabelProvider interfaces.
	specOnlyPolicies := detection.NewPolicySet(m, m)
	enrichmentRequiredPolicies := detection.NewPolicySet(m, m)
	if enforce {
		for _, policy := range newSettings.GetEnforcedDeployTimePolicies().GetPolicies() {
			if policyfields.AlertsOnMissingEnrichment(policy) &&
				!newSettings.GetClusterConfig().GetAdmissionControllerConfig().GetScanInline() {
				log.Warn(errors.ImageScanUnavailableMsg(policy))
				continue
			}
			compiled, err := detection.CompilePolicy(policy, m, m)
			if err != nil {
				log.Errorf("Unable to compile policy %q (%s): %v", policy.GetName(), policy.GetId(), err)
				continue
			}
			if compiled.RequiresImageEnrichment() {
				enrichmentRequiredPolicies.UpsertCompiledPolicy(compiled)
			} else {
				specOnlyPolicies.UpsertCompiledPolicy(compiled)
			}
		}
	}

	oldState := m.currentState()
	newState := &state{
		AdmissionControlSettings:         newSettings,
		specOnlyDeployDetector:           deploytime.NewDetector(specOnlyPolicies),
		enrichmentRequiredDeployDetector: deploytime.NewDetector(enrichmentRequiredPolicies),
		allK8sEventDetector:              runtime.NewDetector(allK8sEventPolicies),
		deployFieldK8sDetector:           runtime.NewDetector(deployFieldK8sEventPolicies),
		eventOnlyK8sDetector:             runtime.NewDetector(k8sEventOnlyPolicies),
		bypassForUsers:                   allowAlwaysUsers,
		bypassForGroups:                  allowAlwaysGroups,
	}

	if oldState != nil && newSettings.GetCentralEndpoint() == oldState.GetCentralEndpoint() {
		newState.centralConn = oldState.centralConn
	} else {
		if oldState != nil && oldState.centralConn != nil {
			// This *should* be non-blocking, but that's not documented, so move to a goroutine to be on the safe
			// side.
			go func() {
				if err := oldState.centralConn.Close(); err != nil {
					log.Warnf("Error closing previous connection to Central after change of central endpoint: %v", err)
				}
			}()
		}

		if newSettings.GetCentralEndpoint() != "" {
			conn, err := clientconn.AuthenticatedGRPCConnection(context.Background(), newSettings.GetCentralEndpoint(), mtls.CentralSubject, clientconn.UseServiceCertToken(true))
			if err != nil {
				log.Errorf("Could not create connection to Central: %v", err)
			} else {
				newState.centralConn = conn
			}
		}
	}

	if newSettings.GetCacheVersion() != m.imageCacheGen.CacheVersion() {
		log.Infof("CacheVersion changed (%s -> %s): purging image cache and resetting gen counters",
			m.imageCacheGen.CacheVersion(), newSettings.GetCacheVersion())
		// UpdateCacheVersion must precede Purge so that any in-flight fetch that
		// snapshotted the old cacheVersion sees the mismatch in Changed() and
		// discards its result instead of re-caching stale data.
		m.imageCacheGen.UpdateCacheVersion(newSettings.GetCacheVersion())
		m.imageCache.Purge()
		m.imageNameToImageCacheKey.Purge()
	}

	m.state.Store(newState)
	if m.lastSettingsUpdate == nil {
		log.Info("RE-ENABLING admission control service")
	}
	m.lastSettingsUpdate = protocompat.ConvertTimestampToTimeOrNil(newSettings.GetTimestamp())

	enforceablePolicies := 0
	for _, policy := range allK8sEventPolicies.GetCompiledPolicies() {
		if len(policy.Policy().GetEnforcementActions()) > 0 {
			enforceablePolicies++
		}
	}
	log.Infof("Applied new admission control settings "+
		"(enforcing on %d deploy-time policies: %d deployment metadata only, %d image enrichment data required; "+
		"detecting on %d run-time policies; "+
		"enforcing on %d run-time policies).",
		len(specOnlyPolicies.GetCompiledPolicies())+len(enrichmentRequiredPolicies.GetCompiledPolicies()),
		len(specOnlyPolicies.GetCompiledPolicies()),
		len(enrichmentRequiredPolicies.GetCompiledPolicies()),
		len(allK8sEventPolicies.GetCompiledPolicies()),
		enforceablePolicies)

	m.settingsStream.Push(newSettings)
}

func (m *manager) HandleValidate(req *admission.AdmissionRequest) (*admission.AdmissionResponse, error) {
	state := m.currentState()

	if state == nil {
		return nil, pkgErr.New("admission controller is disabled, not handling request")
	}

	return m.evaluateAdmissionRequest(state, req)
}

func (m *manager) HandleK8sEvent(req *admission.AdmissionRequest) (*admission.AdmissionResponse, error) {
	state := m.currentState()
	if state == nil {
		return nil, pkgErr.New("admission controller is disabled, not handling request")
	}

	return m.evaluateRuntimeAdmissionRequest(state, req)
}

func (m *manager) SensorConnStatusFlag() *concurrency.Flag {
	return &m.sensorConnStatus
}

func (m *manager) Alerts() <-chan []*storage.Alert {
	return m.alertsC
}

func (m *manager) filterAndPutAttemptedAlertsOnChan(op admission.Operation, alerts ...*storage.Alert) {
	var filtered []*storage.Alert
	for _, alert := range alerts {
		if alert.GetDeployment() == nil {
			continue
		}

		if alert.GetEnforcement() == nil {
			continue
		}

		// Update enforcement for deploy time policy enforcements.
		if op == admission.Create {
			alert.GetDeployment().Inactive = true
			alert.Enforcement = &storage.Alert_Enforcement{
				Action:  storage.EnforcementAction_FAIL_DEPLOYMENT_CREATE_ENFORCEMENT,
				Message: "Failed deployment create in response to this policy violation.",
			}
		} else if op == admission.Update {
			alert.Enforcement = &storage.Alert_Enforcement{
				Action:  storage.EnforcementAction_FAIL_DEPLOYMENT_UPDATE_ENFORCEMENT,
				Message: "Failed deployment update in response to this policy violation.",
			}
		}

		alert.State = storage.ViolationState_ATTEMPTED

		filtered = append(filtered, alert)
	}

	if len(filtered) > 0 {
		go m.putAlertsOnChan(filtered)
	}
}

func (m *manager) putAlertsOnChan(alerts []*storage.Alert) {
	select {
	case <-m.stopper.Flow().StopRequested():
		return
	case m.alertsC <- alerts:
	}
}

func (m *manager) processUpdateResourceRequest(req *sensor.AdmCtrlUpdateResourceRequest) {
	switch req.GetResource().(type) {
	case *sensor.AdmCtrlUpdateResourceRequest_Synced:
		m.initialSyncSig.Signal()
		log.Info("Initial resource sync with Sensor complete")
	case *sensor.AdmCtrlUpdateResourceRequest_Deployment:
		m.deployments.ProcessEvent(req.GetAction(), req.GetDeployment())
	case *sensor.AdmCtrlUpdateResourceRequest_Pod:
		m.pods.ProcessEvent(req.GetAction(), req.GetPod())
	case *sensor.AdmCtrlUpdateResourceRequest_Namespace:
		m.namespaces.ProcessEvent(req.GetAction(), req.GetNamespace())
	case *sensor.AdmCtrlUpdateResourceRequest_ClusterLabels:
		labels := req.GetClusterLabels().GetLabels()
		m.clusterLabels.Store(&labels)
		log.Infof("Updated cluster labels: %v", labels)
	default:
		log.Warnf("Received message of unknown type %T from sensor, not sure what to do with it ...", m)
	}
}

func (m *manager) getDeploymentForPod(namespace, podName string) *storage.Deployment {
	return m.deployments.Get(namespace, m.pods.GetDeploymentID(namespace, podName))
}

// processImageCacheInvalidation removes targeted entries from the image cache
// and bumps generation counters for both the digest key and full name to
// prevent in-flight fetches from re-caching stale data.
func (m *manager) processImageCacheInvalidation(inv *sensor.AdmCtrlImageCacheInvalidation) {
	s := m.currentState()
	flatten := s != nil && s.GetFlattenImageData()

	invalidated := 0
	for _, key := range inv.GetImageKeys() {
		cacheKey := key.GetImageId()
		if flatten && key.GetImageIdV2() != "" {
			cacheKey = key.GetImageIdV2()
		}
		fullName := key.GetImageFullName()

		if cacheKey != "" {
			m.imageCacheGen.Inc(cacheKey)
			m.imageCache.Remove(cacheKey)
			m.imageFetchGroup.Forget(cacheKey)
			invalidated++
		}
		if fullName != "" {
			m.imageCacheGen.Inc(fullName)
			m.imageNameToImageCacheKey.Remove(fullName)
			m.imageFetchGroup.Forget(fullName)
		}
	}

	log.Infof("Targeted image cache invalidation: invalidated %d entries", invalidated)
}
