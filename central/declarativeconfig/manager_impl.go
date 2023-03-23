package declarativeconfig

import (
	"context"
	"fmt"
	"os"
	"path"
	"reflect"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	timestamp "github.com/gogo/protobuf/types"
	"github.com/hashicorp/go-multierror"
	"github.com/mitchellh/hashstructure/v2"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/declarativeconfig/types"
	"github.com/stackrox/rox/central/declarativeconfig/updater"
	declarativeConfigUtils "github.com/stackrox/rox/central/declarativeconfig/utils"
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
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	declarativeConfigDir = "/run/stackrox.io/declarative-configuration"

	// The number of consecutive errors for a declarative configuration that causes its health status to be UNHEALTHY.
	consecutiveReconciliationErrorThreshold = 3

	handlerIntegrationHealthStatusPrefix = "Config Map"
)

type protoMessagesByType = map[reflect.Type][]proto.Message

type managerImpl struct {
	once sync.Once

	universalTransformer transform.Transformer

	transformedMessagesByHandler map[string]protoMessagesByType
	transformedMessagesMutex     sync.RWMutex

	lastReconciliationHash uint64
	lastUpsertFailed       concurrency.Flag
	lastDeletionFailed     concurrency.Flag

	reconciliationTickerDuration time.Duration
	watchIntervalDuration        time.Duration

	nameExtractor types.NameExtractor
	idExtractor   types.IDExtractor

	reconciliationTicker *time.Ticker
	shortCircuitSignal   concurrency.Signal

	reconciliationCtx context.Context

	declarativeConfigErrorReporter integrationhealth.Reporter
	errorsPerDeclarativeConfig     map[string]int32

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
	reconciliationErrorReporter integrationhealth.Reporter, nameExtractor types.NameExtractor, idExtractor types.IDExtractor) Manager {
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
		idExtractor:                    idExtractor,
		nameExtractor:                  nameExtractor,
		shortCircuitSignal:             concurrency.NewSignal(),
	}
}

