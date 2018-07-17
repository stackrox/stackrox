package sensor

import (
	"bitbucket.org/stack-rox/apollo/pkg/enforcers"
	"bitbucket.org/stack-rox/apollo/pkg/listeners"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/sources"
	grpcLib "google.golang.org/grpc"
)

var (
	log = logging.LoggerForModule()
)

// Sensor interface allows you to start and stop the consumption/production loops.
type Sensor interface {
	Start(input <-chan *listeners.EventWrap, output chan<- *enforcers.DeploymentEnforcement) error
	Stop()
}

// NewSensor returns a new Sensor.
func NewSensor(imageIntegrationPoller *sources.Client, conn *grpcLib.ClientConn) Sensor {
	return &sensorImpl{
		centralStream: newCentralStream(conn),
		processLoops:  newProcessLoops(imageIntegrationPoller),
	}
}

// sensorImpl implements the sensor interface by sending inputs to central, and providing the output from central asynchronously.
type sensorImpl struct {
	centralStream *centralStreamImpl
	processLoops  *processLoopsImpl
}

// Start begins listening to the input channel, processing the item, and writing them out to the output channel.
func (p *sensorImpl) Start(input <-chan *listeners.EventWrap, output chan<- *enforcers.DeploymentEnforcement) error {
	stream, err := p.centralStream.openStream()
	if err != nil {
		return err
	}
	p.processLoops.startLoops(input, stream, output)
	return nil
}

// Stop stops the processing loops reading and writing to input and output, and closes the stream open with central.
func (p *sensorImpl) Stop() {
	p.processLoops.stopLoops()
	p.centralStream.closeStream()
}
