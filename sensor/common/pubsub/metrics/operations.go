package metrics

// Operation represents a pubsub operation that we want to track.
type Operation int

const (
	Published Operation = iota
	Processed
	PublishError
	ConsumerError
	NoConsumers
)

var (
	operationToString = map[Operation]string{
		Published:     "published",
		Processed:     "processed",
		PublishError:  "publish_error",
		ConsumerError: "consumer_error",
		NoConsumers:   "no_consumers",
	}
)

func (o Operation) String() string {
	if opStr, ok := operationToString[o]; ok {
		return opStr
	}
	return "unknown"
}
