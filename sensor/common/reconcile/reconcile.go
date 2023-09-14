package reconcile

// SensorReconciliationEvent represents central.SensorEvent in places where the original cannot be used
type SensorReconciliationEvent struct{}

type Reconcilable interface {
	Reconcile(resType, resID string, resHash uint64) (*SensorReconciliationEvent, error)
}
