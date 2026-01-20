package pubsub

type Topic int

const (
	DefaultTopic Topic = iota
	KubernetesDispatcherEventTopic
	FromCentralResolverEventTopic
	UnenrichedProcessIndicatorTopic
	EnrichedProcessIndicatorTopic
)

var (
	topicToString = map[Topic]string{
		DefaultTopic:                        "Default",
		KubernetesDispatcherEventTopic:      "KubernetesDispatcherEvent",
		FromCentralResolverEventTopic:       "FromCentralResolverEvent",
		UnenrichedProcessIndicatorTopic:     "UnenrichedProcessIndicator",
		EnrichedProcessIndicatorTopic:       "EnrichedProcessIndicator",
	}
)

func (t Topic) String() string {
	if topicStr, ok := topicToString[t]; ok {
		return topicStr
	}
	return "unknown"
}
