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
	ClusterStatusUpdaterSensorOnlineConsumer
	ClusterStatusUpdaterSensorOfflineConsumer
	ClusterHealthUpdaterSensorOnlineConsumer
	ClusterHealthUpdaterSensorOfflineConsumer
	ClusterMetricsSensorOnlineConsumer
	ClusterMetricsSensorOfflineConsumer
	TelemetryCommandHandlerSensorOnlineConsumer
	TelemetryCommandHandlerSensorOfflineConsumer
)

var (
	consumerToString = map[ConsumerID]string{
		NoConsumers:                                  "NoConsumers",
		DefaultConsumer:                              "Default",
		ResolverConsumer:                             "Resolver",
		EnrichedProcessConsumer:                      "EnrichedProcess",
		FileActivityEnrichedProcessConsumer:          "FileActivityEnrichedProcess",
		UnenrichedProcessConsumer:                    "UnenrichedProcess",
		DetectorProcessIndicatorConsumer:             "DetectorProcessIndicator",
		DetectorNetworkFlowConsumer:                  "DetectorNetworkFlow",
		DetectorFileAccessConsumer:                   "DetectorFileAccess",
		DetectorAuditLogConsumer:                     "DetectorAuditLog",
		DetectorDeploymentConsumer:                   "DetectorDeployment",
		DetectorScanResultConsumer:                   "DetectorScanResult",
		DetectorDeployAlertOutputConsumer:            "DetectorDeployAlertOutput",
		OutputQueueConsumer:                          "OutputQueue",
		NetworkFlowManagerResourceSyncConsumer:       "NetworkFlowManagerResourceSync",
		SensorSoftRestartConsumer:                    "SensorSoftRestart",
		ClusterStatusUpdaterSensorOnlineConsumer:     "ClusterStatusUpdaterSensorOnline",
		ClusterStatusUpdaterSensorOfflineConsumer:    "ClusterStatusUpdaterSensorOffline",
		ClusterHealthUpdaterSensorOnlineConsumer:     "ClusterHealthUpdaterSensorOnline",
		ClusterHealthUpdaterSensorOfflineConsumer:    "ClusterHealthUpdaterSensorOffline",
		ClusterMetricsSensorOnlineConsumer:           "ClusterMetricsSensorOnline",
		ClusterMetricsSensorOfflineConsumer:          "ClusterMetricsSensorOffline",
		TelemetryCommandHandlerSensorOnlineConsumer:  "TelemetryCommandHandlerSensorOnline",
		TelemetryCommandHandlerSensorOfflineConsumer: "TelemetryCommandHandlerSensorOffline",
	}
)

func (c ConsumerID) String() string {
	if consumerStr, ok := consumerToString[c]; ok {
		return consumerStr
	}
	return "unknown"
}
