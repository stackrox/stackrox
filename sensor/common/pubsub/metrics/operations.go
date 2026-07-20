package metrics

// Operation represents a pubsub operation that we want to track.
type Operation int

const (
	Published Operation = iota
	Processed
	PublishError
	ConsumerError
	NoConsumers
	Deduped
	Requeued
)

var (
	operationToString = map[Operation]string{
		Published:     "success",
		Processed:     "success",
		PublishError:  "error",
		ConsumerError: "error",
		NoConsumers:   "no_consumers",
		Deduped:       "deduped",
		Requeued:      "requeued",
	}
)

func (o Operation) String() string {
	if opStr, ok := operationToString[o]; ok {
		return opStr
	}
	return "unknown"
}
