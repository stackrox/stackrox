package streamer

import (
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

type channeledImpl struct {
	onFinish func()
}

// Start starts pulling, procesing, and pushing.
func (s *channeledImpl) Start(eventsIn <-chan *central.MsgFromSensor, pl pipeline.Pipeline, injector pipeline.MsgInjector) {
	go s.process(eventsIn, pl, injector)
}

func (s *channeledImpl) process(msgsIn <-chan *central.MsgFromSensor, pl pipeline.Pipeline, injector pipeline.MsgInjector) {
	// When we no longer have anything to process, close the sending channel.
	defer s.onFinish()

	for {
		msg, ok := <-msgsIn
		// Looping stops when the output from pending events closes.
		if !ok {
			return
		}

		err := pl.Run(msg, injector)
		if err != nil {
			log.Errorf("error processing msg from sensor: %s", err)
			continue
		}
	}
}
