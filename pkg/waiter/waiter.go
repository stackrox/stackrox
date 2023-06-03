package waiter

import (
	"context"
	"errors"

	"github.com/stackrox/rox/pkg/concurrency"
)

var (
	// ErrWaiterClosed indicates the waiter has been closed prior to receiving a message.
	ErrWaiterClosed = errors.New("waiter has been closed")
)

// Waiter waits for a value to be published to it, typically via another goroutine.
//
// The Manager that built the waiter is responsible for delivering the
// expected value.
//
// It is the callers responsibility to ensure that either Wait finishes or Close
// is invoked so that proper cleanup can be done.
//
//go:generate mockgen-wrapper
type Waiter[T any] interface {
	// ID returns the unique ID assigned to this waiter.
	ID() string

	// Wait will block for a value, context expiration, or closed waiter (whichever occurs first)
	// error will be non-nil if context finishes or waiter closed.
	Wait(ctx context.Context) (T, error)

	// Close signals the waiter to stop, informs the manager to cleanup and no longer process responses
	// for this waiter.
	Close()
}

type waiterImpl[T any] struct {
	// the unique ID generated for this waiter.
	id string

	// the chan for receiving a msg.
	ch <-chan *response[T]

	// the chan used to inform the manager that this waiter is done.
	managerDoneCh chan<- string

	// when signaled indicates this waiter should stop / cleanup.
	doneSignal concurrency.Signal
}

var _ Waiter[struct{}] = (*waiterImpl[struct{}])(nil)

func newWaiter[T any](id string, ch <-chan *response[T], managerDoneCh chan<- string) *waiterImpl[T] {
	return &waiterImpl[T]{
		id:            id,
		ch:            ch,
		managerDoneCh: managerDoneCh,
		doneSignal:    concurrency.NewSignal(),
	}
}

func (w *waiterImpl[T]) ID() string {
	return w.id
}

func (w *waiterImpl[T]) Wait(ctx context.Context) (T, error) {
	var err error
	var data T
	select {
	case r, ok := <-w.ch:
		if !ok {
			// channel has been closed.
			err = ErrWaiterClosed
			break
		}

		// msg received.
		data = r.data
		err = r.err
	case <-w.doneSignal.Done():
		// close called on this waiter.
		err = ErrWaiterClosed
		w.managerDoneCh <- w.id
	case <-ctx.Done():
		err = ctx.Err()
		w.managerDoneCh <- w.id
	}

	return data, err
}

func (w *waiterImpl[T]) Close() {
	w.doneSignal.Signal()
}
