package centralclient

import (
	"context"
	"time"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/util"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/sync"
	"google.golang.org/grpc/connectivity"
)

// CentralConnectionFactory is responsible for establishing a gRPC connection between sensor
// and Central. Sensor used to receive an HTTP client as a parameter which was used to create
// a gRPC stream internally. This factory is now passed to sensor creation, and it can be
// more easily mocked when writing unit/integration tests.
type CentralConnectionFactory interface {
	SetCentralConnectionWithRetries(ptr *util.LazyClientConn, certLoader CertLoader)
	StopSignal() concurrency.ReadOnlyErrorSignal
	ConnectionState() (connectivity.State, error)
}

type centralConnectionFactoryImpl struct {
	httpClient   *Client
	currentState connectivity.State
	lastError    error
	stateMux     *sync.Mutex
	stopSignal   concurrency.ErrorSignal
}

// NewCentralConnectionFactory returns a factory that can create a gRPC stream between Sensor and Central.
func NewCentralConnectionFactory(centralClient *Client) CentralConnectionFactory {
	return &centralConnectionFactoryImpl{
		httpClient:   centralClient,
		currentState: connectivity.State(99),
		lastError:    nil,
		stateMux:     &sync.Mutex{},
		stopSignal:   concurrency.NewErrorSignal(),
	}
}

func (f *centralConnectionFactoryImpl) ConnectionState() (connectivity.State, error) {
	f.stateMux.Lock()
	defer f.stateMux.Unlock()
	return f.currentState, f.lastError
}

func (f *centralConnectionFactoryImpl) changeState(state connectivity.State, err error) bool {
	f.stateMux.Lock()
	defer f.stateMux.Unlock()
	if f.currentState != state || !errors.Is(f.lastError, err) {
		f.currentState = state
		f.lastError = err
		return true
	}
	return false
}

// StopSignal returns a concurrency.ReadOnlyErrorSignal that alerts if there is an error trying to establish gRPC connection.
func (f *centralConnectionFactoryImpl) StopSignal() concurrency.ReadOnlyErrorSignal {
	return &f.stopSignal
}

func (f *centralConnectionFactoryImpl) pingCentral() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// Ping result doesn't matter, as long as Central is reachable.
	_, err := f.httpClient.GetPing(ctx)
	return err
}

func (f *centralConnectionFactoryImpl) getCentralGRPCPreferences() (*v1.Preferences, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return f.httpClient.GetGRPCPreferences(ctx)
}

// SetCentralConnectionWithRetries will set conn pointer once the connection has a chance to be ready.
// This function is supposed to be called asynchronously and allows sensor components to be
// started with an empty util.LazyClientConn. The pointer will be swapped once this
// func finishes. The f.stopSignal is raised if the connection failed to start.
// Executing entire function (not returning early) does not guarantee that the connection is ready -
// it shows that there are no blockers for the connection to become ready soon.
func (f *centralConnectionFactoryImpl) SetCentralConnectionWithRetries(conn *util.LazyClientConn, certLoader CertLoader) {
	// Reset signal state from previous attempts
	f.stopSignal.Reset()
	opts := []clientconn.ConnectionOption{clientconn.UseServiceCertToken(true)}

	// waits until central is ready and has a valid license, otherwise it kills sensor by sending a signal
	if err := f.pingCentral(); err != nil {
		log.Errorf("pinging central over HTTP failed: %v", err)
		f.stopSignal.SignalWithError(errors.Wrap(err, "pinging central over HTTP failed"))
		return
	}

	certs := certLoader()
	if len(certs) != 0 {
		log.Infof("Add %d central CA certs to gRPC connection", len(certs))
		for _, c := range certs {
			log.Infof("Add central CA cert with CommonName: '%s'", c.Subject.CommonName)
		}
		opts = append(opts, clientconn.AddRootCAs(certs...))
	} else {
		log.Info("Did not add central CA cert to gRPC connection")
	}

	var maxGRPCSize int
	log.Info("Getting Central gRPC preferences over HTTP...")
	if p, err := f.getCentralGRPCPreferences(); err != nil {
		maxGRPCSize = env.MaxMsgSizeSetting.IntegerSetting()
		log.Warnf("Couldn't get gRPC preferences from central (%s). Using sensor env config (%d): %s", gRPCPreferences, maxGRPCSize, err)
	} else {
		maxGRPCSize = int(p.GetMaxGrpcReceiveSizeBytes())
		log.Infof("Received max gRPC size from central: %d. Overwriting default sensor value of %d.", maxGRPCSize, env.MaxMsgSizeSetting.IntegerSetting())
	}
	opts = append(opts, clientconn.MaxMsgReceiveSize(maxGRPCSize))

	centralConnection, err := clientconn.AuthenticatedGRPCConnection(context.Background(), env.CentralEndpoint.Setting(), mtls.CentralSubject, opts...)
	if err != nil {
		log.Errorf("creating the gRPC client: %v", err)
		f.changeState(centralConnection.GetState(), errors.Wrap(err, "creating the gRPC client failed"))
		return
	}

	conn.Set(centralConnection)
	f.changeState(centralConnection.GetState(), nil)
	log.Infof("Initial gRPC connection with central state: %s", centralConnection.GetState())
	// TODO: gRPC connection state may change after we leave this function!!!
}
