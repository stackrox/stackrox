package common

import (
	"context"
	"fmt"
	"sync"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/images/integration"
	"github.com/stackrox/rox/pkg/listeners"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/sensor/common/cache"
	"google.golang.org/grpc"
)

var (
	log = logging.LoggerForModule()
)

// Sensor interface allows you to start and stop the consumption/production loops.
type Sensor interface {
	Start(orchestratorInput <-chan *listeners.EventWrap, collectorInput <-chan *listeners.EventWrap, output chan<- *v1.SensorEnforcement)
	Stop(error)
	Wait() error
}

// NewSensor returns a new Sensor.
func NewSensor(centralConn *grpc.ClientConn, clusterID string) Sensor {
	// This will track the set of integrations for this cluster.
	integrationSet := integration.NewSet()

	// This polls central for the integrations specific to this cluster.
	poller := integration.NewPoller(integrationSet, centralConn, clusterID)

	// This uses those integrations to enrich images.
	imageEnricher := enricher.New(integrationSet, metrics.SensorSubsystem)

	return &sensor{
		conn: centralConn,

		pendingCache:  cache.Singleton(),
		imageEnricher: imageEnricher,
		poller:        poller,

		// The Signal needs to be activated so Start() can detect callers that
		// improperly call Start() repeatedly without calling Stop() first.
		// The zero-value of Signal starts in an activated state.
		stopped: concurrency.Signal{},
	}
}

// sensor implements the Sensor interface by sending inputs to central,
// and providing the output from central asynchronously.
type sensor struct {
	conn *grpc.ClientConn

	pendingCache  *cache.PendingEvents
	imageEnricher enricher.ImageEnricher
	poller        integration.Poller

	stopped      concurrency.Signal
	err          error
	stoppingLock sync.Mutex

	cancelFunc func()
}

// Start begins processing inputs and writing responses to the output channel.
// It is an error to call Start repeatedly without first calling Wait(); Wait
// itself will not return unless Stop() is called, or processing must be
// aborted for another reason (stream interrupted, channel closed, etc.).
func (s *sensor) Start(orchestratorInput <-chan *listeners.EventWrap, collectorInput <-chan *listeners.EventWrap, output chan<- *v1.SensorEnforcement) {
	if !s.stopped.Reset() {
		panic("Sensor has already been started without stopping first")
	}
	s.err = nil

	// Create a context for the stream that we can cancel.
	var cancelCtx context.Context
	cancelCtx, s.cancelFunc = context.WithCancel(context.Background())

	// The poller must be started so that its Stop() signal
	// eventually returns.
	go s.poller.Run()

	stream, err := s.openStream(cancelCtx)
	if err != nil {
		s.Stop(fmt.Errorf("stream open: %v", err))
		return
	}

	go s.sendMessages(orchestratorInput, collectorInput, stream)
	go s.receiveMessages(output, stream)
}

func (s *sensor) openStream(ctx context.Context) (v1.SensorEventService_RecordEventClient, error) {
	cli := v1.NewSensorEventServiceClient(s.conn)
	return cli.RecordEvent(ctx)
}

// Stop stops the processing loops reading and writing to input and output, and closes the stream open with central.
func (s *sensor) Stop(err error) {
	s.stoppingLock.Lock()
	defer s.stoppingLock.Unlock()

	if s.cancelFunc != nil {
		s.cancelFunc()
		s.cancelFunc = nil
		// It's not safe to call Stop() on a poller more than once.
		s.poller.Stop()
	}

	if s.stopped.Signal() {
		// Only save the error the first time we signal; later Stop()s may be nil.
		s.err = err
	}
}

// Wait blocks until the processing has stopped.
func (s *sensor) Wait() error {
	s.stopped.Wait()

	s.stoppingLock.Lock()
	defer s.stoppingLock.Unlock()
	return s.err
}
