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
	OutputQueueConsumer
	NetworkFlowManagerResourceSyncConsumer
	SensorSoftRestartConsumer
	ComplianceCommandHandlerSensorOnlineConsumer
	ComplianceCommandHandlerSensorOfflineConsumer
	ComplianceAuditLogManagerSensorOnlineConsumer
	ComplianceAuditLogManagerSensorOfflineConsumer
	ComplianceNodeInventoryHandlerSensorOnlineConsumer
	ComplianceNodeInventoryHandlerSensorOfflineConsumer
	ComplianceServiceSensorOnlineConsumer
	ComplianceServiceSensorOfflineConsumer
)

var (
	consumerToString = map[ConsumerID]string{
		NoConsumers:                                         "NoConsumers",
		DefaultConsumer:                                     "Default",
		ResolverConsumer:                                    "Resolver",
		EnrichedProcessConsumer:                             "EnrichedProcess",
		FileActivityEnrichedProcessConsumer:                 "FileActivityEnrichedProcess",
		UnenrichedProcessConsumer:                           "UnenrichedProcess",
		DetectorProcessIndicatorConsumer:                    "DetectorProcessIndicator",
		DetectorNetworkFlowConsumer:                         "DetectorNetworkFlow",
		DetectorFileAccessConsumer:                          "DetectorFileAccess",
		DetectorAuditLogConsumer:                            "DetectorAuditLog",
		DetectorDeploymentConsumer:                          "DetectorDeployment",
		DetectorScanResultConsumer:                          "DetectorScanResult",
		DetectorDeployAlertOutputConsumer:                   "DetectorDeployAlertOutput",
		OutputQueueConsumer:                                 "OutputQueue",
		NetworkFlowManagerResourceSyncConsumer:              "NetworkFlowManagerResourceSync",
		SensorSoftRestartConsumer:                           "SensorSoftRestart",
		ComplianceCommandHandlerSensorOnlineConsumer:        "ComplianceCommandHandlerSensorOnline",
		ComplianceCommandHandlerSensorOfflineConsumer:       "ComplianceCommandHandlerSensorOffline",
		ComplianceAuditLogManagerSensorOnlineConsumer:       "ComplianceAuditLogManagerSensorOnline",
		ComplianceAuditLogManagerSensorOfflineConsumer:      "ComplianceAuditLogManagerSensorOffline",
		ComplianceNodeInventoryHandlerSensorOnlineConsumer:  "ComplianceNodeInventoryHandlerSensorOnline",
		ComplianceNodeInventoryHandlerSensorOfflineConsumer: "ComplianceNodeInventoryHandlerSensorOffline",
		ComplianceServiceSensorOnlineConsumer:               "ComplianceServiceSensorOnline",
		ComplianceServiceSensorOfflineConsumer:              "ComplianceServiceSensorOffline",
	}
)

func (c ConsumerID) String() string {
	if consumerStr, ok := consumerToString[c]; ok {
		return consumerStr
	}
	return "unknown"
}
