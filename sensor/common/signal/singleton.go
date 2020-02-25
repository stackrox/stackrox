package signal

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/common/processfilter"
	"github.com/stackrox/rox/sensor/common/processsignal"
)

// New creates a new signal service
func New(detector detector.Detector) Service {
	indicators := make(chan *central.MsgFromSensor)

	return &serviceImpl{
		queue:           make(chan *v1.Signal, maxBufferSize),
		indicators:      indicators,
		processPipeline: processsignal.NewProcessPipeline(indicators, clusterentities.StoreInstance(), processfilter.Singleton(), detector),
	}
}
