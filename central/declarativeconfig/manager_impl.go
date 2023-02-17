package declarativeconfig

import (
	"context"
	"os"
	"path"
	"reflect"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/declarativeconfig/transform"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/k8scfgwatch"
	"github.com/stackrox/rox/pkg/maputil"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	declarativeConfigDir = "/run/stackrox.io/declarative-configuration"
)

type protoMessagesByType = map[reflect.Type][]proto.Message

type managerImpl struct {
	once sync.Once

	universalTransformer transform.Transformer

	transformedMessagesByHandler map[string]protoMessagesByType
	transformedMessagesMutex     sync.RWMutex

	reconciliationTickerDuration time.Duration
	watchIntervalDuration        time.Duration

	reconciliationTicker     *time.Ticker
	reconciliationInProgress concurrency.Flag
	shortCircuitSignal       concurrency.Signal
}

// New creates a new instance of Manager.
// Note that it will not watch the declarative configuration directories when created, only after
// ReconcileDeclarativeConfigurations has been called.
func New() Manager {
	return &managerImpl{
		universalTransformer:         transform.New(),
		transformedMessagesByHandler: map[string]protoMessagesByType{},
		reconciliationTickerDuration: env.DeclarativeConfigReconcileInterval.DurationSetting(),
		watchIntervalDuration:        env.DeclarativeConfigWatchInterval.DurationSetting(),
	}
}

func (m *managerImpl) ReconcileDeclarativeConfigurations() {
	m.once.Do(func() {
		// For each directory within the declarative configuration path, create a watch handler.
		// The reason we need multiple watch handlers and cannot simply watch the root directory is that
		// changes to directories are ignored within the watch handler.
		entries, err := os.ReadDir(declarativeConfigDir)
		if err != nil {
			if os.IsNotExist(err) {
				log.Info("Declarative configuration directory does not exist, no reconciliation will be done")
				return
			}
			utils.Should(err)
			return
		}

		var startedWatchHandler bool
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			log.Infof("Start watch handler for declarative configuration for path %s",
				path.Join(declarativeConfigDir, entry.Name()))
			wh := newWatchHandler(entry.Name(), m)
			// Set Force to true, so we explicitly retry watching the files within the directory and not stop on the first
			// error occurred.
			watchOpts := k8scfgwatch.Options{Interval: m.watchIntervalDuration, Force: true}
			_ = k8scfgwatch.WatchConfigMountDir(context.Background(), declarativeConfigDir,
				k8scfgwatch.DeduplicateWatchErrors(wh), watchOpts)
			startedWatchHandler = true
		}

		// Only start the reconciliation loop if at least one watch handler has been started.
		if startedWatchHandler {
			log.Info("Start the reconciliation loop for declarative configurations")
			m.startReconciliationLoop()
		}
	})
}

// UpdateDeclarativeConfigContents will take the file contents and transform these to declarative configurations.
func (m *managerImpl) UpdateDeclarativeConfigContents(handlerID string, contents [][]byte) {
	configurations, err := declarativeconfig.ConfigurationFromRawBytes(contents...)
	if err != nil {
		log.Errorf("Error during unmarshalling of declarative configuration files: %+v", err)
		return
	}
	transformedConfigurations := make(map[reflect.Type][]proto.Message, len(configurations))
	for _, configuration := range configurations {
		transformedConfig, err := m.universalTransformer.Transform(configuration)
		if err != nil {
			log.Errorf("Error during transforming declarative configuration %+v: %+v", configuration, err)
			continue
		}
		for protoType, protoMessages := range transformedConfig {
			transformedConfigurations[protoType] = append(transformedConfigurations[protoType], protoMessages...)
		}
	}

	m.transformedMessagesMutex.Lock()
	defer m.transformedMessagesMutex.Unlock()
	m.transformedMessagesByHandler[handlerID] = transformedConfigurations
	m.shortCircuitReconciliationLoop()
}

// shortCircuitReconciliationLoop will short circuit the reconciliation loop and trigger a reconciliation loop run.
// Note that the reconciliation loop will not be run if:
//   - the short circuit loop signal has not been reset yet and is de-duped.
//   - a current reconciliation loop run is in progress.
func (m *managerImpl) shortCircuitReconciliationLoop() {
	// In case the signal is already triggered, the current call (and the Signal() call) will be effectively de-duped.
	m.shortCircuitSignal.Signal()
}

func (m *managerImpl) startReconciliationLoop() {
	m.reconciliationTicker = time.NewTicker(m.reconciliationTickerDuration)

	go m.reconciliationLoop()
}

func (m *managerImpl) reconciliationLoop() {
	// While we currently do not have an exist in the form of "stopping" the reconciliation, still, ensure that
	// the ticker is stopped when we stop running the reconciliation.
	defer m.reconciliationTicker.Stop()
	for {
		select {
		case <-m.shortCircuitSignal.Done():
			m.shortCircuitSignal.Reset()
			m.runReconciliation()
		case <-m.reconciliationTicker.C:
			m.runReconciliation()
		}
	}
}

func (m *managerImpl) runReconciliation() {
	// We shouldn't trigger a parallel reconciliation run, hence we should ensure that the flag is not set to true.
	if m.reconciliationInProgress.TestAndSet(true) {
		return
	}
	defer m.reconciliationInProgress.Set(false)
	m.transformedMessagesMutex.RLock()
	transformedMessagesByHandler := maputil.ShallowClone(m.transformedMessagesByHandler)
	m.transformedMessagesMutex.RUnlock()
	m.reconcileTransformedMessages(transformedMessagesByHandler)
}

func (m *managerImpl) reconcileTransformedMessages(transformedMessagesByHandler map[string]protoMessagesByType) {
	for handler, protoMessagesByType := range transformedMessagesByHandler {
		for protoType, protoMessages := range protoMessagesByType {
			// TODO(ROX-14693): Add upserting transformed resources.
			log.Debugf("Upserting transformed messages of type %s from file %s: %+v",
				protoType.Name(), handler, protoMessages)
		}
	}
	// TODO(ROX-14694): Add deletion of resources.
	log.Debugf("Deleting all proto messages that have traits.Origin==DECLARATIVE but are not contained"+
		" within the current list of transformed messages: %+v", transformedMessagesByHandler)
}
