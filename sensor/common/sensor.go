package common

import (
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	complianceLogic "github.com/stackrox/rox/sensor/common/compliance"
	"google.golang.org/grpc"
)

var (
	log = logging.LoggerForModule()

	imageDataExpiry = 1 * time.Hour
	imageDataSize   = 50000
)

// Sensor interface allows you to start and stop the consumption/production loops.
type Sensor interface {
	Start(orchestratorInput <-chan *central.SensorEvent, collectorInput <-chan *central.SensorEvent, networkFlowInput <-chan *central.NetworkFlowUpdate, complianceReturns <-chan *compliance.ComplianceReturn, output chan<- *central.SensorEnforcement)
	Stop(error)
	Wait() error
}

// NewSensor returns a new Sensor.
func NewSensor(centralConn *grpc.ClientConn, handler complianceLogic.CommandHandler) Sensor {

	return &sensor{
		conn: centralConn,

		handler: handler,

		// The ErrorSignal needs to be activated so Start() can detect callers that
		// improperly call Start() repeatedly without calling Stop() first.
		// The zero-value of ErrorSignal starts in an activated state.
		stopped: concurrency.ErrorSignal{},
	}
}

// sensor implements the Sensor interface by sending inputs to central,
// and providing the output from central asynchronously.
type sensor struct {
	conn    *grpc.ClientConn
	handler complianceLogic.CommandHandler
	stopped concurrency.ErrorSignal
}

// Start begins processing inputs and writing responses to the output channel.
// It is an error to call Start repeatedly without first calling Wait(); Wait
// itself will not return unless Stop() is called, or processing must be
// aborted for another reason (stream interrupted, channel closed, etc.).
func (s *sensor) Start(orchestratorInput <-chan *central.SensorEvent, collectorInput <-chan *central.SensorEvent, networkFlowInput <-chan *central.NetworkFlowUpdate, complianceReturns <-chan *compliance.ComplianceReturn, output chan<- *central.SensorEnforcement) {
	if !s.stopped.Reset() {
		panic("Sensor has already been started without stopping first")
	}

	s.handler.Start(complianceReturns)

	go s.sendEvents(orchestratorInput, collectorInput, networkFlowInput, s.handler.Output(), output, central.NewSensorServiceClient(s.conn))
}

// Stop stops the processing loops reading and writing to input and output, and closes the stream open with central.
func (s *sensor) Stop(err error) {
	s.handler.Stop(err)
	s.handler.Stopped().Wait()
	s.stopped.SignalWithError(err)
}

// Wait blocks until the processing has stopped.
func (s *sensor) Wait() error {
	return s.stopped.Wait()
}
