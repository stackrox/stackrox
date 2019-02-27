package all

import (
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

// NewClusterPipeline returns a new instance of a ClusterPipeline that handles all event types.
func NewClusterPipeline(clusterID string, fragments ...pipeline.Fragment) pipeline.ClusterPipeline {
	return &pipelineImpl{
		fragments: fragments,
		clusterID: clusterID,
	}
}

type pipelineImpl struct {
	clusterID string
	fragments []pipeline.Fragment
}

// Run looks for one fragment (and only one) that matches the input message and runs that fragment on the message and injector.
func (s *pipelineImpl) Run(msg *central.MsgFromSensor, injector pipeline.MsgInjector) error {
	// This will only happen once per cluster because the pipeline is generated every time the streamer connects
	if msg.GetEvent().GetSynced() != nil {
		log.Infof("Received Synced message from Sensor. Determining if there is any reconciliation to be done")
		errList := errorhelpers.NewErrorList("Reconciling state")
		for _, fragment := range s.fragments {
			errList.AddError(fragment.Reconcile(s.clusterID))
		}
		return errList.ToError()
	}
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
	defer metrics.SetSensorEventRunDuration(time.Now(), common.GetMessageType(msg))
	return matchingFragment.Run(s.clusterID, msg, injector)
}

func (s *pipelineImpl) OnFinish() {
	for _, fragment := range s.fragments {
		fragment.OnFinish()
	}
}
