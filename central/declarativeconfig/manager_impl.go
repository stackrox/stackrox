package declarativeconfig

import (
	"context"
	"fmt"
	"os"
	"path"
	"reflect"
	"time"

	"github.com/gogo/protobuf/proto"
	timestamp "github.com/gogo/protobuf/types"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/declarativeconfig/types"
	"github.com/stackrox/rox/central/declarativeconfig/updater"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/declarativeconfig/transform"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/integrationhealth"
	"github.com/stackrox/rox/pkg/k8scfgwatch"
	"github.com/stackrox/rox/pkg/maputil"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	declarativeConfigDir = "/run/stackrox.io/declarative-configuration"

	// The number of consecutive errors for a declarative configuration that causes its health status to be UNHEALTHY.
	consecutiveReconciliationErrorThreshold = 5
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

	reconciliationCtx context.Context

	declarativeConfigErrorReporter integrationhealth.Reporter
	errorsPerDeclarativeConfig     map[string]int32
	declarativeConfigErrorsLock    sync.RWMutex

	updaters map[reflect.Type]updater.ResourceUpdater
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
func New(reconciliationTickerDuration, watchIntervalDuration time.Duration, updaters map[reflect.Type]updater.ResourceUpdater,
	reconciliationErrorReporter integrationhealth.Reporter) Manager {
	writeDeclarativeRoleCtx := declarativeconfig.WithModifyDeclarativeResource(context.Background())
	writeDeclarativeRoleCtx = sac.WithGlobalAccessScopeChecker(writeDeclarativeRoleCtx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			// TODO: ROX-14398 Replace Role with Access
			sac.ResourceScopeKeys(resources.Role, resources.Access)))
	return &managerImpl{
		universalTransformer:           transform.New(),
		transformedMessagesByHandler:   map[string]protoMessagesByType{},
		reconciliationTickerDuration:   reconciliationTickerDuration,
		watchIntervalDuration:          watchIntervalDuration,
		updaters:                       updaters,
		reconciliationCtx:              writeDeclarativeRoleCtx,
		declarativeConfigErrorReporter: reconciliationErrorReporter,
		errorsPerDeclarativeConfig:     map[string]int32{},
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

			if err := m.declarativeConfigErrorReporter.Register(dirToWatch, "Config Map "+entry.Name(),
				storage.IntegrationHealth_DECLARATIVE_CONFIG); err != nil {
				utils.Should(errors.Wrapf(err, "registering health status for handler %s", dirToWatch))
			}
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
		m.updateHandlerHealth(handlerID, err)
		log.Errorf("Error during unmarshalling of declarative configuration files: %+v", err)
		return
	}

	transformedConfigurations := make(map[reflect.Type][]proto.Message, len(configurations))
	var transformationErrors *multierror.Error
	for _, configuration := range configurations {
		transformedConfig, err := m.universalTransformer.Transform(configuration)
		if err != nil {
			log.Errorf("Error during transforming declarative configuration %+v: %+v", configuration, err)
			transformationErrors = multierror.Append(transformationErrors, err)
			continue
		}
		for protoType, protoMessages := range transformedConfig {
			transformedConfigurations[protoType] = append(transformedConfigurations[protoType], protoMessages...)
			// Register health status for all new messages. Existing messages will not be re-registered.
			m.registerHealthForMessage(protoMessages...)
		}
	}

	if err := transformationErrors.ErrorOrNil(); err != nil {
		m.updateHandlerHealth(handlerID, errors.Wrap(err, "during transforming configuration"))
	} else {
		m.updateHandlerHealth(handlerID, nil)
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
			err := typeUpdater.Upsert(m.reconciliationCtx, message)
			m.updateHealthForMessage(message, err, consecutiveReconciliationErrorThreshold)
		}
	}
	// TODO(ROX-14694): Add deletion of resources.
	log.Debugf("Deleting all proto messages that have traits.Origin==DECLARATIVE but are not contained"+
		" within the current list of transformed messages: %+v", transformedMessagesByHandler)
}

func (m *managerImpl) handleMissingTypeUpdater(protoType reflect.Type, messages []proto.Message) {
	err := fmt.Errorf("manager does not have updater for type %v", protoType)
	for _, message := range messages {
		// Set the threshold to 0, meaning we will _always_ update the integration health status to unhealthy.
		m.updateHealthForMessage(message, err, 0)
	}
	utils.Should(err)
	m.stopSignal.Signal()
}

