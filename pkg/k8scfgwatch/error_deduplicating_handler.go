package k8scfgwatch

type errDeduplicatingHandler struct {
	Handler

	lastWatchErr string
}

func (h *errDeduplicatingHandler) OnStableUpdate(val interface{}, err error) {
	h.lastWatchErr = ""
	h.Handler.OnStableUpdate(val, err)
}

func (h *errDeduplicatingHandler) OnWatchError(err error) {
	errMsg := err.Error()
	if h.lastWatchErr != "" && h.lastWatchErr == errMsg {
		return
	}
	h.lastWatchErr = errMsg
	h.Handler.OnWatchError(err)
}

// DeduplicateWatchErrors returns a handler that ensures that `OnWatchError` will not be invoked consecutively with
// errors having the same message. Note that the occurrence of a stable update (regardless of erroneous or not) ensures
// that the next watch error is passed on to the underlying handler, even if it is the same error as the previous watch
// error.
func DeduplicateWatchErrors(h Handler) Handler {
	return &errDeduplicatingHandler{
		Handler: h,
	}
}
