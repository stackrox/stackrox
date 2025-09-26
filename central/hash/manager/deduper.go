package manager

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/alert"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	ops "github.com/stackrox/rox/pkg/metrics"
	eventPkg "github.com/stackrox/rox/pkg/sensor/event"
	"github.com/stackrox/rox/pkg/sensor/hash"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

// Deduper is an interface to deduping logic used to determine whether an event should be processed
//
//go:generate mockgen-wrapper
type Deduper interface {
	// GetSuccessfulHashes returns a map of key to hashes that were successfully processed by Central
	// and thus can be persisted in the database as being processed
	GetSuccessfulHashes() map[string]uint64
	// ShouldProcess takes a message and determines if it should be processed
	ShouldProcess(msg *central.MsgFromSensor) bool
	// MarkSuccessful marks the message if necessary as being successfully processed, so it can be committed to the database
	// It will promote the message from the received map to the successfully processed map
	MarkSuccessful(msg *central.MsgFromSensor)
	// RemoveMessage deletes the msg from the received map in order to allow future messages a chance to be successfully processed
	RemoveMessage(msg *central.MsgFromSensor)
	// StartSync is called once a new Sensor connection is initialized
	StartSync()
	// ProcessSync processes the Sensor sync message and reconciles the successfully processed and received maps
	ProcessSync()
}

var (
	maxHashes = env.MaxEventHashSize.IntegerSetting()

	alertResourceKey      = eventPkg.GetEventTypeWithoutPrefix((*central.SensorEvent_AlertResults)(nil))
	deploymentResourceKey = eventPkg.GetEventTypeWithoutPrefix((*central.SensorEvent_Deployment)(nil))
)

// NewDeduper creates a new deduper from the passed existing hashes
func NewDeduper(existingHashes map[string]uint64, clusterID string) Deduper {
	existingEntries := make(map[string]*entry)

	if len(existingHashes) < maxHashes {
		for k, v := range existingHashes {
			existingEntries[k] = &entry{
				val: v,
			}
		}
	}

	return &deduperImpl{
		received:              make(map[string]*entry),
		successfullyProcessed: existingEntries,
		hasher:                hash.NewHasher(),
		clusterID:             clusterID,
	}
}

type entry struct {
	val       uint64
	processed bool
}

type deduperImpl struct {
	hashLock sync.RWMutex
	// received map contains messages that have been received but not successfully processed
	received map[string]*entry
	// successfully processed map contains hashes of objects that have been successfully processed
	successfullyProcessed map[string]*entry

	hasher    *hash.Hasher
	clusterID string
}

// skipDedupe signifies that a message from Sensor cannot be deduped and won't be stored
func skipDedupe(msg *central.MsgFromSensor) bool {
	eventMsg, ok := msg.GetMsg().(*central.MsgFromSensor_Event)
	if !ok {
		return true
	}

	switch eventMsg.Event.GetResource().(type) {
	case *central.SensorEvent_Synced:
		return false
	case *central.SensorEvent_Pod:
		return false
	case *central.SensorEvent_Deployment:
		return false
	case *central.SensorEvent_Namespace:
		return false
	case *central.SensorEvent_AlertResults:
		alertResults := msg.GetEvent().GetAlertResults()
		if alert.IsRuntimeAlertResult(alertResults) {
			return true
		}
		// This can occur for a very short-lived deployment where alerts are not generated
		// but the deployment is being removed
		if alert.IsAlertResultResolved(alertResults) {
			return true
		}
		if alert.AnyAttemptedAlert(alertResults.GetAlerts()...) {
			return true
		}
		return false

	// Network and Security Resources
	case *central.SensorEvent_NetworkPolicy:
		return false
	case *central.SensorEvent_Secret:
		return false

	// Infrastructure Resources
	case *central.SensorEvent_Node:
		return true
	case *central.SensorEvent_NodeInventory:
		return true
	case *central.SensorEvent_IndexReport:
		return true

	// RBAC Resources
	case *central.SensorEvent_ServiceAccount:
		return false
	case *central.SensorEvent_Role:
		return false
	case *central.SensorEvent_Binding:
		return false

	// Process and Runtime
	case *central.SensorEvent_ProcessIndicator:
		return true
	case *central.SensorEvent_ReprocessDeployment:
		return true

	// Metadata and Configuration
	case *central.SensorEvent_ProviderMetadata:
		return true
	case *central.SensorEvent_OrchestratorMetadata:
		return true
	case *central.SensorEvent_ImageIntegration:
		return true

	// Compliance Operator (V1)
	case *central.SensorEvent_ComplianceOperatorResult,
		*central.SensorEvent_ComplianceOperatorProfile,
		*central.SensorEvent_ComplianceOperatorRule,
		*central.SensorEvent_ComplianceOperatorScanSettingBinding,
		*central.SensorEvent_ComplianceOperatorScan:
		return true

	// Virtual Machines
	case *central.SensorEvent_VirtualMachineIndexReport:
		return true
	case *central.SensorEvent_VirtualMachine:
		return false

	// Compliance Operator V2
	case *central.SensorEvent_ComplianceOperatorResultV2,
		*central.SensorEvent_ComplianceOperatorProfileV2,
		*central.SensorEvent_ComplianceOperatorRuleV2,
		*central.SensorEvent_ComplianceOperatorScanV2,
		*central.SensorEvent_ComplianceOperatorScanSettingBindingV2,
		*central.SensorEvent_ComplianceOperatorSuiteV2,
		*central.SensorEvent_ComplianceOperatorRemediationV2:
		return true

	default:
		utils.Should(errors.Errorf("unexpected sensor event type %q.  Please add to the switch and evaluate if it should be added to hashes or not", eventPkg.GetEventTypeWithoutPrefix(eventMsg.Event.GetResource())))
		return true
	}
}

