package reconcile

// SensorReconciliationEvent defines what kind of message needs to be sent back to central to sync the states
type SensorReconciliationEvent string

const (
	// SensorReconciliationEventNoop means that no change in Central is needed
	SensorReconciliationEventNoop SensorReconciliationEvent = "noop"
	// SensorReconciliationEventDelete means that entry in Central shall be deleted
	SensorReconciliationEventDelete = "delete"
	// SensorReconciliationEventUpdate means that no entry in Central shall be updated
	SensorReconciliationEventUpdate = "update"
)

// Reconcilable is a Sensor object that supports reconciliation.
// The Reconcile method is called after Sensor reconnects with Central and receives its state hashes.
// Reconciliacion ensures that Sensor and Central have the same state.
type Reconcilable interface {
	Reconcile(resType, resID string, resHash uint64) (map[string]SensorReconciliationEvent, error)
}
