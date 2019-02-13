package sensor

import (
	"errors"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/listeners"
	"github.com/stackrox/rox/sensor/common/compliance"
	"github.com/stackrox/rox/sensor/common/deduper"
	"github.com/stackrox/rox/sensor/common/metrics"
	networkConnManager "github.com/stackrox/rox/sensor/common/networkflow/manager"
	"github.com/stackrox/rox/sensor/common/signal"
)

type centralSenderImpl struct {
	// Generate messages to be sent to central.
	listener             listeners.Listener
	signalService        signal.Service
	networkConnManager   networkConnManager.Manager
	scrapeCommandHandler compliance.CommandHandler

	stopC    concurrency.ErrorSignal
	stoppedC concurrency.ErrorSignal
}

func (s *centralSenderImpl) Start(stream central.SensorService_CommunicateClient, onStops ...func(error)) {
	go s.send(stream, onStops...)
}

func (s *centralSenderImpl) Stop(err error) {
	s.stopC.SignalWithError(err)
}

func (s *centralSenderImpl) Stopped() concurrency.ReadOnlyErrorSignal {
	return &s.stoppedC
}

func (s *centralSenderImpl) send(stream central.SensorService_CommunicateClient, onStops ...func(error)) {
	defer func() {
		s.stoppedC.SignalWithError(s.stopC.Err())
		runAll(s.stopC.Err(), onStops...)
	}()

	wrappedStream := metrics.NewCountingEventStream(stream, "unique")
	wrappedStream = deduper.NewDedupingMessageStream(wrappedStream)
	wrappedStream = metrics.NewCountingEventStream(wrappedStream, "total")

	for {
		var msg *central.MsgFromSensor

		select {
		case sig, ok := <-s.signalService.Indicators():
			if !ok {
				s.stopC.SignalWithError(errors.New("signals channel closed"))
				return
			}
			msg = &central.MsgFromSensor{
				Msg: &central.MsgFromSensor_Event{
					Event: sig,
				},
			}
		case evt, ok := <-s.listener.Events():
			if !ok {
				s.stopC.SignalWithError(errors.New("orchestrator events channel closed"))
				return
			}
			msg = &central.MsgFromSensor{
				Msg: &central.MsgFromSensor_Event{
					Event: evt,
				},
			}
		case flowUpdate, ok := <-s.networkConnManager.FlowUpdates():
			if !ok {
				s.stopC.SignalWithError(errors.New("flow updates channel closed"))
				return
			}
			msg = &central.MsgFromSensor{
				Msg: &central.MsgFromSensor_NetworkFlowUpdate{
					NetworkFlowUpdate: flowUpdate,
				},
			}
		case scrapeUpdate, ok := <-s.scrapeCommandHandler.Output():
			if !ok {
				s.stopC.SignalWithError(errors.New("scrape command handler channel closed"))
				return
			}
			msg = &central.MsgFromSensor{
				Msg: &central.MsgFromSensor_ScrapeUpdate{
					ScrapeUpdate: scrapeUpdate,
				},
			}
		case <-s.stopC.Done():
			return
		case <-stream.Context().Done():
			s.stopC.SignalWithError(stream.Context().Err())
			return
		}

		if msg != nil {
			if err := wrappedStream.Send(msg); err != nil {
				s.stopC.SignalWithError(err)
				return
			}
		}
	}
}
