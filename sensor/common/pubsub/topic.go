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
	}
)

func (t Topic) String() string {
	if topicStr, ok := topicToString[t]; ok {
		return topicStr
	}
	return "unknown"
}
