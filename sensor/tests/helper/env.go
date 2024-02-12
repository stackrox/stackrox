package helper

import "github.com/stackrox/rox/pkg/env"

var (
	// UseRealCollector defines whether the test should expect a real collector or not.
	UseRealCollector = env.RegisterBooleanSetting("ROX_USE_REAL_COLLECTOR_IN_TEST", false)
)