func (m *managerImpl) ReconcileDeclarativeConfigurations() {
	m.once.Do(func() {
		if err := m.verifyUpdaters(); err != nil {
			utils.Should(err)
			log.Error("Received an error during verification of updaters. No reconciliation will be done.")
			return
		}

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

			if err := m.declarativeConfigErrorReporter.Register(dirToWatch, handlerNameForIntegrationHealth(dirToWatch),
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
			m.registerHealthForMessage(handlerID, protoMessages...)
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
			log.Debug("Received a short circuit signal, running the reconciliation")
			m.shortCircuitSignal.Reset()
			m.runReconciliation()
		case <-m.reconciliationTicker.C:
			log.Debug("Received a ticker signal, running the reconciliation")
			m.runReconciliation()
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

	hasChanges := m.calculateHashAndIndicateChanges(transformedMessagesByHandler)

	// If no changes are indicated within the message we reconcile, and no previous reconciliation failed, do not
	// run the reconciliation.
	if !hasChanges && !m.lastUpsertFailed.Get() && !m.lastDeletionFailed.Get() {
		log.Debug("No changes found compared to the previous reconciliation, and no errors have occurred." +
			" The reconciliation will be skipped.")
		return
	}

	// Only upsert resources if either the messages we reconcile on changed or the last upsert failed, as it might
	// have been a transient error or successfully remediated by now.
	if hasChanges || m.lastUpsertFailed.Get() {
		m.doUpsert(transformedMessagesByHandler)
	}

	// Only delete resources if either the messages we reconcile on changed or the last deletion failed, as it might
	// have been a transient error or successfully remediated by now.
	if hasChanges || m.lastDeletionFailed.Get() {
		m.doDeletion(transformedMessagesByHandler)
	}
}

func (m *managerImpl) doUpsert(transformedMessagesByHandler map[string]protoMessagesByType) {
	var failureInUpsert bool
	for _, protoType := range protoTypesOrder {
		for handler, protoMessagesByType := range transformedMessagesByHandler {
			messages, hasMessages := protoMessagesByType[protoType]
			if !hasMessages {
				continue
			}
			typeUpdater := m.updaters[protoType]
			for _, message := range messages {
				err := typeUpdater.Upsert(m.reconciliationCtx, message)
				m.updateHealthForMessage(handler, message, err, consecutiveReconciliationErrorThreshold)
				if err != nil {
					failureInUpsert = true
				}
			}
		}
	}
	m.lastUpsertFailed.Set(failureInUpsert)
}

func (m *managerImpl) doDeletion(transformedMessagesByHandler map[string]protoMessagesByType) {
	reversedProtoTypes := sliceutils.Reversed(protoTypesOrder)
	var failureInDeletion bool
	var allProtoIDsToSkip []string
	for _, protoType := range reversedProtoTypes {
		var idsToSkip []string
		for _, protoMessageByType := range transformedMessagesByHandler {
			messages := protoMessageByType[protoType]
			for _, message := range messages {
				idsToSkip = append(idsToSkip, m.idExtractor(message))
			}
		}
		allProtoIDsToSkip = append(allProtoIDsToSkip, idsToSkip...)
		typeUpdater := m.updaters[protoType]
		log.Debugf("Running deletion with resource updater %T, skipping IDs %+v", typeUpdater, idsToSkip)
		failedDeletionIDs, err := typeUpdater.DeleteResources(m.reconciliationCtx, idsToSkip...)
		log.Debugf("Finished deletion, return value: %+v", err)
		// In case of an error, ensure we do not delete the integration health status for resources we failed to delete.
		// Otherwise, the reason why the deletion failed will not be visible to users while the resource may still
		// exist.
		if err != nil {
			log.Debugf("The following IDs failed deletion: [%s]", strings.Join(failedDeletionIDs, ","))
			allProtoIDsToSkip = append(allProtoIDsToSkip, failedDeletionIDs...)
			failureInDeletion = true
		}
	}

	if err := m.removeStaleHealthStatuses(allProtoIDsToSkip); err != nil {
		log.Errorf("Failed to delete stale health status entries for declarative config: %v", err)
	}
	m.lastDeletionFailed.Set(failureInDeletion)
}

// updateHealthForMessage will update the health status of a message using the integrationhealth.Reporter.
// In case err == nil, the health status will be set to healthy.
// In case err != nil _and_ the number of errors for this message is >= the given threshold, the health
// status will be set to unhealthy.
func (m *managerImpl) updateHealthForMessage(handler string, message proto.Message, err error, threshold int32) {
	messageID := m.idExtractor(message)
	integrationHealth := declarativeConfigUtils.IntegrationHealthForProtoMessage(message, handler, err, m.idExtractor, m.nameExtractor)

	if err != nil {
		m.errorsPerDeclarativeConfig[messageID]++
		if m.errorsPerDeclarativeConfig[messageID] >= threshold {
			m.declarativeConfigErrorReporter.UpdateIntegrationHealthAsync(integrationHealth)
		}
		log.Debugf("Error within reconciliation for %+v: %v", message, err)
	} else {
		m.errorsPerDeclarativeConfig[messageID] = 0
		m.declarativeConfigErrorReporter.UpdateIntegrationHealthAsync(integrationHealth)
		log.Debugf("Message %+v marked as healthy", message)
	}
}

// updateHandlerHealth will update the health status of a handler using the integrationhealth.Reporter.
func (m *managerImpl) updateHandlerHealth(handlerID string, err error) {
	var errMsg string
	if err != nil {
		errMsg = err.Error()
	}

	m.declarativeConfigErrorReporter.UpdateIntegrationHealthAsync(&storage.IntegrationHealth{
		Id:            handlerID,
		Name:          handlerNameForIntegrationHealth(handlerID),
		Type:          storage.IntegrationHealth_DECLARATIVE_CONFIG,
		Status:        utils.IfThenElse(err != nil, storage.IntegrationHealth_UNHEALTHY, storage.IntegrationHealth_HEALTHY),
		ErrorMessage:  errMsg,
		LastTimestamp: timestamp.TimestampNow(),
	})
}

func (m *managerImpl) registerHealthForMessage(handler string, messages ...proto.Message) {
	for _, message := range messages {
		messageID := m.idExtractor(message)
		messageName := declarativeConfigUtils.NameForIntegrationHealthFromProtoMessage(message, handler, m.nameExtractor, m.idExtractor)

		if err := m.declarativeConfigErrorReporter.Register(messageID, messageName,
			storage.IntegrationHealth_DECLARATIVE_CONFIG); err != nil {
			log.Errorf("Error registering health status for declarative config %+v: %v", message, err)
		}
	}
}

// removeStaleHealthStatuses is expected to run after reconciliation has deleted declarative proto messages.
// It deletes health status entries for deleted proto messages, as well as entries for which the first creation
// of a resource failed.
func (m *managerImpl) removeStaleHealthStatuses(idsToSkip []string) error {
	healths, err := m.declarativeConfigErrorReporter.
		RetrieveIntegrationHealths(storage.IntegrationHealth_DECLARATIVE_CONFIG)
	if err != nil {
		return errors.Wrap(err, "retrieving integration health statuses for declarative config")
	}

	idsToSkipSet := set.NewFrozenStringSet(idsToSkip...)

	var removingIntegrationHealthsErr *multierror.Error
	for _, health := range healths {
		if idsToSkipSet.Contains(health.GetId()) {
			continue
		}

		if strings.Contains(health.GetName(), handlerIntegrationHealthStatusPrefix) {
			continue
		}

		if err := m.declarativeConfigErrorReporter.RemoveIntegrationHealth(health.GetId()); err != nil {
			removingIntegrationHealthsErr = multierror.Append(removingIntegrationHealthsErr, err)
		}
	}

	return removingIntegrationHealthsErr.ErrorOrNil()
}

func (m *managerImpl) verifyUpdaters() error {
	for _, protoType := range protoTypesOrder {
		if updater, ok := m.updaters[protoType]; !ok {
			return errox.InvariantViolation.Newf("found no updater for proto type %v", protoType)
		} else if updater == nil {
			return errox.InvariantViolation.Newf("updater for proto type %v was nil", protoType)
		}
	}
	return nil
}

func (m *managerImpl) calculateHashAndIndicateChanges(transformedMessagesByHandler map[string]protoMessagesByType) bool {
	// Create a hash from the transformed messages by handler map.
	// Setting the option ZeroNil will ensure empty byte arrays will be treated as a zero value instead of using
	// the pointer's value.
	hash, err := hashstructure.Hash(transformedMessagesByHandler, hashstructure.FormatV2,
		&hashstructure.HashOptions{ZeroNil: true})

	// If we received an error for hash generation, log it and _always_ run the deletion. This way we ensure
	// we don't mistakenly skip reconciliation runs where we shouldn't (e.g. consecutive errors).
	if err != nil {
		log.Errorf("Failed to create hash for transformed messages by handler %+v, "+
			"reconciliation will be executed: %v",
			transformedMessagesByHandler, err)
		return true
	}

	if m.lastReconciliationHash != hash {
		m.lastReconciliationHash = hash
		return true
	}
	return false
}

func handlerNameForIntegrationHealth(handlerID string) string {
	return fmt.Sprintf("%s %s", handlerIntegrationHealthStatusPrefix, path.Base(handlerID))
}
