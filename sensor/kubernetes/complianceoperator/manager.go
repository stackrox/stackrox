package complianceoperator

import (
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/pubsub"
)

// Manager is the sole writer of ScanSetting and ScanSettingBinding
// resources to the Kubernetes API. It multiplexes Central requests
// and MCP change events into a single goroutine to avoid races.
type Manager struct {
	complianceIsReady *concurrency.Signal
	stopper           concurrency.Stopper
	dispatcher        common.PubSubDispatcher
	inner             chan pubsub.Event
}

// NewManager creates a new Manager.
func NewManager(complianceIsReady *concurrency.Signal, dispatcher common.PubSubDispatcher) *Manager {
	return &Manager{
		complianceIsReady: complianceIsReady,
		stopper:           concurrency.NewStopper(),
		dispatcher:        dispatcher,
		inner:             make(chan pubsub.Event),
	}
}

// Start registers the pubsub consumer and launches the run loop.
func (m *Manager) Start() error {
	if err := m.dispatcher.RegisterConsumerToLane(
		pubsub.ComplianceManagerConsumer,
		pubsub.ComplianceCentralRequestTopic,
		pubsub.ComplianceCentralRequestLane,
		m.onCentralRequest,
	); err != nil {
		return err
	}
	go m.run()
	return nil
}

// Stop requests shutdown and waits for the run loop to finish.
func (m *Manager) Stop() {
	if !m.stopper.Client().Stopped().IsDone() {
		defer func() {
			_ = m.stopper.Client().Stopped().Wait()
		}()
	}
	m.stopper.Client().Stop()
}

func (m *Manager) onCentralRequest(ev pubsub.Event) error {
	select {
	case m.inner <- ev:
		return nil
	case <-m.stopper.Flow().StopRequested():
		return nil
	}
}

func (m *Manager) run() {
	defer m.stopper.Flow().ReportStopped()

	select {
	case <-m.complianceIsReady.Done():
	case <-m.stopper.Flow().StopRequested():
		return
	}

	if centralcaps.Has(centralsensor.ComplianceV2ScanConfigSync) {
		m.waitAndProcessSync()
	}

	for {
		select {
		case ev := <-m.inner:
			m.process(ev)
		case <-m.stopper.Flow().StopRequested():
			return
		}
	}
}

func (m *Manager) waitAndProcessSync() {
	select {
	case ev := <-m.inner:
		if err := m.processSync(ev); err != nil {
			log.Errorf("Failed to process scan config sync: %v", err)
		}
	case <-m.stopper.Flow().StopRequested():
	}
}

func (m *Manager) processSync(_ pubsub.Event) error {
	// TODO: reconcile all scan configs from Central.
	return nil
}

func (m *Manager) process(_ pubsub.Event) {
	// TODO: switch on concrete event types to perform K8s operations.
}
