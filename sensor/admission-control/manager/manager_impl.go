package manager

import (
	"sync/atomic"
	"unsafe"

	"github.com/gogo/protobuf/types"
	pkgErr "github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyfields"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/detection"
	"github.com/stackrox/rox/pkg/detection/deploytime"
	"github.com/stackrox/rox/pkg/detection/runtime"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/size"
	"github.com/stackrox/rox/pkg/sizeboundedcache"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/admission-control/errors"
	"github.com/stackrox/rox/sensor/admission-control/resources"
	"google.golang.org/grpc"
	admission "k8s.io/api/admission/v1beta1"
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
	deploytimeDetector deploytime.Detector

	allRuntimePoliciesDetector                    runtime.Detector
	runtimeDetectorForPoliciesWithoutDeployFields runtime.Detector
	runtimeDetectorForPoliciesWithDeployFields    runtime.Detector

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

func (s *state) activeForOperation(op admission.Operation) bool {
	_, active := s.enforcedOps[op]
	return active
}

type manager struct {
	stopSig    concurrency.Signal
	stoppedSig concurrency.ErrorSignal

	client     sensor.ImageServiceClient
	imageCache sizeboundedcache.Cache

	depClient        sensor.DeploymentServiceClient
	resourceUpdatesC chan *sensor.AdmCtrlUpdateResourceRequest
	namespaces       *resources.NamespaceStore
	deployments      *resources.DeploymentStore
	pods             *resources.PodStore
	initialSyncSig   concurrency.Signal

	settingsStream     *concurrency.ValueStream
	settingsC          chan *sensor.AdmissionControlSettings
	lastSettingsUpdate *types.Timestamp

	statePtr unsafe.Pointer

	cacheVersion string

	sensorConnStatus concurrency.Flag

	alertsC chan []*storage.Alert
}

func newManager(conn *grpc.ClientConn) *manager {
	cache, err := sizeboundedcache.New(200*size.MB, 2*size.MB, func(key interface{}, value interface{}) int64 {
		return int64(len(key.(string)) + value.(imageCacheEntry).Size())
	})
	utils.Must(err)

	podStore := resources.NewPodStore()
	depStore := resources.NewDeploymentStore(podStore)
	nsStore := resources.NewNamespaceStore(depStore, podStore)
	return &manager{
		settingsStream: concurrency.NewValueStream(nil),
		settingsC:      make(chan *sensor.AdmissionControlSettings),
		stoppedSig:     concurrency.NewErrorSignal(),

		client:     sensor.NewImageServiceClient(conn),
		imageCache: cache,

		alertsC: make(chan []*storage.Alert),

		namespaces:       nsStore,
		deployments:      depStore,
		pods:             podStore,
		resourceUpdatesC: make(chan *sensor.AdmCtrlUpdateResourceRequest),
		initialSyncSig:   concurrency.NewSignal(),
		depClient:        sensor.NewDeploymentServiceClient(conn),
	}
}

func (m *manager) currentState() *state {
	return (*state)(atomic.LoadPointer(&m.statePtr))
}

func (m *manager) SettingsStream() concurrency.ReadOnlyValueStream {
	return m.settingsStream
}

func (m *manager) IsReady() bool {
	return m.currentState() != nil
}

func (m *manager) Start() error {
	if !m.stopSig.Reset() {
		return pkgErr.New("admission control manager has already been started")
	}

	go m.runSettingsWatch()
	go m.runUpdateResourceReqWatch()
	return nil
}

func (m *manager) Stop() {
	m.stopSig.Signal()
}

func (m *manager) Stopped() concurrency.ErrorWaitable {
	return &m.stoppedSig
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

func (m *manager) runSettingsWatch() {
	defer m.stoppedSig.Signal()
	defer log.Info("Stopping watcher for new settings")

	for !m.stopSig.IsDone() {
		select {
		case <-m.stopSig.Done():
			m.processNewSettings(nil)
			return
		case newSettings := <-m.settingsC:
			m.processNewSettings(newSettings)
		}
	}
}

func (m *manager) runUpdateResourceReqWatch() {
	if !features.K8sEventDetection.Enabled() {
		return
	}

	defer m.stoppedSig.Signal()
	defer log.Info("Stopping watcher for new sensor events")

	for {
		select {
		case <-m.stopSig.Done():
			return
		case req := <-m.resourceUpdatesC:
			m.processUpdateResourceRequest(req)
		}
	}
}

func (m *manager) processNewSettings(newSettings *sensor.AdmissionControlSettings) {
	if newSettings == nil {
		log.Info("DISABLING admission control service (config map was deleted).")
		atomic.StorePointer(&m.statePtr, nil)
		m.lastSettingsUpdate = nil
		m.settingsStream.Push(newSettings) // typed nil ptr, not nil!
		return
	}

	if m.lastSettingsUpdate != nil && newSettings.GetTimestamp().Compare(m.lastSettingsUpdate) <= 0 {
		return // no update
	}

	deployTimePolicySet := detection.NewPolicySet()
	for _, policy := range newSettings.GetEnforcedDeployTimePolicies().GetPolicies() {
		if policyfields.ContainsUnscannedImageField(policy) && !newSettings.GetClusterConfig().GetAdmissionControllerConfig().GetScanInline() {
			log.Warnf(errors.ImageScanUnavailableMsg(policy))
			continue
		}
		if err := deployTimePolicySet.UpsertPolicy(policy); err != nil {
			log.Errorf("Unable to upsert policy %q (%s), will not be able to enforce", policy.GetName(), policy.GetId())
		}
	}

	allRuntimePolicySet := detection.NewPolicySet()
	runtimePoliciesWithDeployFields, runtimePoliciesWithoutDeployFields := detection.NewPolicySet(), detection.NewPolicySet()
	for _, policy := range newSettings.GetRuntimePolicies().GetPolicies() {
		if policyfields.ContainsUnscannedImageField(policy) && !newSettings.GetClusterConfig().GetAdmissionControllerConfig().GetScanInline() {
			log.Warnf(errors.ImageScanUnavailableMsg(policy))
			continue
		}

		if err := allRuntimePolicySet.UpsertPolicy(policy); err != nil {
			log.Errorf("Unable to upsert policy %q (%s), will not be able to detect", policy.GetName(), policy.GetId())
		}

		if booleanpolicy.ContainsDeployTimeFields(policy) {
			if err := runtimePoliciesWithDeployFields.UpsertPolicy(policy); err != nil {
				log.Errorf("Unable to upsert policy %q (%s), will not be able to detect", policy.GetName(), policy.GetId())
			}
		} else {
			if err := runtimePoliciesWithoutDeployFields.UpsertPolicy(policy); err != nil {
				log.Errorf("Unable to upsert policy %q (%s), will not be able to detect", policy.GetName(), policy.GetId())
			}
		}
		log.Debugf("Upserted policy %q (%s)", policy.GetName(), policy.GetId())
	}

	var deployTimeDetector deploytime.Detector
	if newSettings.GetClusterConfig().GetAdmissionControllerConfig().GetEnabled() && len(deployTimePolicySet.GetCompiledPolicies()) > 0 {
		deployTimeDetector = deploytime.NewDetector(deployTimePolicySet)
	}

	enforcedOperations := map[admission.Operation]struct{}{
		admission.Create: {},
	}

	if newSettings.GetClusterConfig().GetAdmissionControllerConfig().GetEnforceOnUpdates() {
		enforcedOperations[admission.Update] = struct{}{}
	}

	oldState := m.currentState()
	newState := &state{
		AdmissionControlSettings:                      newSettings,
		deploytimeDetector:                            deployTimeDetector,
		allRuntimePoliciesDetector:                    runtime.NewDetector(allRuntimePolicySet),
		runtimeDetectorForPoliciesWithDeployFields:    runtime.NewDetector(runtimePoliciesWithDeployFields),
		runtimeDetectorForPoliciesWithoutDeployFields: runtime.NewDetector(runtimePoliciesWithoutDeployFields),
		bypassForUsers:                                allowAlwaysUsers,
		bypassForGroups:                               allowAlwaysGroups,
		enforcedOps:                                   enforcedOperations,
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
			conn, err := clientconn.AuthenticatedGRPCConnection(newSettings.GetCentralEndpoint(), mtls.CentralSubject, clientconn.UseServiceCertToken(true))
			if err != nil {
				log.Errorf("Could not create connection to Central: %v", err)
			} else {
				newState.centralConn = conn
			}
		}
	}

	if newSettings.GetCacheVersion() != m.cacheVersion {
		m.imageCache.Purge()
		m.cacheVersion = newSettings.GetCacheVersion()
	}

	atomic.StorePointer(&m.statePtr, unsafe.Pointer(newState))
	if m.lastSettingsUpdate == nil {
		log.Info("RE-ENABLING admission control service")
	}
	m.lastSettingsUpdate = newSettings.GetTimestamp()

	enforceablePolicies := 0
	for _, policy := range allRuntimePolicySet.GetCompiledPolicies() {
		if len(policy.Policy().GetEnforcementActions()) > 0 {
			enforceablePolicies++
		}
	}
	log.Infof("Applied new admission control settings "+
		"(enforcing on %d deploy-time policies; "+
		"detecting on %d run-time policies; "+
		"enforcing on %d run-time policies).",
		len(deployTimePolicySet.GetCompiledPolicies()),
		len(allRuntimePolicySet.GetCompiledPolicies()),
		enforceablePolicies)

	m.settingsStream.Push(newSettings)
}

func (m *manager) HandleReview(req *admission.AdmissionRequest) (*admission.AdmissionResponse, error) {
	state := m.currentState()

	if state == nil {
		return nil, pkgErr.New("admission controller is disabled, not handling request")
	}
	return m.evaluateAdmissionRequest(state, req)
}

func (m *manager) HandleK8sEvent(req *admission.AdmissionRequest) (*admission.AdmissionResponse, error) {
	if !features.K8sEventDetection.Enabled() {
		return pass(req.UID), nil
	}

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

func (m *manager) putAlertsOnChan(alerts []*storage.Alert) {
	select {
	case <-m.stopSig.Done():
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
