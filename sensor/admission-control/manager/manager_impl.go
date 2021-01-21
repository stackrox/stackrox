package manager

import (
	"sync/atomic"
	"unsafe"

	"github.com/gogo/protobuf/types"
	pkgErr "github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
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
	runtimeDetector    runtime.Detector

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

	settingsStream *concurrency.ValueStream

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

	return &manager{
		settingsStream: concurrency.NewValueStream(nil),
		settingsC:      make(chan *sensor.AdmissionControlSettings),
		stoppedSig:     concurrency.NewErrorSignal(),

		client:     sensor.NewImageServiceClient(conn),
		imageCache: cache,

		alertsC: make(chan []*storage.Alert),
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

	policySet := detection.NewPolicySet()
	for _, policy := range newSettings.GetEnforcedDeployTimePolicies().GetPolicies() {
		if policyfields.ContainsUnscannedImageField(policy) && !newSettings.GetClusterConfig().GetAdmissionControllerConfig().GetScanInline() {
			log.Warnf(errors.ImageScanUnavailableMsg(policy))
			continue
		}
		if err := policySet.UpsertPolicy(policy); err != nil {
			log.Errorf("Unable to upsert policy %q (%s), will not be able to enforce", policy.GetName(), policy.GetId())
		}
	}

	runtimePolicySet := detection.NewPolicySet()
	for _, policy := range newSettings.GetRuntimePolicies().GetPolicies() {
		if policyfields.ContainsUnscannedImageField(policy) && !newSettings.GetClusterConfig().GetAdmissionControllerConfig().GetScanInline() {
			log.Warnf(errors.ImageScanUnavailableMsg(policy))
			continue
		}
		if err := runtimePolicySet.UpsertPolicy(policy); err != nil {
			log.Errorf("Unable to upsert policy %q (%s), will not be able to detect", policy.GetName(), policy.GetId())
		}
		log.Debugf("Upserted policy %q (%s)", policy.GetName(), policy.GetId())
	}

	var deployTimeDetector deploytime.Detector
	if newSettings.GetClusterConfig().GetAdmissionControllerConfig().GetEnabled() && len(policySet.GetCompiledPolicies()) > 0 {
		deployTimeDetector = deploytime.NewDetector(policySet)
	}

	enforcedOperations := map[admission.Operation]struct{}{
		admission.Create: {},
	}

	if newSettings.GetClusterConfig().GetAdmissionControllerConfig().GetEnforceOnUpdates() {
		enforcedOperations[admission.Update] = struct{}{}
	}

	oldState := m.currentState()
	newState := &state{
		AdmissionControlSettings: newSettings,
		deploytimeDetector:       deployTimeDetector,
		runtimeDetector:          runtime.NewDetector(runtimePolicySet),
		bypassForUsers:           allowAlwaysUsers,
		bypassForGroups:          allowAlwaysGroups,
		enforcedOps:              enforcedOperations,
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

	log.Infof("Applied new admission control settings (enforcing on %d policies).", len(policySet.GetCompiledPolicies()))
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
	//TODO add logic to process k8s events here
	return pass(req.UID), nil

}

func (m *manager) SensorConnStatusFlag() *concurrency.Flag {
	return &m.sensorConnStatus
}

func (m *manager) Alerts() <-chan []*storage.Alert {
	return m.alertsC
}
