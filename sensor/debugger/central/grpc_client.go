package central

import (
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/grpc/util"
	roxLogging "github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/centralclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

var roxlog = roxLogging.LoggerForModule()

type fakeGRPCClient struct {
	okSig             concurrency.Signal
	stopSig           concurrency.ErrorSignal
	transientErrorSig concurrency.ErrorSignal
	conn              *grpc.ClientConn

	// connMtx guards the connection and the state (as the state is a connection state)
	connMtx      *sync.Mutex
	currentState connectivity.State
}

// FakeGRPCFactory implements centralclient.CentralConnectionFactory interface and additional functions for testing
// purposes only.
type FakeGRPCFactory interface {
	centralclient.CentralConnectionFactory
	OverwriteCentralConnection(newConn *grpc.ClientConn)
}

// MakeFakeConnectionFactory creates the fake gRPC client object given a gRPC connection.
func MakeFakeConnectionFactory(c *grpc.ClientConn) *fakeGRPCClient {
	roxlog.Infof("Making new fake connection factory (this resets signals)")
	return &fakeGRPCClient{
		conn:              c,
		stopSig:           concurrency.NewErrorSignal(),
		transientErrorSig: concurrency.NewErrorSignal(),
		okSig:             concurrency.NewSignal(),
		connMtx:           &sync.Mutex{},
		currentState:      99, // invalid state
	}
}

func (f *fakeGRPCClient) StopSignal() concurrency.ReadOnlyErrorSignal {
	return &f.stopSig
}

func (f *fakeGRPCClient) ConnectionState() (connectivity.State, error) {
	f.connMtx.Lock()
	defer f.connMtx.Unlock()
	return f.currentState, nil
}

func (f *fakeGRPCClient) OverwriteCentralConnection(newConn *grpc.ClientConn) {
	concurrency.WithLock(f.connMtx, func() {
		f.conn = newConn
		f.currentState = newConn.GetState()
	})
}

// SetCentralConnectionWithRetries is the implementation of the concurrent function SetCentralConnectionWithRetries
// that sensor uses to set the gRPC connection to all its components. Present test version simply.
func (f *fakeGRPCClient) SetCentralConnectionWithRetries(ptr *util.LazyClientConn, _ centralclient.CertLoader) {
	defer roxlog.Infof("SetCentralConnectionWithRetries: done")
	roxlog.Infof("SetCentralConnectionWithRetries: waiting for conn mutex")
	concurrency.WithLock(f.connMtx, func() {
		ptr.Set(f.conn)

		roxlog.Infof("SetCentralConnectionWithRetries: waiting for state mutex")
		if f.currentState != f.conn.GetState() {
			roxlog.Infof("State change from %s to %s", f.currentState.String(), f.conn.GetState().String())
			f.currentState = f.conn.GetState()
		} else {
			roxlog.Infof("No State change and is %s", f.currentState.String())
		}
	})
}

// Reset signals
func (f *fakeGRPCClient) Reset() {
	roxlog.Infof("Resetting fake grpc client")
	concurrency.WithLock(f.connMtx, func() {
		f.currentState = 99
	})
}
