package reconcile

// Reconcilable is a Sensor object that supports reconciliation.
// The Reconcile method is called after Sensor reconnects with Central and receives its state hashes.
// Reconciliacion ensures that Sensor and Central have the same state.
type Reconcilable interface {
	ReconcileDelete(resType, resID string, resHash uint64) ([]Resource, error)
}

// Resource represents a resource's id and its type that needs to be reconciled
type Resource interface {
	GetPair() (string, string)
}
