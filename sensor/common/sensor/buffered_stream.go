package sensor

import (
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common/messagestream"
	"github.com/stackrox/rox/sensor/common/metrics"
)

const (
	loggingRateLimiter = "buffered-stream"
)

var (
	stopTimeout = 10 * time.Second
)

type bufferedStream struct {
	buffer chan *central.MsgFromSensor
	stopC  concurrency.ReadOnlyErrorSignal
	stream messagestream.SensorMessageStream
}

func (s bufferedStream) Send(msg *central.MsgFromSensor) error {
	select {
	case s.buffer <- msg:
		metrics.ResponsesChannelAdd(msg)
	default:
		// The buffer is full, we drop the message and return
		logging.GetRateLimitedLogger().WarnL(loggingRateLimiter, "Dropping message in the gRPC stream")
		metrics.ResponsesChannelDrop(msg)
		return nil
	}
	return nil
}

// NewBufferedStream returns a SensorMessageStream 'buffStream' that implements a buffer for MsgFromSensor messages.
// If the buffer limit is reached, new messages will be dropped.
// buffStream is the bufferedStream.
// errC is a channel containing errors coming from the internal Send function.
// onStop is a function that waits for errC to be closed. This is needed to
// avoid races between the Send and the CloseSend functions.
func NewBufferedStream(stream messagestream.SensorMessageStream, msgC chan *central.MsgFromSensor, stopC concurrency.ReadOnlyErrorSignal) (buffStream messagestream.SensorMessageStream, errC <-chan error, onStop func() error) {
	// if the capacity of the buffer is zero then we just return the inner stream
	if cap(msgC) == 0 {
		return stream, nil, nil
	}
	ret := bufferedStream{
		buffer: msgC,
		stopC:  stopC,
		stream: stream,
	}
	errC = ret.run()
	return ret, errC, func() error {
		for {
			select {
			case _, ok := <-errC:
				if !ok {
					return nil
				}
			case <-time.After(stopTimeout):
				// If we reach this timeout we could have a deadlock in the gRPC stream
				return errors.New("timeout waiting for the buffered stream to stop")
			}
		}
	}
}

func (s bufferedStream) run() <-chan error {
	errC := make(chan error)
	go func() {
		defer close(errC)
		for {
			select {
			case <-s.stopC.Done():
				return
			case msg, ok := <-s.buffer:
				if !ok {
					return
				}
				metrics.ResponsesChannelRemove(msg)
				select {
				case errC <- s.stream.Send(msg):
				case <-s.stopC.Done():
					return
				}
			}
		}
	}()
	return errC
}
