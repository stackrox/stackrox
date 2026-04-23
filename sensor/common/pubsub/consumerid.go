package pubsub

type ConsumerID int

const (
	NoConsumers ConsumerID = iota
	DefaultConsumer
	ResolverConsumer
	EnrichedProcessConsumer
	FileActivityEnrichedProcessConsumer
	UnenrichedProcessConsumer
)

var (
	consumerToString = map[ConsumerID]string{
		NoConsumers:                         "NoConsumers",
		DefaultConsumer:                     "Default",
		ResolverConsumer:                    "Resolver",
		EnrichedProcessConsumer:             "EnrichedProcess",
		FileActivityEnrichedProcessConsumer: "FileActivityEnrichedProcess",
		UnenrichedProcessConsumer:           "UnenrichedProcess",
	}
)

func (c ConsumerID) String() string {
	if consumerStr, ok := consumerToString[c]; ok {
		return consumerStr
	}
	return "unknown"
}
