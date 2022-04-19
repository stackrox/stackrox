package service

import (
	"strconv"

	"github.com/stackrox/rox/pkg/env"
)

var (
	defaultMaxNumberOfDeploymentsInGraph = 2000
	maxNumberOfDeploymentsInGraphEnv     = env.RegisterSetting("ROX_MAX_DEPLOYMENTS_NETWORK_GRAPH", env.WithDefault(strconv.Itoa(defaultMaxNumberOfDeploymentsInGraph)))
)

func maxNumberOfDeploymentsInGraph() (int, error) {
	return strconv.Atoi(maxNumberOfDeploymentsInGraphEnv.Setting())
}
