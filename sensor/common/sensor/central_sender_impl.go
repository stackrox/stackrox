package sensor

import (
	"errors"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/deduper"
	"github.com/stackrox/rox/sensor/common/metrics"
)

type centralSenderImpl struct {
	senders []common.SensorComponent
	stopper concurrency.Stopper
}

func (s *centralSenderImpl) Start(stream central.SensorService_CommunicateClient, onStops ...func(error)) {
	go s.send(stream, onStops...)
}

func (s *centralSenderImpl) Stop(_ error) {
	s.stopper.Client().Stop()
}

func (s *centralSenderImpl) Stopped() concurrency.ReadOnlyErrorSignal {
	return s.stopper.Client().Stopped()
}

func (s *centralSenderImpl) forwardResponses(from <-chan *central.MsgFromSensor, to chan<- *central.MsgFromSensor) {
	for {
		select {
		case msg, ok := <-from:
			if !ok {
				return
			}
			select {
			case to <- msg:
			case <-s.stopper.Flow().StopRequested():
				return
			}
		case <-s.stopper.Flow().StopRequested():
			return
		}
	}
}

func (s *centralSenderImpl) send(stream central.SensorService_CommunicateClient, onStops ...func(error)) {
	defer func() {
		s.stopper.Flow().ReportStopped()
		runAll(s.stopper.Client().Stopped().Err(), onStops...)
	}()

	wrappedStream := metrics.NewCountingEventStream(stream, "unique")
	wrappedStream = metrics.NewTimingEventStream(wrappedStream, "unique")
	wrappedStream = deduper.NewDedupingMessageStream(wrappedStream)
	wrappedStream = metrics.NewCountingEventStream(wrappedStream, "total")
	wrappedStream = metrics.NewTimingEventStream(wrappedStream, "total")

	// NB: The centralSenderImpl reserves the right to perform arbitrary reads and writes on the returned objects.
	// The providers that send the messages below are responsible for making sure that once they send events here,
	// they do not use the objects again in any way, since that can result in a race condition caused by concurrent
	// reads and writes.
	// Ideally, if you're going to continue to hold a reference to the object, you want to proto.Clone it before
	// sending it to this function.
	componentMsgsC := make(chan *central.MsgFromSensor)
	for _, component := range s.senders {
		if responsesC := component.ResponsesC(); responsesC != nil {
			go s.forwardResponses(responsesC, componentMsgsC)
		}
	}

	for {
		var msg *central.MsgFromSensor
		var ok bool
		select {
		case msg, ok = <-componentMsgsC:
			if !ok {
				s.stopper.Flow().StopWithError(errors.New("channel closed"))
				return
			}
		case <-s.stopper.Flow().StopRequested():
			return
		case <-stream.Context().Done():
			s.stopper.Flow().StopWithError(stream.Context().Err())
			return
		}
		if msg != nil {
			if msg.GetEvent().GetSynced() != nil {
				log.Info("Sending synced signal to Central")
			}
			log.Debug("Sending message to Central")
			if err := wrappedStream.Send(msg); err != nil {
				s.stopper.Flow().StopWithError(err)
				return
			}
		}
	}
}
