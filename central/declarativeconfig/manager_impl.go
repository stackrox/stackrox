package declarativeconfig

import (
	"context"
	"fmt"
	"os"
	"path"
	"reflect"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/declarativeconfig/types"
	"github.com/stackrox/rox/central/declarativeconfig/updater"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/declarativeconfig/transform"
	"github.com/stackrox/rox/pkg/k8scfgwatch"
	"github.com/stackrox/rox/pkg/maputil"
	"github.com/stackrox/rox/pkg/sac"
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

	reconciliationTicker *time.Ticker
	shortCircuitSignal   concurrency.Signal
	stopSignal           concurrency.Signal

	reconciliationCtx           context.Context
	reconciliationErrorReporter ReconciliationErrorReporter
	updaters                    map[reflect.Type]updater.ResourceUpdater
}

var protoTypesOrder = []reflect.Type{
	types.AccessScopeType,
	types.PermissionSetType,
	types.RoleType,
	types.AuthProviderType,
	types.GroupType,
}

// New creates a new instance of Manager.
// Note that it will not watch the declarative configuration directories when created, only after
// ReconcileDeclarativeConfigurations has been called.
func New(reconciliationTickerDuration, watchIntervalDuration time.Duration, updaters map[reflect.Type]updater.ResourceUpdater, reconciliationErrorReporter ReconciliationErrorReporter) Manager {
	writeDeclarativeRoleCtx := declarativeconfig.WithModifyDeclarativeResource(context.Background())
	writeDeclarativeRoleCtx = sac.WithGlobalAccessScopeChecker(writeDeclarativeRoleCtx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			// TODO: ROX-14398 Replace Role with Access
			sac.ResourceScopeKeys(resources.Role, resources.Access)))
	return &managerImpl{
		universalTransformer:         transform.New(),
		transformedMessagesByHandler: map[string]protoMessagesByType{},
		reconciliationTickerDuration: reconciliationTickerDuration,
		watchIntervalDuration:        watchIntervalDuration,
		updaters:                     updaters,
		reconciliationCtx:            writeDeclarativeRoleCtx,
		reconciliationErrorReporter:  reconciliationErrorReporter,
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

			dirToWatch := path.Join(declarativeConfigDir, entry.Name())
			log.Infof("Start watch handler for declarative configuration for path %s",
				dirToWatch)
			wh := newWatchHandler(dirToWatch, m)
			// Set Force to true, so we explicitly retry watching the files within the directory and not stop on the first
			// error occurred.
			watchOpts := k8scfgwatch.Options{Interval: m.watchIntervalDuration, Force: true}
			_ = k8scfgwatch.WatchConfigMountDir(context.Background(), dirToWatch,
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
	m.transformedMessagesByHandler[handlerID] = transformedConfigurations
	m.transformedMessagesMutex.Unlock()
	m.shortCircuitReconciliationLoop()
}

// shortCircuitReconciliationLoop will short circuit the reconciliation loop and trigger a reconciliation loop run.
// Note that the reconciliation loop will not be run if:
//   - the short circuit loop signal has not been reset yet and is de-duped.
func (m *managerImpl) shortCircuitReconciliationLoop() {
	// In case the signal is already triggered, the current call (and the Signal() call) will be effectively de-duped.
	m.shortCircuitSignal.Signal()
}

func (m *managerImpl) startReconciliationLoop() {
	m.reconciliationTicker = time.NewTicker(m.reconciliationTickerDuration)

	go m.reconciliationLoop()
}

func (m *managerImpl) reconciliationLoop() {
	// While we currently do not have an exit in the form of "stopping" the reconciliation, still, ensure that
	// the ticker is stopped when we stop running the reconciliation.
	defer m.reconciliationTicker.Stop()
	for {
		select {
		case <-m.shortCircuitSignal.Done():
			m.shortCircuitSignal.Reset()
			m.runReconciliation()
		case <-m.reconciliationTicker.C:
			m.runReconciliation()
		case <-m.stopSignal.Done():
			return
		}
	}
}

func (m *managerImpl) runReconciliation() {
	m.transformedMessagesMutex.RLock()
	transformedMessagesByHandler := maputil.ShallowClone(m.transformedMessagesByHandler)
	m.transformedMessagesMutex.RUnlock()
	m.reconcileTransformedMessages(transformedMessagesByHandler)
}

func (m *managerImpl) reconcileTransformedMessages(transformedMessagesByHandler map[string]protoMessagesByType) {
	log.Debugf("Run reconciliation for the next handlers: %v", maputil.Keys(transformedMessagesByHandler))
	transformedMessages := map[reflect.Type][]proto.Message{}
	for _, protoMessagesByType := range transformedMessagesByHandler {
		for protoType, protoMessages := range protoMessagesByType {
			transformedMessages[protoType] = append(transformedMessages[protoType], protoMessages...)
		}

	}
	for _, protoType := range protoTypesOrder {
		messages, hasMessages := transformedMessages[protoType]
		if !hasMessages {
			continue
		}
		typeUpdater, hasUpdater := m.updaters[protoType]
		if !hasUpdater {
			m.handleMissingTypeUpdater(protoType, messages)
			return
		}
		for _, message := range messages {
			if err := typeUpdater.Upsert(m.reconciliationCtx, message); err != nil {
				m.reconciliationErrorReporter.ProcessError(message, err)
			}
		}
	}
	// TODO(ROX-14694): Add deletion of resources.
	log.Debugf("Deleting all proto messages that have traits.Origin==DECLARATIVE but are not contained"+
		" within the current list of transformed messages: %+v", transformedMessagesByHandler)
}

func (m *managerImpl) handleMissingTypeUpdater(protoType reflect.Type, messages []proto.Message) {
	err := fmt.Errorf("manager does not have updater for type %v", protoType)
	for _, message := range messages {
		m.reconciliationErrorReporter.ProcessError(message, err)
	}
	utils.Should(err)
	m.stopSignal.Done()
}
