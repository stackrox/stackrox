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
	"github.com/stackrox/rox/pkg/logging"
	deploymentOptions "github.com/stackrox/rox/pkg/search/options/deployments"
	"github.com/stackrox/rox/pkg/searchbasedpolicies/matcher"
)

var (
	builder = matcher.NewBuilder(
		matcher.NewRegistry(
			nil,
		),
		deploymentOptions.OptionsMap,
	)

	log = logging.LoggerForModule()
)

type settingsAndDetector struct {
	settings *sensor.AdmissionControlSettings
	detector deploytime.Detector
}

type manager struct {
	stopSig    concurrency.Signal
	stoppedSig concurrency.ErrorSignal

	settingsC          chan *sensor.AdmissionControlSettings
	lastSettingsUpdate *types.Timestamp

	settingsAndDetectorPtr unsafe.Pointer
}

func newManager() *manager {
	return &manager{
		settingsC:  make(chan *sensor.AdmissionControlSettings),
		stoppedSig: concurrency.NewErrorSignal(),
	}
}

func (m *manager) currentSettingsAndDetector() *settingsAndDetector {
	return (*settingsAndDetector)(atomic.LoadPointer(&m.settingsAndDetectorPtr))
}

func (m *manager) IsReady() bool {
	return m.currentSettingsAndDetector() != nil
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
		atomic.StorePointer(&m.settingsAndDetectorPtr, nil)
		m.lastSettingsUpdate = nil
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

	detector := deploytime.NewDetector(policySet)

	newSettingsAndDetector := &settingsAndDetector{
		settings: newSettings,
		detector: detector,
	}

	atomic.StorePointer(&m.settingsAndDetectorPtr, unsafe.Pointer(newSettingsAndDetector))
	if m.lastSettingsUpdate == nil {
		log.Info("RE-ENABLING admission control service")
	}
	m.lastSettingsUpdate = newSettings.GetTimestamp()

	log.Infof("Applied new admission control settings (enforcing on %d policies).", len(newSettings.GetEnforcedDeployTimePolicies().GetPolicies()))
}
