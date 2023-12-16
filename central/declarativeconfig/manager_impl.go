package declarativeconfig

import (
	"context"
	"os"
	"path"
	"reflect"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/hashicorp/go-multierror"
	"github.com/mitchellh/hashstructure/v2"
	"github.com/pkg/errors"
	declarativeConfigHealth "github.com/stackrox/rox/central/declarativeconfig/health/datastore"
	"github.com/stackrox/rox/central/declarativeconfig/types"
	"github.com/stackrox/rox/central/declarativeconfig/updater"
	declarativeConfigUtils "github.com/stackrox/rox/central/declarativeconfig/utils"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/declarativeconfig/transform"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/k8scfgwatch"
	"github.com/stackrox/rox/pkg/maputil"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	declarativeConfigDir = "/run/stackrox.io/declarative-configuration"

	// The number of consecutive errors for a declarative configuration that causes its health status to be UNHEALTHY.
	consecutiveReconciliationErrorThreshold = 3
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

	declarativeConfigHealthDS  declarativeConfigHealth.DataStore
	errorsPerDeclarativeConfig map[string]int32

	updaters map[reflect.Type]updater.ResourceUpdater

	numberOfWatchHandlers atomic.Int32
}

var protoTypesOrder = []reflect.Type{
	types.AccessScopeType,
	types.PermissionSetType,
	types.RoleType,
	types.AuthProviderType,
	types.GroupType,
	types.NotifierType,
}

// New creates a new instance of Manager.
// Note that it will not watch the declarative configuration directories when created, only after
// ReconcileDeclarativeConfigurations has been called.
func New(reconciliationTickerDuration, watchIntervalDuration time.Duration, updaters map[reflect.Type]updater.ResourceUpdater,
	declarativeConfigHealthStore declarativeConfigHealth.DataStore, nameExtractor types.NameExtractor, idExtractor types.IDExtractor) Manager {
	writeDeclarativeRoleCtx := declarativeconfig.WithModifyDeclarativeResource(context.Background())
	writeDeclarativeRoleCtx = sac.WithGlobalAccessScopeChecker(writeDeclarativeRoleCtx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.Access, resources.Integration)))

	return &managerImpl{
		universalTransformer:         transform.New(),
		transformedMessagesByHandler: map[string]protoMessagesByType{},
		reconciliationTickerDuration: reconciliationTickerDuration,
		watchIntervalDuration:        watchIntervalDuration,
		updaters:                     updaters,
		reconciliationCtx:            writeDeclarativeRoleCtx,
		declarativeConfigHealthDS:    declarativeConfigHealthStore,
		errorsPerDeclarativeConfig:   map[string]int32{},
		idExtractor:                  idExtractor,
		nameExtractor:                nameExtractor,
		shortCircuitSignal:           concurrency.NewSignal(),
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
		var numberOfWatchHandlers int
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
			numberOfWatchHandlers++

			m.registerDeclarativeConfigHealth(declarativeConfigUtils.HealthStatusForHandler(dirToWatch, nil))
		}

		// Only start the reconciliation loop if at least one watch handler has been started.
		if startedWatchHandler {
			log.Info("Start the reconciliation loop for declarative configurations")
			m.startReconciliationLoop()
		}
		m.numberOfWatchHandlers.Swap(int32(numberOfWatchHandlers))
	})
}

func (m *managerImpl) Gather() phonehome.GatherFunc {
	return func(ctx context.Context) (map[string]any, error) {
		return map[string]any{
			"Total Number of declarative configuration mounts": m.numberOfWatchHandlers.Load(),
		}, nil
	}
}

