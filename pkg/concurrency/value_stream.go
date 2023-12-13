package concurrency

import (
	"sync/atomic"
	"unsafe"
)

// ValueStream is a stream of values that can be pushed sequentially by a sender and observed in the same sequence
// by any number of observers. It does not retain old values for new observers; any new observer can only start
// observing values starting from the current at the respective point in time.
//
// This class optimizes for fast, non-blocking pushes of values, suited for, e.g., critical sections (such as when
// implementing a listener pattern). It is not super well suited for high frequency pushes, as slow observers might
// cause unbounded memory growth. It is considerably slower than writing to a fully buffered channel or appending to
// a slice, but considerably faster than those options if some synchronization/contention is involved. It also
// has a number of practical benefits over approaches with explicit synchronization on the sender-side:
//   - Since senders do not have to synchronize, the performance of pushing a value does not depend on the number of
//     subscribed observers.
//   - The order of pushed elements is always preserved, since it is never necessary to spawn goroutines for pushing.
//   - The observer maintains the entire state required for reading all future values in a single iterator, which obeys
//     Go garbage collection rules. Hence, no explicit registration/deregistration of observers is necessary.
//
// Receivers operate on this class via iterators obtained via the `Iterator` method. There are two modes of iteration:
//   - If you care about processing every single value in the stream, set the `strict` parameter to `true` when calling
//     `Iterator`. Traversing the stream via the iterator is guaranteed to yield every single element that subsequently
//     gets pushed to the stream, no matter how fast the writer and how slow this reader. This mode of operation might
//     cause unbounded memory growth.
//     Should you occasionally want to skip over intermediate elements and jump to the most recent one, you can call
//     `FastForward` or `TryFastForward` on the `ValueStream`.
//   - Perhaps equally common is the case that you *do not* care about every single value in the stream, and are fine
//     skipping over values, if, e.g., your observer might take a long time to process an observation, as long as you
//     are sure to always be informed about the most recent value. For this, set the `strict` parameter to `false` when
//     calling `Iterator`. Calling `Next` on the returned iterator will always return the most recent element in the
//     stream that is newer than the current one. If the element pointed to by the current iterator is the most recent
//     one, `Next` will block until a more recent one becomes available.
//
// The following code exemplifies how to iterate over all values in the stream, in strict mode.
//
//	it := stream.Iterator(true)  // pass false for skipping mode
//	var err error
//	for it != nil {
//	  fmt.Println("Value", it.Value())
//	  time.Sleep(1)
//	  it, err = it.Next(ctx)
//	}
//	if err != nil {
//	  fmt.Fprintln("Context error aborted iteration: %v", err)
//	}
type ValueStream[T any] struct {
	curr unsafe.Pointer // always holds a *valueStreamStrictIter[T]
}

// NewValueStream initializes a value stream with an initial value.
func NewValueStream[T any](initVal T) *ValueStream[T] {
	return &ValueStream[T]{
		//#nosec G103
		curr: unsafe.Pointer(&valueStreamStrictIter[T]{
			valueStreamIterBase: valueStreamIterBase[T]{
				currVal: initVal,
				nextC:   make(chan struct{}),
			},
		}),
	}
}

// ValueStreamIter is an iterator that points to a position in a ValueStream.
// A ValueStreamIter always has a current element associated with it. It may eventually have a next element, which
// can be obtained in a context-respecting way via `Next`, or in a non-blocking way via `TryNext`. It is also possible
// to `select` on the channel returned by `Done()` in order to wait for it to become available.
type ValueStreamIter[T any] interface {

	// Value returns the value associated with this iterator.
	Value() T

	// Next fetches an iterator to the next element in the stream, or waits for the given context to expire, whatever
	// happens first. If the context expires first, the respective error is returned as the second return value. Otherwise,
	// if the next element becomes available before the context expires, it returns an iterator pointing to the next element
	// in the stream.
	Next(ctx ErrorWaitable) (ValueStreamIter[T], error)

	// TryNext attempts to obtain an iterator to the next element in the stream, or returns nil if no next element is
	// available yet. This method never blocks.
	TryNext() ValueStreamIter[T]

	// Done returns a channel indicating when the next element is available. It can be used to `select` on the next element
	// becoming available while simultaneously trying to send or receive on other channels. After the returned channel is
	// closed, `TryNext()` is guaranteed to always return a non-`nil` result.
	Done() <-chan struct{}

	isValueStreamIter(T)
}

type valueStreamIterBase[T any] struct {
	currVal T
	nextC   chan struct{}
}

func (i *valueStreamIterBase[T]) Value() T {
	return i.currVal
}

func (i *valueStreamIterBase[T]) Done() <-chan struct{} {
	return i.nextC
}

func (*valueStreamIterBase[T]) isValueStreamIter(T) {} //nolint:unused // This is required for generic magic

type valueStreamStrictIter[T any] struct {
	valueStreamIterBase[T]
	next *valueStreamStrictIter[T]
}

func (i *valueStreamStrictIter[T]) TryNext() ValueStreamIter[T] {
	select {
	case <-i.nextC:
		return i.next
	default:
		return nil
	}
}

