package centralclient

import (
	"context"
	"crypto/x509"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/util"
	"github.com/stackrox/rox/pkg/mtls"
)

// CentralConnectionFactory is responsible for establishing a gRPC connection between sensor
// and Central. Sensor used to receive an HTTP client as a parameter which was used to create
// a gRPC stream internally. This factory is now passed to sensor creation, and it can be
// more easily mocked when writing unit/integration tests.
type CentralConnectionFactory interface {
	SetCentralConnectionWithRetries(ptr *util.LazyClientConn)
	StopSignal() *concurrency.ErrorSignal
	OkSignal() *concurrency.Signal
	Reset()
}

type centralConnectionFactoryImpl struct {
	endpoint   string
	httpClient *Client

	stopSignal concurrency.ErrorSignal
	okSignal   concurrency.Signal
}

// NewCentralConnectionFactory returns a factory that can create a gRPC stream between Sensor and Central.
func NewCentralConnectionFactory(endpoint string) (*centralConnectionFactoryImpl, error) {
	centralClient, err := NewClient(env.CentralEndpoint.Setting())
	if err != nil {
		return nil, errors.Wrap(err, "creating central client")
	}
	return &centralConnectionFactoryImpl{
		endpoint:   endpoint,
		httpClient: centralClient,

		okSignal:   concurrency.NewSignal(),
		stopSignal: concurrency.NewErrorSignal(),
	}, nil
}

// OkSignal returns a concurrency.Signal that is sends signal once connection object is successfully established
// and the util.LazyClientConn pointer is swapped.
func (f *centralConnectionFactoryImpl) OkSignal() *concurrency.Signal {
	return &f.okSignal
}

// StopSignal returns a concurrency.Signal that alerts if there is an error trying to establish gRPC connection.
func (f *centralConnectionFactoryImpl) StopSignal() *concurrency.ErrorSignal {
	return &f.stopSignal
}

// Reset signals. This should be used when re-attempting the connection in case it was broken.
func (f *centralConnectionFactoryImpl) Reset() {
	f.stopSignal.Reset()
	f.okSignal.Reset()
}

func (f *centralConnectionFactoryImpl) pingCentral() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// Ping result doesn't matter, as long as Central is reachable.
	_, err := f.httpClient.GetPing(ctx)
	return err
}

// waitUntilCentralIsReady blocks until Central responds to a ping or until the
// retry budget is exhausted, in which case the sensor is marked as stopped and
// the program will exit.
func (f *centralConnectionFactoryImpl) waitUntilCentralIsReady() error {
	exponential := backoff.NewExponentialBackOff()
	exponential.MaxElapsedTime = 5 * time.Minute
	exponential.MaxInterval = 32 * time.Second
	err := backoff.RetryNotify(func() error {
		return f.pingCentral()
	}, exponential, func(err error, d time.Duration) {
		log.Infof("Check Central status failed: %s. Retrying after %s...", err, d.Round(time.Millisecond))
	})

	if err != nil {
		return errors.Wrapf(err, "checking central status failed after %s", exponential.GetElapsedTime())
	}

	return nil
}

// getCentralTLSCerts only logs errors because this feature should not break
// sensors start-up.
func (f *centralConnectionFactoryImpl) getCentralTLSCerts() []*x509.Certificate {
	certs, err := f.httpClient.GetTLSTrustedCerts(context.Background())
	if err != nil {
		log.Warnf("Error fetching centrals TLS certs: %s", err)
	}
	return certs
}

// SetCentralConnectionWithRetries will set conn pointer once the connection is ready.
// This function is supposed to be called asynchronously and allows sensor components to be
// started with an empty util.LazyClientConn. The pointer will be swapped once this
// func finishes.
// f.okSignal is used if the connection is successful and f.stopSignal if the connection failed to start.
func (f *centralConnectionFactoryImpl) SetCentralConnectionWithRetries(conn *util.LazyClientConn) {
	opts := []clientconn.ConnectionOption{clientconn.UseServiceCertToken(true)}

	// waits until central is ready and has a valid license, otherwise it kills sensor by sending a signal
	if err := f.waitUntilCentralIsReady(); err != nil {
		f.stopSignal.SignalWithError(err)
		return
	}

	certs := f.getCentralTLSCerts()
	if len(certs) != 0 {
		log.Infof("Add %d central CA certs to gRPC connection", len(certs))
		for _, c := range certs {
			log.Infof("Add central CA cert with CommonName: '%s'", c.Subject.CommonName)
		}
		opts = append(opts, clientconn.AddRootCAs(certs...))
	} else {
		log.Infof("Did not add central CA cert to gRPC connection")
	}

	centralConnection, err := clientconn.AuthenticatedGRPCConnection(env.CentralEndpoint.Setting(), mtls.CentralSubject, opts...)
	if err != nil {
		f.stopSignal.SignalWithErrorWrap(err, "Error connecting to central")
		return
	}

	conn.Set(centralConnection)
	f.okSignal.Signal()
}