func (d *deduperImpl) shouldReprocess(hashKey string, hash uint64) bool {
	d.hashLock.RLock()
	defer d.hashLock.RUnlock()

	prevValue, ok := d.getValueNoLock(hashKey)
	if !ok {
		// This implies that a REMOVE event has been processed before this event
		// Note: we may want to handle alerts specifically because we should insert them as already resolved for completeness
		return false
	}
	// This implies that no new event was processed after the initial processing of the current message
	return prevValue == hash
}

// StartSync is called when Sensor starts a new connection
func (d *deduperImpl) StartSync() {
	d.hashLock.Lock()
	defer d.hashLock.Unlock()

	for _, v := range d.received {
		v.processed = false
	}

	// Mark all hashes as unseen when a new sensor connection is created
	for _, v := range d.successfullyProcessed {
		v.processed = false
	}
}

// RemoveMessage removes a message that was unsuccessfully processed and purges any values for it from the deduper.
// This only applies when the message had an unretryable error such as context canceled
func (d *deduperImpl) RemoveMessage(msg *central.MsgFromSensor) {
	if skipDedupe(msg) {
		return
	}
	key := eventPkg.GetKeyFromMessage(msg)
	concurrency.WithLock(&d.hashLock, func() {
		delete(d.received, key)
	})
}

// MarkSuccessful marks a message as successfully processed
func (d *deduperImpl) MarkSuccessful(msg *central.MsgFromSensor) {
	// If the object isn't eligible for deduping then do not mark it as being successfully processed
	// because it does not exist in the received map
	if skipDedupe(msg) {
		return
	}
	key := eventPkg.GetKeyFromMessage(msg)

	d.hashLock.Lock()
	defer d.hashLock.Unlock()
	// If we are removing, then we do not need to mark it as successful as there is nothing more
	// to potentially dedupe. We do need to remove it from successfully processed as ShouldProcess is
	// evaluated as objects come into the queue and an object may have been successfully processed after
	if msg.GetEvent().GetAction() == central.ResourceAction_REMOVE_RESOURCE {
		delete(d.successfullyProcessed, key)
		dedupingHashCounterVec.With(prometheus.Labels{
			"cluster":      d.clusterID,
			"ResourceType": eventPkg.GetEventTypeWithoutPrefix(msg.GetEvent().GetResource()),
			"Operation":    ops.Remove.String()}).Inc()
		return
	}

	val, ok := d.received[key]
	// Only remove from this map if the hash matches as received could contain a more recent event
	if ok && val.val == msg.GetEvent().GetSensorHash() {
		delete(d.received, key)
	}

	dedupingHashCounterVec.With(prometheus.Labels{
		"cluster":      d.clusterID,
		"ResourceType": eventPkg.GetEventTypeWithoutPrefix(msg.GetEvent().GetResource()),
		"Operation":    ops.Add.String()}).Inc()
	d.successfullyProcessed[key] = &entry{
		val:       msg.GetEvent().GetSensorHash(),
		processed: true,
	}
}

