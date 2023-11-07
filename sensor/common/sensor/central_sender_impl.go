package sensor

import (
	"errors"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/deduperkey"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/deduper"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/metrics"
)

type centralSenderImpl struct {
	senders             []common.SensorComponent
	stopper             concurrency.Stopper
	finished            *sync.WaitGroup
	initialDeduperState map[deduperkey.Key]uint64
	observationSet      *deduper.ClosableSet
}

func (s *centralSenderImpl) Start(stream central.SensorService_CommunicateClient, initialDeduperState map[deduperkey.Key]uint64, onStops ...func(error)) {
	s.initialDeduperState = initialDeduperState
	go s.send(stream, onStops...)
}

func (s *centralSenderImpl) Stop(_ error) {
	s.stopper.Client().Stop()
}

func (s *centralSenderImpl) Stopped() concurrency.ReadOnlyErrorSignal {
	return s.stopper.Client().Stopped()
}

func (s *centralSenderImpl) forwardResponses(from <-chan *message.ExpiringMessage, to chan<- *message.ExpiringMessage) {
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
		s.finished.Done()
	}()

	wrappedStream := metrics.NewCountingEventStream(stream, "unique")
	wrappedStream = metrics.NewTimingEventStream(wrappedStream, "unique")
	wrappedStream = deduper.NewDedupingMessageStream(wrappedStream, s.initialDeduperState, s.observationSet)
	wrappedStream = metrics.NewCountingEventStream(wrappedStream, "total")
	wrappedStream = metrics.NewTimingEventStream(wrappedStream, "total")

	// NB: The centralSenderImpl reserves the right to perform arbitrary reads and writes on the returned objects.
	// The providers that send the messages below are responsible for making sure that once they send events here,
	// they do not use the objects again in any way, since that can result in a race condition caused by concurrent
	// reads and writes.
	// Ideally, if you're going to continue to hold a reference to the object, you want to proto.Clone it before
	// sending it to this function.
	componentMsgsC := make(chan *message.ExpiringMessage)
	for _, component := range s.senders {
		if responsesC := component.ResponsesC(); responsesC != nil {
			go s.forwardResponses(responsesC, componentMsgsC)
		}
	}

	for {
		var msg *message.ExpiringMessage
		var ok bool
		select {
		case msg, ok = <-componentMsgsC:
			if !ok {
				log.Info("componentMsgsC channel closed")
				s.stopper.Flow().StopWithError(errors.New("channel closed"))
				return
			}
		case <-s.stopper.Flow().StopRequested():
			log.Info("Stop flow requested")
			return
		case <-stream.Context().Done():
			log.Info("Context done")
			s.stopper.Flow().StopWithError(stream.Context().Err())
			return
		}
		if msg != nil && msg.MsgFromSensor != nil {
			// If the connection restarted, there could be messages stuck
			// in channels in Sensor pipeline, that will be attempted to
			// be streamed when connection is back up. This can mess up
			// the reconciliation in central in case some resource that
			// was deleted has an UPDATE event in some queue.
			// The event's context is canceled if the message is no longer
			// valid.
			if msg.IsExpired() {
				continue
			}

			if msg.GetEvent().GetSynced() != nil {
				unchangedIds := s.observationSet.Close()
				// Enhance sync with all the observed IDs
				msg.GetEvent().GetSynced().UnchangedIds = unchangedIds
				log.Infof("Sending synced signal to Central. Adding %d events as unchanged", len(unchangedIds))
			}

			if err := wrappedStream.Send(msg.MsgFromSensor); err != nil {
				log.Infof("Error on sending to stream: %s", err)
				s.stopper.Flow().StopWithError(err)
				return
			}
		}
	}
}
