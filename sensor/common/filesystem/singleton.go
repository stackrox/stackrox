package filesystem

import (
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/detector"
)

func New(detector detector.Detector, clusterEntities *clusterentities.Store) Service {
	return &serviceImpl{
		name:             "it's me!",
		writer:           nil,
		authFuncOverride: authFuncOverride,
		fsPipeline:       NewFileSystemPipeline(detector, clusterEntities),
	}
}
