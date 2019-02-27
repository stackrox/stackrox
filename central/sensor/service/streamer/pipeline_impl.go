package streamer

import (
	"errors"

	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

type channeledImpl struct {
	pl pipeline.ClusterPipeline

	stopC    concurrency.ErrorSignal
	stoppedC concurrency.ErrorSignal
}

// Start starts pulling, procesing, and pushing.
func (s *channeledImpl) Start(msgsIn <-chan *central.MsgFromSensor, injector pipeline.MsgInjector, dependents ...Stoppable) {
	go s.process(msgsIn, injector, dependents...)
}

func (s *channeledImpl) Stop(err error) bool {
	return s.stopC.SignalWithError(err)
}

func (s *channeledImpl) Stopped() concurrency.ReadOnlyErrorSignal {
	return &s.stoppedC
}

func (s *channeledImpl) process(msgsIn <-chan *central.MsgFromSensor, injector pipeline.MsgInjector, dependents ...Stoppable) {
	defer func() {
		s.pl.OnFinish()
		s.stoppedC.SignalWithError(s.stopC.Err())
		StopAll(s.stoppedC.Err(), dependents...)
	}()

	for !s.stopC.IsDone() {
		select {
		case msg, ok := <-msgsIn:
			// Looping stops when the output from pending events closes.
			if !ok {
				s.stopC.SignalWithError(errors.New("channel unexpectedly closed"))
				return
			}

			err := s.pl.Run(msg, injector)
			if err != nil {
				log.Errorf("error processing msg from sensor: %s", err)
			}

		case <-s.stopC.Done():
			return
		}
	}
}
