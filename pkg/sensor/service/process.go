package service

import (
	"time"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/listeners"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/sensor/metrics"
	"github.com/stackrox/rox/pkg/uuid"
)

func (s *serviceImpl) processProcessSignal(signal *v1.Signal) {
	indicator := &v1.ProcessIndicator{
		Id:     uuid.NewV4().String(),
		Signal: signal,
	}

	// Log lag metrics from collector
	lag := time.Now().Sub(protoconv.ConvertTimestampToTimeOrNow(indicator.GetSignal().GetTime()))
	metrics.RegisterSignalToIndicatorCreateLag(env.ClusterID.Setting(), float64(lag.Nanoseconds()))

	wrappedEvent := &listeners.EventWrap{
		SensorEvent: &v1.SensorEvent{
			Id:     indicator.GetId(),
			Action: v1.ResourceAction_CREATE_RESOURCE,
			Resource: &v1.SensorEvent_ProcessIndicator{
				ProcessIndicator: indicator,
			},
		},
	}
	s.pushEventToChannel(wrappedEvent)
}
