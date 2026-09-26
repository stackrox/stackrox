package backgroundworker

// StatusAdapter wraps a custom component for registry integration without
// requiring it to conform to a standard archetype. Use this for components
// that stay custom but still want debug endpoint and metrics visibility.
//
//	adapter := &backgroundworker.StatusAdapter{
//	    WorkerName: "my-custom-worker",
//	    WorkerKind: "custom",
//	    StatusFunc: func() backgroundworker.WorkerStatus { ... },
//	}
//	backgroundworker.Global.Register(adapter)
type StatusAdapter struct {
	WorkerName string
	WorkerKind string
	StatusFunc func() WorkerStatus
}

// Status returns the status from the wrapped StatusFunc, filling in Name
// and Kind if the StatusFunc leaves them empty.
func (a *StatusAdapter) Status() WorkerStatus {
	if a.StatusFunc == nil {
		return WorkerStatus{
			Name:  a.WorkerName,
			Kind:  a.WorkerKind,
			State: stateIdle,
		}
	}
	s := a.StatusFunc()
	if s.Name == "" {
		s.Name = a.WorkerName
	}
	if s.Kind == "" {
		s.Kind = a.WorkerKind
	}
	return s
}
