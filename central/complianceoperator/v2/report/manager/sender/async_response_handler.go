package sender

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
)

var (
	ErrInvalidInput = errors.New("invalid input")
)

type asyncResponseHandlerImpl[T any] struct {
	onSuccessCallback func(T) error
	onErrorCallback   func()
	responseC         <-chan T
	stopper           concurrency.Stopper
}

type AsyncResponseHandler[T any] interface {
	Start()
	Stop()
}

func NewAsyncResponseHandler[T any](onSuccess func(T) error, onError func(), responseC <-chan T) (AsyncResponseHandler[T], error) {
	if onSuccess == nil || onError == nil || responseC == nil {
		return nil, ErrInvalidInput
	}
	ret := &asyncResponseHandlerImpl[T]{
		onSuccessCallback: onSuccess,
		onErrorCallback:   onError,
		responseC:         responseC,
		stopper:           concurrency.NewStopper(),
	}
	return ret, nil
}

func (h *asyncResponseHandlerImpl[T]) Start() {
	go h.run()
}

func (h *asyncResponseHandlerImpl[T]) run() {
	go func() {
		defer h.stopper.Flow().ReportStopped()
		select {
		case <-h.stopper.Flow().StopRequested():
			h.onErrorCallback()
		case msg, ok := <-h.responseC:
			if !ok {
				h.onErrorCallback()
				return
			}
			if err := h.onSuccessCallback(msg); err != nil {
				h.onErrorCallback()
			}
		}
	}()
}

func (h *asyncResponseHandlerImpl[T]) Stop() {
	h.stopper.Client().Stop()
}
