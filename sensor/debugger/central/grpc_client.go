package central

import (
	"github.com/stackrox/stackrox/pkg/concurrency"
	"github.com/stackrox/stackrox/pkg/grpc/util"
	"google.golang.org/grpc"
)

type fakeGRPCClient struct {
	okSig   concurrency.Signal
	stopSig concurrency.ErrorSignal
	conn    *grpc.ClientConn
}

// MakeFakeConnectionFactory creates the fake gRPC client object given a gRPC connection.
func MakeFakeConnectionFactory(c *grpc.ClientConn) *fakeGRPCClient {
	return &fakeGRPCClient{
		conn:    c,
		stopSig: concurrency.NewErrorSignal(),
		okSig:   concurrency.NewSignal(),
	}
}

// SetCentralConnectionWithRetries is the implementation of the concurrent function SetCentralConnectionWithRetries
// that sensor uses to set the gRPC connection to all its components. Present test version simply.
func (f *fakeGRPCClient) SetCentralConnectionWithRetries(ptr *util.LazyClientConn) {
	ptr.Set(f.conn)
	f.okSig.Signal()
}

// StopSignal returns a signal that is sent if there is an error.
// This signal is never called.
func (f *fakeGRPCClient) StopSignal() *concurrency.ErrorSignal {
	return &f.stopSig
}

// OkSignal returns a signal that is sent if connection was swapped.
// This signal is triggered instantly on calling SetCentralConnectionWithRetries.
func (f *fakeGRPCClient) OkSignal() *concurrency.Signal {
	return &f.okSig
}
