package pubsub

type ConsumerID int

const (
	UnknownConsumer ConsumerID = iota
	DefaultConsumer
	ResolverConsumer
)

var (
	consumerToString = map[ConsumerID]string{
		UnknownConsumer:  "Unknown",
		DefaultConsumer:  "Default",
		ResolverConsumer: "Resolver",
	}
)

func (c ConsumerID) String() string {
	if consumerStr, ok := consumerToString[c]; ok {
		return consumerStr
	}
	return "unknown"
}
