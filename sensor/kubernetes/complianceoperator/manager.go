package complianceoperator

import (
	"github.com/stackrox/rox/pkg/concurrency"
)

// Manager is the sole writer of ScanSetting and ScanSettingBinding
// resources to the Kubernetes API. It multiplexes Central requests
// and MCP change events into a single goroutine to avoid races.
type Manager struct {
	complianceIsReady *concurrency.Signal
	stopper           concurrency.Stopper
}

// NewManager creates a new Manager.
func NewManager(complianceIsReady *concurrency.Signal) *Manager {
	return &Manager{
		complianceIsReady: complianceIsReady,
		stopper:           concurrency.NewStopper(),
	}
}

// Start launches the manager's run loop.
func (m *Manager) Start() error {
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

func (m *Manager) run() {
	defer m.stopper.Flow().ReportStopped()

	select {
	case <-m.complianceIsReady.Done():
	case <-m.stopper.Flow().StopRequested():
		return
	}

	for {
		select {
		case <-m.stopper.Flow().StopRequested():
			return
		}
	}
}
