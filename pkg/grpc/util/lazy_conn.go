package util

import (
	"context"
	"sync/atomic"

	"google.golang.org/grpc"
)

type lazyConnState struct {
	waitC chan struct{}
	cc    grpc.ClientConnInterface
}

func makeState(cc grpc.ClientConnInterface) *lazyConnState {
	if cc != nil {
		return &lazyConnState{cc: cc}
	}
	return &lazyConnState{waitC: make(chan struct{})}
}

// LazyClientConn implements the grpc.ClientConnInterface and delegates calls to an underlying grpc.ClientConnInterface
// object specified by `Set(...)` invocations. Initially, and after a `Set(nil)` call, operations will wait (in a
// context-aware manner) until such a connection becomes available via a `Set` invocation with a non-nil argument
// from a concurrent goroutine.
type LazyClientConn struct {
	state atomic.Pointer[lazyConnState]
}

// NewLazyClientConn creates and returns a new LazyClientConn that does not have an underlying delegate client
// connection. Until `Set` is called, operations will block.
// Note: If you want a fail-fast behavior until a connection is available, you need to implement your own
// client conn type that returns an error right away.
func NewLazyClientConn() *LazyClientConn {
	lcc := &LazyClientConn{}
	lcc.state.Store(makeState(nil))
	return lcc
}

// Set specifies the client connection to delegate to, or nil. All goroutines currently waiting for a connection to
// become available will be woken up, although they might block again soon afterwards if nil was specified.
func (c *LazyClientConn) Set(cc grpc.ClientConnInterface) {
	newState := makeState(cc)
	oldState := c.state.Swap(newState)
	if oldState.waitC != nil {
		oldState.cc = cc
		close(oldState.waitC)
	}
}

func (c *LazyClientConn) getClientConn(ctx context.Context) (grpc.ClientConnInterface, error) {
	for {
		st := c.state.Load()
		if st.waitC == nil {
			return st.cc, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-st.waitC:
			if st.cc != nil {
				return st.cc, nil
			}
		}
	}
}

// Invoke waits for a delegate ClientConnInterface to become available and delegates the call to that.
func (c *LazyClientConn) Invoke(ctx context.Context, method string, args interface{}, reply interface{}, opts ...grpc.CallOption) error {
	cc, err := c.getClientConn(ctx)
	if err != nil {
		return err
	}
	return cc.Invoke(ctx, method, args, reply, opts...)
}

// NewStream waits for a delegate ClientConnInterface to become available and delegates the call to that.
func (c *LazyClientConn) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	cc, err := c.getClientConn(ctx)
	if err != nil {
		return nil, err
	}
	return cc.NewStream(ctx, desc, method, opts...)
}
