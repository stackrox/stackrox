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
	DetectorNetworkFlowConsumer
	DetectorFileAccessConsumer
	DetectorAuditLogConsumer
	DetectorDeploymentConsumer
	DetectorScanResultConsumer
	DetectorDeployAlertOutputConsumer
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
		DetectorNetworkFlowConsumer:         "DetectorNetworkFlow",
		DetectorFileAccessConsumer:          "DetectorFileAccess",
		DetectorAuditLogConsumer:            "DetectorAuditLog",
		DetectorDeploymentConsumer:          "DetectorDeployment",
		DetectorScanResultConsumer:          "DetectorScanResult",
		DetectorDeployAlertOutputConsumer:   "DetectorDeployAlertOutput",
	}
)

func (c ConsumerID) String() string {
	if consumerStr, ok := consumerToString[c]; ok {
		return consumerStr
	}
	return "unknown"
}
