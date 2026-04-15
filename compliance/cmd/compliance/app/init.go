package app

import (
	"github.com/stackrox/rox/compliance/collection/file"
	"github.com/stackrox/rox/compliance/collection/kubernetes"
	collectionMetrics "github.com/stackrox/rox/compliance/collection/metrics"
	"github.com/stackrox/rox/compliance/node/index"
	vmRelayMetrics "github.com/stackrox/rox/compliance/virtualmachines/relay/metrics"
)

func initMetrics() {
	vmRelayMetrics.Init()
	collectionMetrics.Init()
}

func initCollectors() {
	index.InitZerolog()
	kubernetes.InitScheme()
	file.InitUserMaps()
}
