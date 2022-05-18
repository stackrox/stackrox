package connection

import (
	"context"
	"crypto/x509"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/util"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/sensor/common/centralclient"
)

var (
	log = logging.LoggerForModule()
)

// This package is responsible for establishing a gRPC connection between sensor
// and Central. The code here was previously part of sensor. Sensor used to
// receive an HTTP client as a parameter and create the gRPC connection internally.
// The idea here is to extract this behavior to outside of sensor so mocking becomes
// easier.

type ConnectionFactory interface {
	SetCentralConnectionWithRetries(ptr *util.LazyClientConn)
	StopSignal() concurrency.ErrorSignal
	OkSignal() concurrency.Signal
}


type connectionFactoryImpl struct {
	endpoint   string
	httpClient *centralclient.Client

	stopSignal concurrency.ErrorSignal
	okSignal   concurrency.Signal
}

func NewConnectionFactor(endpoint string) (*connectionFactoryImpl, error) {
	centralClient, err := centralclient.NewClient(env.CentralEndpoint.Setting())
	if err != nil {
		return nil, errors.Wrap(err, "creating central client")
	}
	return &connectionFactoryImpl{
		endpoint:   endpoint,
		httpClient: centralClient,

		okSignal:   concurrency.NewSignal(),
		stopSignal: concurrency.NewErrorSignal(),
	}, nil
}

func (f *connectionFactoryImpl) OkSignal() concurrency.Signal {
	return f.okSignal
}

func (f *connectionFactoryImpl) StopSignal() concurrency.ErrorSignal {
	return f.stopSignal
}

func (f *connectionFactoryImpl) pollMetadata() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// Metadata result doesn't matter, as long as central is reachable.
	_, err := s.centralRestClient.GetMetadata(ctx)
	return err
}

// waitUntilCentralIsReady blocks until central responds with a valid license status on its metadata API,
// or until the retry budget is exhausted (in which case the sensor is marked as stopped and the program
// will exit).
func (f *connectionFactoryImpl) waitUntilCentralIsReady() {
	exponential := backoff.NewExponentialBackOff()
	exponential.MaxElapsedTime = 5 * time.Minute
	exponential.MaxInterval = 32 * time.Second
	err := backoff.RetryNotify(func() error {
		return f.pollMetadata()
	}, exponential, func(err error, d time.Duration) {
		log.Infof("Check Central status failed: %s. Retrying after %s...", err, d.Round(time.Millisecond))
	})

	// TODO: handle error
	if err != nil {
		f.stopSignal.SignalWithErrorWrapf(err, "checking central status failed after %s", exponential.GetElapsedTime())
	}
}

// getCentralTLSCerts only logs errors because this feature should not break
// sensors start-up.
func (f *connectionFactoryImpl) getCentralTLSCerts() []*x509.Certificate {
	certs, err := f.httpClient.GetTLSTrustedCerts(context.Background())
	if err != nil {
		log.Warnf("Error fetching centrals TLS certs: %s", err)
	}
	return certs
}

func (f *connectionFactoryImpl) SetCentralConnectionWithRetries(ptr *util.LazyClientConn) {
	opts := []clientconn.ConnectionOption{clientconn.UseServiceCertToken(true)}

	// waits until central is ready and has a valid license, otherwise it kills sensor by sending a signal
	f.waitUntilCentralIsReady()

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

	// NOTE: This creates the gRPC connection used for the messages exchanges between sensor and central
	centralConnection, err := clientconn.AuthenticatedGRPCConnection(env.CentralEndpoint.Setting(), mtls.CentralSubject, opts...)
	// TODO: Handle error
	if err != nil {
		f.stopSignal.SignalWithErrorWrap(err, "Error connecting to central")
		return
	}

	ptr.Set(centralConnection)
	f.okSignal.Signal()
}
