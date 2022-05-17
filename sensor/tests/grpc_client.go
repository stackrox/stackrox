package tests

import (
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/grpc/util"
	"google.golang.org/grpc"
)

type fakeGRPCClient struct {
	okSig   concurrency.Signal
	stopSig concurrency.ErrorSignal
	conn    *grpc.ClientConn
}

func makeFakeConnectionFactory(c *grpc.ClientConn) *fakeGRPCClient {
	return &fakeGRPCClient{
		conn:    c,
		stopSig: concurrency.NewErrorSignal(),
		okSig:   concurrency.NewSignal(),
	}
}

func (f *fakeGRPCClient) SetCentralConnectionWithRetries(ptr *util.LazyClientConn) {
	ptr.Set(f.conn)
	f.okSig.Signal()
}

func (f *fakeGRPCClient) StopSignal() concurrency.ErrorSignal {
	return f.stopSig
}

func (f *fakeGRPCClient) OkSignal() concurrency.Signal {
	return f.okSig
}