func (i *valueStreamStrictIter[T]) Next(ctx ErrorWaitable) (ValueStreamIter[T], error) {
	select {
	case <-i.nextC:
		return i.next, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (i *valueStreamStrictIter[T]) toSkipIter(vs *ValueStream[T]) *valueStreamSkipIter[T] {
	return &valueStreamSkipIter[T]{
		valueStreamIterBase: i.valueStreamIterBase,
		vs:                  vs,
	}
}

func (i *valueStreamStrictIter[T]) withMode(vs *ValueStream[T], strict bool) ValueStreamIter[T] {
	if strict {
		return i
	}
	return i.toSkipIter(vs)
}

type valueStreamSkipIter[T any] struct {
	valueStreamIterBase[T]
	vs *ValueStream[T]
}

func (i *valueStreamSkipIter[T]) TryNext() ValueStreamIter[T] {
	strictIt := i.vs.tryFastForward(i)
	if strictIt == nil {
		return nil
	}
	return strictIt.toSkipIter(i.vs)
}

func (i *valueStreamSkipIter[T]) Next(ctx ErrorWaitable) (ValueStreamIter[T], error) {
	strictIt, err := i.vs.fastForward(ctx, i)
	if strictIt == nil {
		return nil, err
	}
	return strictIt.toSkipIter(i.vs), nil
}

// Push pushes a value onto the stream. It returns both the current value, as well as the iterator pointing to the just
// inserted value.
func (s *ValueStream[T]) Push(val T) (T, ValueStreamIter[T]) {
	newIter := &valueStreamStrictIter[T]{
		valueStreamIterBase: valueStreamIterBase[T]{
			currVal: val,
			nextC:   make(chan struct{}),
		},
	}

	//#nosec G103
	oldIter := (*valueStreamStrictIter[T])(atomic.SwapPointer(&s.curr, unsafe.Pointer(newIter)))
	oldIter.next = newIter
	close(oldIter.nextC)

	return oldIter.currVal, newIter
}

func (s *ValueStream[T]) current() *valueStreamStrictIter[T] {
	return (*valueStreamStrictIter[T])(atomic.LoadPointer(&s.curr))
}

// Iterator obtains an iterator to the current value in the stream. If strict is true, it returns an iterator that
// is guaranteed to yield every element that is subsequently pushed to the stream. Otherwise, a "skip iterator" is
// returned.
func (s *ValueStream[T]) Iterator(strict bool) ValueStreamIter[T] {
	strictIt := s.current()
	if strict {
		return strictIt
	}
	return strictIt.toSkipIter(s)
}

func (s *ValueStream[T]) fastForward(ctx ErrorWaitable, prev ValueStreamIter[T]) (*valueStreamStrictIter[T], error) {
	curr := s.current()
	if curr.nextC != prev.Done() {
		return curr, nil
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-curr.nextC:
		return s.current(), nil
	}
}

func (s *ValueStream[T]) tryFastForward(prev ValueStreamIter[T]) *valueStreamStrictIter[T] {
	curr := s.current()
	if curr.nextC != prev.Done() {
		return curr
	}
	return nil
}

// FastForward retrieves an iterator (in strict or non-strict mode) pointing to the most recent element that is newer
// than the element pointed to by prev, possibly blocking. If the context expires before another element becomes
// available, the respective error is returned along with a `nil` ValueStreamIter. This can be used to conditionally
// skip some elements when otherwise using strict iteration behavior.
func (s *ValueStream[T]) FastForward(ctx ErrorWaitable, prev ValueStreamIter[T], strict bool) (ValueStreamIter[T], error) {
	strictFfwd, err := s.fastForward(ctx, prev)
	if strictFfwd == nil {
		return nil, err
	}
	return strictFfwd.withMode(s, strict), nil
}

// TryFastForward attempts to retrieve an iterator (in strict or non-strict mode) pointing to the most recent element
// that is newer than the element pointed to by prev. If prev points to the most recent element, `nil` is returned.
func (s *ValueStream[T]) TryFastForward(prev ValueStreamIter[T], strict bool) ValueStreamIter[T] {
	strictFfwd := s.tryFastForward(prev)
	if strictFfwd == nil {
		return nil
	}
	return strictFfwd.withMode(s, strict)
}

// SubscribeChan subscribes to the sequence induced by a value stream starting iterator, writing every observed
// value (including the initial one) to a given output channel. The skip behavior is determined by the starting
// iterator.
// This function is synchronous, you most likely want to invoke it in a goroutine. It runs until the context expires
// and passes through any error from the context.
func SubscribeChan[T any](ctx ErrorWaitable, output chan<- T, startIt ValueStreamIter[T]) error {
	it := startIt

	var err error
	for err == nil && it != nil {
		select {
		case output <- it.Value():
		case <-ctx.Done():
			return ctx.Err()
		}

		it, err = it.Next(ctx)
	}
	return err
}

// ReadOnlyValueStream is an interface that limits the functionality of a ValueStream to reading elements only.
type ReadOnlyValueStream[T any] interface {
	Iterator(strict bool) ValueStreamIter[T]
	FastForward(ctx ErrorWaitable, it ValueStreamIter[T], strict bool) (ValueStreamIter[T], error)
	TryFastForward(it ValueStreamIter[T], strict bool) ValueStreamIter[T]
}
