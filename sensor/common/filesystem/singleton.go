package filesystem

import "github.com/stackrox/rox/sensor/common/detector"

func New(detector detector.Detector) Service {
	return &serviceImpl{
		name:             "it's me!",
		writer:           nil,
		authFuncOverride: authFuncOverride,
		fsPipeline:       NewFileSystemPipeline(detector),
	}
}
