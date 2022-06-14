package service

import (
	"github.com/stackrox/stackrox/pkg/env"
)

var (
	defaultMaxNumberOfDeploymentsInGraph = 2000
	maxNumberOfDeploymentsInGraphEnv     = env.RegisterIntegerSetting("ROX_MAX_DEPLOYMENTS_NETWORK_GRAPH", defaultMaxNumberOfDeploymentsInGraph)
)
