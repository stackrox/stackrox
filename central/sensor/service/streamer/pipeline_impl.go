package streamer

import (
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

type channeledImpl struct {
	onFinish func(error)
}

// Start starts pulling, procesing, and pushing.
func (s *channeledImpl) Start(eventsIn <-chan *central.MsgFromSensor, pl pipeline.Pipeline, injector pipeline.MsgInjector, stopSig concurrency.ReadOnlyErrorSignal) {
	go s.process(eventsIn, pl, injector, stopSig)
}

func (s *channeledImpl) process(msgsIn <-chan *central.MsgFromSensor, pl pipeline.Pipeline, injector pipeline.MsgInjector, stopSig concurrency.ReadOnlyErrorSignal) {
	err := s.doProcess(msgsIn, pl, injector, stopSig)
	// When we no longer have anything to process, close the sending channel.
	s.onFinish(err)
}

func (s *channeledImpl) doProcess(msgsIn <-chan *central.MsgFromSensor, pl pipeline.Pipeline, injector pipeline.MsgInjector, stopSig concurrency.ReadOnlyErrorSignal) error {
	for {
		select {
		case msg, ok := <-msgsIn:
			// Looping stops when the output from pending events closes.
			if !ok {
				return nil
			}

			err := pl.Run(msg, injector)
			if err != nil {
				log.Errorf("error processing msg from sensor: %s", err)
				continue
			}
		case <-stopSig.Done():
			return stopSig.Err()
		}
	}
}
