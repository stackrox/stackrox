package manager

import (
	"sync/atomic"
	"unsafe"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/detection"
	"github.com/stackrox/rox/pkg/detection/deploytime"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/size"
	"github.com/stackrox/rox/pkg/sizeboundedcache"
	"github.com/stackrox/rox/pkg/utils"
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
	detector deploytime.Detector

	bypassForUsers, bypassForGroups set.FrozenStringSet
	enforcedOps                     map[admission.Operation]struct{}
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
		return errors.New("admission control manager has already been started")
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

	policySet := detection.NewPolicySet(detection.NewPolicyCompiler(builder))
	for _, policy := range newSettings.GetEnforcedDeployTimePolicies().GetPolicies() {
		if err := policySet.UpsertPolicy(policy); err != nil {
			log.Errorf("Unable to upsert policy %s (%s), will not be able to enforce", policy.GetName(), policy.GetId())
		}
	}

	var detector deploytime.Detector
	if newSettings.GetClusterConfig().GetAdmissionControllerConfig().GetEnabled() && len(policySet.GetCompiledPolicies()) > 0 {
		detector = deploytime.NewDetector(policySet)
	}

	enforcedOperations := map[admission.Operation]struct{}{
		admission.Create: {},
	}

	if features.AdmissionControlEnforceOnUpdate.Enabled() && newSettings.GetClusterConfig().GetAdmissionControllerConfig().GetEnforceOnUpdates() {
		enforcedOperations[admission.Update] = struct{}{}
	}

	newSettingsAndDetector := &state{
		AdmissionControlSettings: newSettings,
		detector:                 detector,
		bypassForUsers:           allowAlwaysUsers,
		bypassForGroups:          allowAlwaysGroups,
		enforcedOps:              enforcedOperations,
	}

	atomic.StorePointer(&m.statePtr, unsafe.Pointer(newSettingsAndDetector))
	if m.lastSettingsUpdate == nil {
		log.Info("RE-ENABLING admission control service")
	}
	m.lastSettingsUpdate = newSettings.GetTimestamp()

	log.Infof("Applied new admission control settings (enforcing on %d policies).", len(newSettings.GetEnforcedDeployTimePolicies().GetPolicies()))
	m.settingsStream.Push(newSettings)
}

func (m *manager) HandleReview(req *admission.AdmissionRequest) (*admission.AdmissionResponse, error) {
	state := m.currentState()

	if state == nil {
		return nil, errors.New("admission controller is disabled, not handling request")
	}
	return m.evaluateAdmissionRequest(state, req)
}
