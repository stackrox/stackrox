package pubsub

type ConsumerID int

const (
	NoConsumers ConsumerID = iota
	DefaultConsumer
	ResolverConsumer
)

var (
	consumerToString = map[ConsumerID]string{
		NoConsumers:      "NoConsumers",
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