// UpdateDeclarativeConfigContents will take the file contents and transform these to declarative configurations.
func (m *managerImpl) UpdateDeclarativeConfigContents(handlerID string, contents [][]byte) {
	// Operate the whole function under a single lock.
	// This is due to the nature of the function being called from multiple go routines, and the possibility currently
	// being there that two concurrent calls to Update lead to transformed configurations being potentially overwritten.
	m.transformedMessagesMutex.Lock()
	defer m.transformedMessagesMutex.Unlock()
	configurations, err := declarativeconfig.ConfigurationFromRawBytes(contents...)
	if err != nil {
		m.updateDeclarativeConfigHealth(declarativeConfigUtils.HealthStatusForHandler(handlerID, err))
		log.Debugf("Error during unmarshalling of declarative configuration files: %+v", err)
		return
	}

	transformedConfigurations := make(map[reflect.Type][]proto.Message, len(configurations))
	var transformationErrors *multierror.Error
	for _, configuration := range configurations {
		transformedConfig, err := m.universalTransformer.Transform(configuration)
		if err != nil {
			log.Debugf("Error during transforming declarative configuration %+v: %+v", configuration, err)
			transformationErrors = multierror.Append(transformationErrors, err)
			continue
		}
		for protoType, protoMessages := range transformedConfig {
			transformedConfigurations[protoType] = append(transformedConfigurations[protoType], protoMessages...)
			// Register health status for all new messages. Existing messages will not be re-registered.
			m.registerHealthForMessages(handlerID, protoMessages...)
		}
	}
	m.updateDeclarativeConfigHealth(declarativeConfigUtils.HealthStatusForHandler(handlerID,
		errors.Wrap(transformationErrors.ErrorOrNil(), "during transforming configuration")))

	m.transformedMessagesByHandler[handlerID] = transformedConfigurations
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
	transformedMessagesByHandler := concurrency.WithRLock1(&m.transformedMessagesMutex, func() map[string]protoMessagesByType {
		return maputil.ShallowClone(m.transformedMessagesByHandler)
	})

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
	healthStatus := declarativeConfigUtils.HealthStatusForProtoMessage(message, handler, err, m.idExtractor, m.nameExtractor)

	if err != nil {
		m.errorsPerDeclarativeConfig[messageID]++
		if m.errorsPerDeclarativeConfig[messageID] >= threshold {
			m.updateDeclarativeConfigHealth(healthStatus)
		}
		log.Debugf("Error within reconciliation for %+v: %v", message, err)
	} else {
		m.errorsPerDeclarativeConfig[messageID] = 0
		m.updateDeclarativeConfigHealth(healthStatus)
		log.Debugf("Message %+v marked as healthy", message)
	}
}

func (m *managerImpl) registerHealthForMessages(handler string, messages ...proto.Message) {
	for _, message := range messages {
		health := declarativeConfigUtils.HealthStatusForProtoMessage(message, handler, nil, m.idExtractor, m.nameExtractor)
		m.registerDeclarativeConfigHealth(health)
	}
}

// removeStaleHealthStatuses is expected to run after reconciliation has deleted declarative proto messages.
// It deletes health status entries for deleted proto messages, as well as entries for which the first creation
// of a resource failed.
func (m *managerImpl) removeStaleHealthStatuses(idsToSkip []string) error {
	healths, err := m.declarativeConfigHealthDS.GetDeclarativeConfigs(m.reconciliationCtx)
	if err != nil {
		return errors.Wrap(err, "retrieving integration health statuses for declarative config")
	}

	idsToSkipSet := set.NewFrozenStringSet(idsToSkip...)

	var removingIntegrationHealthsErr *multierror.Error
	for _, health := range healths {
		if idsToSkipSet.Contains(health.GetId()) {
			continue
		}

		if health.GetResourceType() == storage.DeclarativeConfigHealth_CONFIG_MAP {
			continue
		}

		// Special case: for roles, the health ID will be a UUID instead of the name. Hence, need to verify whether
		// the IDs to skip (which will include the role names) contains the health's resource name.
		if health.GetResourceType() == storage.DeclarativeConfigHealth_ROLE &&
			idsToSkipSet.Contains(health.GetResourceName()) {
			continue
		}

		if err := m.declarativeConfigHealthDS.RemoveDeclarativeConfig(m.reconciliationCtx,
			health.GetId()); err != nil {
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

func (m *managerImpl) updateDeclarativeConfigHealth(healthStatus *storage.DeclarativeConfigHealth) {
	if err := m.declarativeConfigHealthDS.UpsertDeclarativeConfig(m.reconciliationCtx, healthStatus); err != nil {
		log.Errorf("Could not upsert declarative config health %q: %v", healthStatus.GetId(), err)
	}
}

func (m *managerImpl) registerDeclarativeConfigHealth(healthStatus *storage.DeclarativeConfigHealth) {
	_, exists, err := m.declarativeConfigHealthDS.GetDeclarativeConfig(m.reconciliationCtx,
		healthStatus.GetId())
	// No-op. We do not want to upsert existing health status, as this might override the status and error messages.
	if exists {
		return
	}

	if err != nil {
		log.Errorf("Failed to retrieve declarative config health %q: %v", healthStatus.GetId(), err)
	}

	// Irrespective if we received an error to retrieve the health status, attempt to upsert it.
	m.updateDeclarativeConfigHealth(healthStatus)
}
