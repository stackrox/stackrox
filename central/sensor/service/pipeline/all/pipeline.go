package all

import (
	"fmt"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

// NewPipeline returns a new instance of a Pipeline that handles all event types.
func NewPipeline(fragments ...pipeline.Fragment) pipeline.Pipeline {
	return &pipelineImpl{
		fragments: fragments,
	}
}

type pipelineImpl struct {
	fragments []pipeline.Fragment
}

// Run looks for one fragment (and only one) that matches the input message and runs that fragment on the message and injector.
func (s *pipelineImpl) Run(msg *central.MsgFromSensor, injector pipeline.MsgInjector) error {
	var matchingFragment pipeline.Fragment
	for _, fragment := range s.fragments {
		if fragment.Match(msg) {
			if matchingFragment == nil {
				matchingFragment = fragment
			} else {
				return fmt.Errorf("multiple pipeline fragments matched: %s", proto.MarshalTextString(msg))
			}
		}
	}
	if matchingFragment == nil {
		return fmt.Errorf("no pipeline present to process message: %s", proto.MarshalTextString(msg))
	}
	return matchingFragment.Run(msg, injector)
}
