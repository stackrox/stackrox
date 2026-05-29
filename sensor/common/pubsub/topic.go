package pubsub

type Topic int

const (
	DefaultTopic Topic = iota
	KubernetesDispatcherEventTopic
	FromCentralResolverEventTopic
	UnenrichedProcessIndicatorTopic
	EnrichedProcessIndicatorTopic
	DetectorProcessIndicatorTopic
	DetectorNetworkFlowTopic
	DetectorFileAccessTopic
	DetectorAuditLogTopic
	DetectorDeploymentTopic
	DetectorScanResultTopic
	DetectorDeployAlertOutputTopic
	ResolvedResourceEventTopic
)

var (
	topicToString = map[Topic]string{
		DefaultTopic:                    "Default",
		KubernetesDispatcherEventTopic:  "KubernetesDispatcherEvent",
		FromCentralResolverEventTopic:   "FromCentralResolverEvent",
		UnenrichedProcessIndicatorTopic: "UnenrichedProcessIndicator",
		EnrichedProcessIndicatorTopic:   "EnrichedProcessIndicator",
		DetectorProcessIndicatorTopic:   "DetectorProcessIndicator",
		DetectorNetworkFlowTopic:        "DetectorNetworkFlow",
		DetectorFileAccessTopic:         "DetectorFileAccess",
		DetectorAuditLogTopic:           "DetectorAuditLog",
		DetectorDeploymentTopic:         "DetectorDeployment",
		DetectorScanResultTopic:         "DetectorScanResult",
		DetectorDeployAlertOutputTopic:  "DetectorDeployAlertOutput",
		ResolvedResourceEventTopic:      "ResolvedResourceEvent",
	}
)

func (t Topic) String() string {
	if topicStr, ok := topicToString[t]; ok {
		return topicStr
	}
	return "unknown"
}
