package sensor

import (
	"context"

	"github.com/stackrox/rox/generated/api/v1"
	grpcLib "google.golang.org/grpc"
)

// newCentralStream returns a new Sensor.
func newCentralStream(conn *grpcLib.ClientConn) *centralStreamImpl {
	return &centralStreamImpl{
		conn: conn,
	}
}

// sensorImpl implements the sensor interface by sending inputs to central, and providing the output from central asynchronously.
type centralStreamImpl struct {
	conn *grpcLib.ClientConn

	cancelFunction func()
}

// Helper functions that open/close the stream with central and start/stop the processing loops.
/////////////////////////////////////////////////////////////////////////////////////////////////

// Opens the bidirectional grpc stream with central.
func (p *centralStreamImpl) openStream() (v1.SensorEventService_RecordEventClient, error) {
	if p.cancelFunction != nil {
		panic("do not open a stream that is already open")
	}

	// Create a context for the stream that we can cancel.
	cancellable, cancelFunction := context.WithCancel(context.Background())

	// Open the stream, and store our cancel function.
	cli := v1.NewSensorEventServiceClient(p.conn)
	stream, err := cli.RecordEvent(cancellable)
	if err != nil {
		cancelFunction()
		return stream, err
	}
	p.cancelFunction = cancelFunction
	return stream, nil
}

// Closes the bidirectional grpc stream with central.
func (p *centralStreamImpl) closeStream() {
	// Call the cancel function we stored, and reset it to nil, service will see the cancelled context and stop processing,
	p.cancelFunction()
	p.cancelFunction = nil
}
