package manager

import "github.com/stackrox/rox/pkg/env"

var (
	// debugCloseHistoricalEntities is an experimental setting - DO NOT CHANGE IN PRODUCTION.
	debugCloseHistoricalEntities = env.RegisterBooleanSetting("ROX_DEBUG_CLOSE_HISTORICAL_ENTITIES", true)
)
