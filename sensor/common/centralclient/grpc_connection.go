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
	"github.com/stackrox/rox/sensor/common/trace"
)

// CentralConnectionFactory is responsible for establishing a gRPC connection between sensor
// and Central. Sensor used to receive an HTTP client as a parameter which was used to create
// a gRPC stream internally. This factory is now passed to sensor creation, and it can be
// more easily mocked when writing unit/integration tests.
type CentralConnectionFactory interface {
	SetCentralConnectionWithRetries(ptr *util.LazyClientConn, certLoader CertLoader)
	StopSignal() concurrency.ReadOnlyErrorSignal
	OkSignal() concurrency.ReadOnlySignal
}

type centralConnectionFactoryImpl struct {
	httpClient *Client

	stopSignal concurrency.ErrorSignal
	okSignal   concurrency.Signal
}

// NewCentralConnectionFactory returns a factory that can create a gRPC stream between Sensor and Central.
func NewCentralConnectionFactory(centralClient *Client) CentralConnectionFactory {
	return &centralConnectionFactoryImpl{
		httpClient: centralClient,

		okSignal:   concurrency.NewSignal(),
		stopSignal: concurrency.NewErrorSignal(),
	}
}

// OkSignal returns a concurrency.ReadOnlySignal that is sends signal once connection object is successfully established
// and the util.LazyClientConn pointer is swapped.
func (f *centralConnectionFactoryImpl) OkSignal() concurrency.ReadOnlySignal {
	return &f.okSignal
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

// SetCentralConnectionWithRetries will set conn pointer once the connection is set up.
// This function is supposed to be called asynchronously and allows sensor components to be
// started with an empty util.LazyClientConn. The pointer will be swapped once this
// func finishes.
// f.okSignal is used if the connection setup was successful and f.stopSignal if the
// connection setup failed. Hence, both signals are reset here.
// There is no guarantee that the connection to central will be ready when this function finishes!
// Connection setup involves the configuration of certificates, parameters, and the endpoint.
func (f *centralConnectionFactoryImpl) SetCentralConnectionWithRetries(conn *util.LazyClientConn, certLoader CertLoader) {
	// Both signals should not be in a triggered state at the same time.
	// If we run into this situation something went wrong with the handling of these signals.
	if f.stopSignal.IsDone() && f.okSignal.IsDone() {
		log.Warn("Unexpected: the stopSignal and the okSignal are both triggered")
	}
	f.stopSignal.Reset()
	f.okSignal.Reset()
	opts := []clientconn.ConnectionOption{clientconn.UseServiceCertToken(true)}

	// Waits until central is ready and has a valid license, otherwise it kills sensor by sending a signal.
	// This ping runs over HTTP and does check the health of gRPC connection.
	if err := f.pingCentral(); err != nil {
		log.Errorf("checking central status over HTTP failed: %v", err)
		f.stopSignal.SignalWithError(errors.Wrap(err, "checking central status over HTTP failed"))
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
	if p, err := f.getCentralGRPCPreferences(); err != nil {
		maxGRPCSize = env.MaxMsgSizeSetting.IntegerSetting()
		log.Warnf("Couldn't get gRPC preferences from central (%s). Using sensor env config (%d): %s", gRPCPreferences, maxGRPCSize, err)
	} else {
		maxGRPCSize = int(p.GetMaxGrpcReceiveSizeBytes())
		log.Infof("Received max gRPC size from central: %d. Overwriting default sensor value of %d.", maxGRPCSize, env.MaxMsgSizeSetting.IntegerSetting())
	}
	opts = append(opts, clientconn.MaxMsgReceiveSize(maxGRPCSize))

	// This returns a dial function, but does not call dial!
	// Thus, we cannot treat the connection as established and ready at this point.
	centralConnection, err := clientconn.AuthenticatedGRPCConnection(trace.Background(), env.CentralEndpoint.Setting(), mtls.CentralSubject, opts...)
	if err != nil {
		log.Errorf("creating the gRPC client: %v", err)
		f.stopSignal.SignalWithErrorWrap(err, "creating the gRPC client")
		return
	}

	conn.Set(centralConnection)
	f.okSignal.Signal()
	log.Info("Done setting up gRPC connection with central")
}
