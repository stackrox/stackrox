package waiter

import (
	"context"
	"errors"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/uuid"
)

const (
	maxIDCollisions = 5
)

var (
	// ErrTooManyCollisions the ID generator produced too many non-unique IDs.
	ErrTooManyCollisions = errors.New("too many id collisions")

	// ErrManagerShutdown manager has been shutdown and therefore cannot process requests.
	ErrManagerShutdown = errors.New("manager is shutdown")

	log = logging.LoggerForModule()
)

type response[T any] struct {
	id   string
	data T
	err  error
}

// Manager builds waiters and delivers messages to waiters from async publishers.
//
//go:generate mockgen-wrapper
type Manager[T any] interface {
	// Start spawns a goroutine that will run forever or until ctx.Done() delivering
	// messages to waiters. This should be called before anything else (when not
	// called invocations to Send, Wait, etc. will block forever).
	Start(ctx context.Context)

	// Send sends data and err to the waiter with the provided ID. Only the first send
	// will be delivered, subsequent sends are no-ops.
	Send(id string, data T, err error) error

	// NewWaiter creates a waiter with a unique ID.
	NewWaiter() (Waiter[T], error)
}

type managerImpl[T any] struct {
	// waiters holds a waiter's id and the channel to send responses on.
	waiters map[string]chan *response[T]

	// waitersMu ensures only one goroutine is manipulating the waiters map at a time.
	waitersMu sync.Mutex

	// responseCh is the global channel that receives all responses meant for waiting waiters.
	responseCh chan *response[T]

	// doneWaiterCh ids sent on this channel will be cleaned up by the manager.
	doneWaiterCh chan string

	// managerShutdownSignal is used to indicate the manager is shutdown and no more messages should be processed.
	managerShutdownSignal concurrency.Signal
}

// NewManager creates a new waiter Manager.
func NewManager[T any]() *managerImpl[T] {
	return &managerImpl[T]{
		waiters:               make(map[string]chan *response[T]),
		responseCh:            make(chan *response[T]),
		doneWaiterCh:          make(chan string),
		managerShutdownSignal: concurrency.NewSignal(),
	}
}

// Start spawns a goroutine that will run forever or until ctx.Done() delivering
// messages to waiters. This should be called before anything else (when not
// called invocations to Send, Wait, etc. will block forever).
func (w *managerImpl[T]) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				// ensure are no more sends.
				w.managerShutdownSignal.Signal()

				// inform the waiters.
				w.closeWaiters()
				return
			case r := <-w.responseCh:
				waiterCh, found := w.removeWaiter(r.id)
				if !found {
					log.Debugf("Received response for non-existent waiter %q", r.id)
					continue
				}

				// to prevent blocking, waiterCh should be buffered with
				// size of 1 (each waiterCh should only ever receive 1 msg)
				// (alternatively use a goroutine).
				waiterCh <- r
			case id := <-w.doneWaiterCh:
				// the waiter has been closed or canceled.
				w.removeWaiter(id)
			}
		}
	}()
}

// Send sends data and err to the waiter with the provided id.
func (w *managerImpl[T]) Send(id string, data T, err error) error {
	if w.managerShutdownSignal.IsDone() {
		return ErrManagerShutdown
	}

	// check again if the manager is shutdown, return if so
	// otherwise write the message to the proper channel.
	select {
	case <-w.managerShutdownSignal.Done():
		return ErrManagerShutdown
	case w.responseCh <- &response[T]{
		id:   id,
		data: data,
		err:  err,
	}:
	}

	return nil
}

// NewWaiter creates a waiter with a unique ID. A call to Wait() will complete when
// a response is published using this ID.
func (w *managerImpl[T]) NewWaiter() (Waiter[T], error) {
	// if the manager is shutdown, error out.
	if w.managerShutdownSignal.IsDone() {
		return nil, ErrManagerShutdown
	}

	// otherwise setup the new waiter.
	for i := 0; i < maxIDCollisions; i++ {
		id := uuid.NewV4().String()

		waiterCh, created := w.addWaiter(id)
		if created {
			return newWaiter(id, waiterCh, w.doneWaiterCh), nil
		}
	}

	return nil, ErrTooManyCollisions
}

// addWaiter will return true if the waiter chan was successfully created, otherwise
// will return false indicating an id collision.
func (w *managerImpl[T]) addWaiter(id string) (chan *response[T], bool) {
	w.waitersMu.Lock()
	defer w.waitersMu.Unlock()

	if _, ok := w.waiters[id]; ok {
		return nil, false
	}

	// create a buffered channel of size 1 so that the loop initiated by w.Start()
	// will not have to wait for this channel to be read (making it non-blocking).
	ch := make(chan *response[T], 1)

	w.waiters[id] = ch
	return ch, true
}

// removeWaiter will return true if a waiter was found and removed with the waiters
// chan, otherwise will return false indicating no waiter found.
func (w *managerImpl[T]) removeWaiter(id string) (chan *response[T], bool) {
	w.waitersMu.Lock()
	defer w.waitersMu.Unlock()

	waiterCh, ok := w.waiters[id]
	if !ok {
		return nil, false
	}

	delete(w.waiters, id)

	return waiterCh, true
}

func (w *managerImpl[T]) closeWaiters() {
	w.waitersMu.Lock()
	defer w.waitersMu.Unlock()

	for id, ch := range w.waiters {
		close(ch)
		delete(w.waiters, id)
	}

	w.waiters = nil
}

// len is used in tests to verify waiter cleanup, added to avoid race condition.
func (w *managerImpl[T]) len() int {
	w.waitersMu.Lock()
	defer w.waitersMu.Unlock()

	return len(w.waiters)
}
