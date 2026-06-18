package pubsub

type ConsumerID int

const (
	NoConsumers ConsumerID = iota
	DefaultConsumer
	ResolverConsumer
	EnrichedProcessConsumer
	UnenrichedProcessConsumer
	FileActivityEnrichedProcessConsumer
	DetectorProcessIndicatorConsumer
)

var (
	consumerToString = map[ConsumerID]string{
		NoConsumers:                         "NoConsumers",
		DefaultConsumer:                     "Default",
		ResolverConsumer:                    "Resolver",
		EnrichedProcessConsumer:             "EnrichedProcess",
		FileActivityEnrichedProcessConsumer: "FileActivityEnrichedProcess",
		UnenrichedProcessConsumer:           "UnenrichedProcess",
		DetectorProcessIndicatorConsumer:    "DetectorProcessIndicator",
	}
)

func (c ConsumerID) String() string {
	if consumerStr, ok := consumerToString[c]; ok {
		return consumerStr
	}
	return "unknown"
}
