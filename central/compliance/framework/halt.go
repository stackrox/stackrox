package framework

type haltSignal struct {
	err error
}

// halt terminates the current compliance run immediately. It should only be used for errors that can occur during
// normal operation, not for unexpected/abnormal error conditions.
func halt(err error) {
	panic(haltSignal{err: err})
}
