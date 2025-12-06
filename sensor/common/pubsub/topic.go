package pubsub

type Topic int

const (
	DefaultTopic Topic = iota
)

var (
	topicToString = map[Topic]string{
		DefaultTopic: "Default",
	}
)

func (t Topic) String() string {
	if topicStr, ok := topicToString[t]; ok {
		return topicStr
	}
	return "unknown"
}