func (d *deduperImpl) getValueNoLock(key string) (uint64, bool) {
	// Only return here if `processed` is true. If `processed` is false, it means
	// the value is dirty (was received in a previous connection but not processed).
	// if the value is dirty, it needs to be processed.
	if prevValue, ok := d.received[key]; ok && prevValue.processed {
		return prevValue.val, ok
	}
	// We do not check the `processed` field here because if it was successfully
	// processed at any time, we can skip the processing.
	prevValue, ok := d.successfullyProcessed[key]
	if !ok {
		return 0, false
	}
	return prevValue.val, true
}

// ProcessSync is triggered by the sync message sent from Sensor
func (d *deduperImpl) ProcessSync() {
	// Reconcile successfully processed map with received map. Any keys that exist in successfully processed
	// but do not exist in received, can be dropped from successfully processed
	d.hashLock.Lock()
	defer d.hashLock.Unlock()

	for k, v := range d.successfullyProcessed {
		// Ignore alerts in the first pass because they are not reconciled at the same time
		if strings.HasPrefix(k, alertResourceKey) {
			continue
		}
		if !v.processed {
			if val, ok := d.received[k]; ok && val.processed {
				continue
			}
			delete(d.successfullyProcessed, k)
			// If a deployment is being removed due to reconciliation, then we will need to remove the alerts too
			if strings.HasPrefix(k, deploymentResourceKey) {
				alertKey := eventPkg.FormatKey(alertResourceKey, eventPkg.ParseIDFromKey(k))
				delete(d.successfullyProcessed, alertKey)
			}
		}
	}
}

// ShouldProcess determines if a message should be processed or if it should be deduped and dropped
func (d *deduperImpl) ShouldProcess(msg *central.MsgFromSensor) bool {
	if skipDedupe(msg) {
		return true
	}
	event := msg.GetEvent()
	key := eventPkg.GetKeyFromMessage(msg)
	switch event.GetAction() {
	case central.ResourceAction_REMOVE_RESOURCE:
		d.hashLock.Lock()
		defer d.hashLock.Unlock()

		delete(d.received, key)
		delete(d.successfullyProcessed, key)
		return true
	case central.ResourceAction_SYNC_RESOURCE:
		// check if element is in successfully processed and mark as processed for syncs so that these are not reconciled away
		concurrency.WithLock(&d.hashLock, func() {
			if val, ok := d.successfullyProcessed[key]; ok {
				val.processed = true
			}
		})
	}
	// Backwards compatibility with a previous Sensor
	if event.GetSensorHashOneof() == nil {
		// Compute the sensor hash
		hashValue, ok := d.hasher.HashEvent(msg.GetEvent())
		if !ok {
			return true
		}
		event.SensorHashOneof = &central.SensorEvent_SensorHash{
			SensorHash: hashValue,
		}
	}
	// In the reprocessing case, the above will never evaluate to not nil, but it makes testing easier
	if msg.GetProcessingAttempt() > 0 {
		return d.shouldReprocess(key, event.GetSensorHash())
	}

	d.hashLock.Lock()
	defer d.hashLock.Unlock()

	prevValue, ok := d.getValueNoLock(key)
	if ok && prevValue == event.GetSensorHash() {
		return false
	}
	d.received[key] = &entry{
		val:       event.GetSensorHash(),
		processed: true,
	}
	return true
}

// GetSuccessfulHashes returns a copy of the successfullyProcessed map
func (d *deduperImpl) GetSuccessfulHashes() map[string]uint64 {
	d.hashLock.RLock()
	defer d.hashLock.RUnlock()

	copied := make(map[string]uint64, len(d.successfullyProcessed))
	for k, v := range d.successfullyProcessed {
		copied[k] = v.val
	}

	// Do not persist copied map if it has more than the max number of hashes. This could occur in two cases
	// 1. The environment is much larger than anticipated (default max hashes is 1 million).
	// 2. There is an event that does not have a proper lifecycle and it is causing unbounded growth.
	if len(copied) > maxHashes {
		return make(map[string]uint64)
	}
	return copied
}
