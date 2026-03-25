package manager

import (
	"context"
	"sync/atomic"
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
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/size"
	"github.com/stackrox/rox/pkg/sizeboundedcache"
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
	enforcedOps                     map[admission.Operation]struct{}

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

func (s *state) activeForOperation(op admission.Operation) bool {
	_, active := s.enforcedOps[op]
	return active
}

const (
	// imageNameCacheSize is the maximum number of entries in the image name-to-cache-key
	// LRU. Each entry maps a full image name (e.g. "docker.io/library/nginx:1.25") to its
	// resolved cache key in imageCache. 8192 covers large clusters with aggressive CI/CD
	// while bounding memory to ~1.6MB. Dead entries (old tags never referenced again) are
	// naturally evicted by LRU pressure from new entries.
	imageNameCacheSize = 8192
)

type manager struct {
	stopper concurrency.Stopper

	client     sensor.ImageServiceClient
	imageCache sizeboundedcache.Cache[string, imageCacheEntry]
	// imageNameToImageCacheKey resolves image full names (e.g. "docker.io/library/nginx:1.25")
	// to their cache keys in imageCache. This is needed because admission requests for CREATE/UPDATE
	// operations only contain image names (no digest/ID), so imageKey() returns the full name as the
	// cache key. After a scan, the cache stores the result under the image's resolved digest. Without this
	// map, subsequent requests for the same image by name would miss the cache and trigger redundant scans.
	// Bounded by imageNameCacheSize to prevent unbounded growth during long-lived Sensor sessions.
	imageNameToImageCacheKey *lru.Cache[string, string]
	imageNameCacheEnabled    bool
	imageFetchGroup          *coalescer.Coalescer[*storage.Image]

	depClient        sensor.DeploymentServiceClient
	resourceUpdatesC chan *sensor.AdmCtrlUpdateResourceRequest
	namespaces       *resources.NamespaceStore
	deployments      *resources.DeploymentStore
	pods             *resources.PodStore
	initialSyncSig   concurrency.Signal

	settingsStream     *concurrency.ValueStream[*sensor.AdmissionControlSettings]
	settingsC          chan *sensor.AdmissionControlSettings
	lastSettingsUpdate *time.Time

	syncC chan *concurrency.Signal

	state atomic.Pointer[state]

	cacheVersion string

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
		imageFetchGroup:          coalescer.New[*storage.Image](),

		alertsC: make(chan []*storage.Alert),

		namespaces:       nsStore,
		deployments:      depStore,
		pods:             podStore,
		resourceUpdatesC: make(chan *sensor.AdmCtrlUpdateResourceRequest),
		initialSyncSig:   concurrency.NewSignal(),
		depClient:        deploymentServiceClient,

		ownNamespace: namespace,
	}
}

func (m *manager) currentState() *state {
	return m.state.Load()
}

func (m *manager) SettingsStream() concurrency.ReadOnlyValueStream[*sensor.AdmissionControlSettings] {
	return m.settingsStream
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

	// TODO(ROX-33188): Wire cluster and namespace label providers.
	// For now, passing nil providers means policies with cluster_label/namespace_label scopes will
	// fail closed (not match) in admission control.
	allK8sEventPolicies := detection.NewPolicySet(nil, nil)
	deployFieldK8sEventPolicies, k8sEventOnlyPolicies := detection.NewPolicySet(nil, nil), detection.NewPolicySet(nil, nil)
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

	enforceOnCreates := newSettings.GetClusterConfig().GetAdmissionControllerConfig().GetEnabled()
	enforceOnUpdates := newSettings.GetClusterConfig().GetAdmissionControllerConfig().GetEnforceOnUpdates()

	specOnlyPolicies := detection.NewPolicySet(nil, nil)
	enrichmentRequiredPolicies := detection.NewPolicySet(nil, nil)
	if enforceOnCreates || enforceOnUpdates {
		for _, policy := range newSettings.GetEnforcedDeployTimePolicies().GetPolicies() {
			if policyfields.AlertsOnMissingEnrichment(policy) &&
				!newSettings.GetClusterConfig().GetAdmissionControllerConfig().GetScanInline() {
				log.Warn(errors.ImageScanUnavailableMsg(policy))
				continue
			}
			compiled, err := detection.CompilePolicy(policy, nil, nil)
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

	enforcedOperations := make(map[admission.Operation]struct{})
	if enforceOnCreates {
		enforcedOperations[admission.Create] = struct{}{}
	}

	if enforceOnUpdates {
		enforcedOperations[admission.Update] = struct{}{}
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
		enforcedOps:                      enforcedOperations,
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

	if newSettings.GetCacheVersion() != m.cacheVersion {
		m.imageCache.Purge()
		m.imageNameToImageCacheKey.Purge()
		m.cacheVersion = newSettings.GetCacheVersion()
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
	default:
		log.Warnf("Received message of unknown type %T from sensor, not sure what to do with it ...", m)
	}
}

func (m *manager) getDeploymentForPod(namespace, podName string) *storage.Deployment {
	return m.deployments.Get(namespace, m.pods.GetDeploymentID(namespace, podName))
}
