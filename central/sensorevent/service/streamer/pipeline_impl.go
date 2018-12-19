package streamer

import (
	"github.com/stackrox/rox/central/sensorevent/service/pipeline"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

type channeledImpl struct {
	onFinish func()
}

// Start starts pulling, procesing, and pushing.
func (s *channeledImpl) Start(eventsIn <-chan *central.SensorEvent, pl pipeline.Pipeline, injector pipeline.EnforcementInjector) {
	go s.process(eventsIn, pl, injector)
}

func (s *channeledImpl) process(eventsIn <-chan *central.SensorEvent, pl pipeline.Pipeline, injector pipeline.EnforcementInjector) {
	// When we no longer have anything to process, close the sending channel.
	defer s.onFinish()

	for {
		event, ok := <-eventsIn
		// Looping stops when the output from pending events closes.
		if !ok {
			return
		}

		err := pl.Run(event, injector)
		if err != nil {
			log.Errorf("error processing event: %s %s", event.Id, err)
			continue
		}
	}
}