// updateHealthForMessage will update the health status of a handler using the integrationhealth.Reporter.
// In case err == nil, the health status will be set to healthy.
// In case err != nil _and_ the number of errors for this message is >= the given threshold, the health
// status will be set to unhealthy.
func (m *managerImpl) updateHealthForMessage(message proto.Message, err error, threshold int32) {
	messageID := extractIDFromProtoMessage(message)

	if err != nil {
		var currentDeclarativeConfigErrors int32
		concurrency.WithLock(&m.declarativeConfigErrorsLock, func() {
			currentDeclarativeConfigErrors = m.errorsPerDeclarativeConfig[messageID] + 1
			m.errorsPerDeclarativeConfig[messageID] = currentDeclarativeConfigErrors
		})

		if currentDeclarativeConfigErrors >= threshold {
			m.declarativeConfigErrorReporter.UpdateIntegrationHealthAsync(&storage.IntegrationHealth{
				Id:            messageID,
				Name:          stringutils.FirstNonEmpty(extractNameFromProtoMessage(message), messageID),
				Type:          storage.IntegrationHealth_DECLARATIVE_CONFIG,
				Status:        storage.IntegrationHealth_UNHEALTHY,
				ErrorMessage:  err.Error(),
				LastTimestamp: timestamp.TimestampNow(),
			})
		}
	} else {
		var currentDeclarativeConfigErrors int32
		concurrency.WithRLock(&m.declarativeConfigErrorsLock, func() {
			currentDeclarativeConfigErrors = m.errorsPerDeclarativeConfig[messageID]
		})

		if currentDeclarativeConfigErrors > 0 {
			concurrency.WithLock(&m.declarativeConfigErrorsLock, func() {
				// Ensure the error count hasn't updated in the mean time.
				if m.errorsPerDeclarativeConfig[messageID] == currentDeclarativeConfigErrors {
					m.errorsPerDeclarativeConfig[messageID] = 0
				}
			})
		}

		m.declarativeConfigErrorReporter.UpdateIntegrationHealthAsync(&storage.IntegrationHealth{
			Id:            messageID,
			Name:          stringutils.FirstNonEmpty(extractNameFromProtoMessage(message), messageID),
			Type:          storage.IntegrationHealth_DECLARATIVE_CONFIG,
			Status:        storage.IntegrationHealth_HEALTHY,
			ErrorMessage:  "",
			LastTimestamp: timestamp.TimestampNow(),
		})
	}

	log.Debugf("Error within reconciliation for %+v: %v", message, err)
}

// updateHandlerHealth will update the health status of a handler using the integrationhealth.Reporter.
func (m *managerImpl) updateHandlerHealth(handlerID string, err error) {
	var errMsg string
	if err != nil {
		errMsg = err.Error()
	}

	m.declarativeConfigErrorReporter.UpdateIntegrationHealthAsync(&storage.IntegrationHealth{
		Id:            handlerID,
		Name:          fmt.Sprintf("Config Map %s", path.Base(handlerID)),
		Type:          storage.IntegrationHealth_DECLARATIVE_CONFIG,
		Status:        utils.IfThenElse(err != nil, storage.IntegrationHealth_UNHEALTHY, storage.IntegrationHealth_HEALTHY),
		ErrorMessage:  errMsg,
		LastTimestamp: timestamp.TimestampNow(),
	})
}

func (m *managerImpl) registerHealthForMessage(messages ...proto.Message) {
	for _, message := range messages {
		messageID := extractIDFromProtoMessage(message)
		messageName := extractNameFromProtoMessage(message)

		if err := m.declarativeConfigErrorReporter.Register(messageID, stringutils.FirstNonEmpty(messageName, messageID),
			storage.IntegrationHealth_DECLARATIVE_CONFIG); err != nil {
			log.Errorf("Error registering health status for declarative config %+v: %v", message, err)
		}
	}
}

// Helpers.

func extractIDFromProtoMessage(message proto.Message) string {
	// Special case, as the group specifies the ID nested within the groups properties.
	if group, ok := message.(*storage.Group); ok {
		return group.GetProps().GetId()
	}
	// Special case, as the name of the role is the ID.
	if role, ok := message.(*storage.Role); ok {
		return role.GetName()
	}

	messageWithID, ok := message.(interface {
		GetId() string
	})
	// Theoretically, this should never happen unless we add more proto messages to the reconciliation. Hence, we use
	// utils.Should to guard this.
	if !ok {
		utils.Should(errox.InvariantViolation.Newf("could not retrieve ID from message type %T %+v",
			message, message))
		return ""
	}
	return messageWithID.GetId()
}

func extractNameFromProtoMessage(message proto.Message) string {
	messageWithName, ok := message.(interface {
		GetName() string
	})
	// This may happen for some resources (such as groups, roles) as they do not define a name.
	if !ok {
		return ""
	}

	return messageWithName.GetName()
}
