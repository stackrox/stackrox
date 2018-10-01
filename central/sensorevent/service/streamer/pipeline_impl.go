package streamer

import (
	"github.com/stackrox/rox/central/sensorevent/service/pipeline"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

type channeledImpl struct {
	onFinish func()
}

// Start starts pulling, procesing, and pushing.
func (s *channeledImpl) Start(eventsIn <-chan *v1.SensorEvent, pl pipeline.Pipeline, enforcementsOut chan<- *v1.SensorEnforcement) {
	go s.process(eventsIn, pl, enforcementsOut)
}

func (s *channeledImpl) process(eventsIn <-chan *v1.SensorEvent, pl pipeline.Pipeline, enforcementsOut chan<- *v1.SensorEnforcement) {
	// When we no longer have anything to process, close the sending channel.
	defer s.onFinish()

	for {
		event, ok := <-eventsIn
		// Looping stops when the output from pending events closes.
		if !ok {
			return
		}

		enforcement, err := pl.Run(event)
		if err != nil {
			log.Errorf("error processing event: %s %s", event.Id, err)
			continue
		}
		if enforcement == nil {
			log.Debugf("no enforcement action taken for: %s", event.Id)
			continue
		}
		enforcementsOut <- enforcement
	}
}
