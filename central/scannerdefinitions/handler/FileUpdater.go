package handler

// RequestedUpdater defines the methods for updating files.
type RequestedUpdater interface {
	Start()
	Stop()
}
