package sensor

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common/messagestream"
	"github.com/stackrox/rox/sensor/common/metrics"
)

const (
	loggingRateLimiter = "buffered-stream"
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

func NewBufferedStream(stream messagestream.SensorMessageStream, msgC chan *central.MsgFromSensor, stopC concurrency.ReadOnlyErrorSignal) (messagestream.SensorMessageStream, <-chan error) {
	// if the capacity of the buffer is zero then we just return the inner stream
	if cap(msgC) == 0 {
		return stream, nil
	}
	ret := bufferedStream{
		buffer: msgC,
		stopC:  stopC,
		stream: stream,
	}
	errC := ret.run()
	return ret, errC
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
