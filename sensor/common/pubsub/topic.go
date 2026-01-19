package pubsub

type Topic int

const (
	DefaultTopic Topic = iota
	KubernetesDispatcherEventTopic
	FromCentralResolverEventTopic
	EnrichedProcessIndicatorTopic
)

var (
	topicToString = map[Topic]string{
		DefaultTopic:                      "Default",
		KubernetesDispatcherEventTopic:    "KubernetesDispatcherEvent",
		FromCentralResolverEventTopic:     "FromCentralResolverEvent",
		EnrichedProcessIndicatorTopic:     "EnrichedProcessIndicator",
	}
)

func (t Topic) String() string {
	if topicStr, ok := topicToString[t]; ok {
		return topicStr
	}
	return "unknown"
}
