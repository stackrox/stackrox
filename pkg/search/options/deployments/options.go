package deployments

import (
	"github.com/stackrox/rox/pkg/postgres/schema"
)

var (
	// OptionsMap describes the options for Deployments
	OptionsMap = schema.DeploymentsSchema.OptionsMap.
		Merge(schema.ProcessIndicatorsSchema.OptionsMap).
		Merge(schema.ImagesSchema.OptionsMap)
)
