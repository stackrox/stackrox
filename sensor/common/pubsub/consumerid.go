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
	CertRefreshSecuredClusterTLSIssuerSensorOnlineConsumer
	CertRefreshSecuredClusterTLSIssuerSensorOfflineConsumer
	CentralProxyHandlerSensorOnlineConsumer
	CentralProxyHandlerSensorOfflineConsumer
	ImageServiceSensorOnlineConsumer
	ImageServiceSensorOfflineConsumer
	VirtualMachineIndexHandlerSensorOnlineConsumer
	VirtualMachineIndexHandlerSensorOfflineConsumer
	EventPipelineSensorOnlineConsumer
	EventPipelineSensorOfflineConsumer
)

var (
	consumerToString = map[ConsumerID]string{
		NoConsumers:                                             "NoConsumers",
		DefaultConsumer:                                         "Default",
		ResolverConsumer:                                        "Resolver",
		EnrichedProcessConsumer:                                 "EnrichedProcess",
		FileActivityEnrichedProcessConsumer:                     "FileActivityEnrichedProcess",
		UnenrichedProcessConsumer:                               "UnenrichedProcess",
		DetectorProcessIndicatorConsumer:                        "DetectorProcessIndicator",
		DetectorNetworkFlowConsumer:                             "DetectorNetworkFlow",
		DetectorFileAccessConsumer:                              "DetectorFileAccess",
		DetectorAuditLogConsumer:                                "DetectorAuditLog",
		DetectorDeploymentConsumer:                              "DetectorDeployment",
		DetectorScanResultConsumer:                              "DetectorScanResult",
		DetectorDeployAlertOutputConsumer:                       "DetectorDeployAlertOutput",
		OutputQueueConsumer:                                     "OutputQueue",
		NetworkFlowManagerResourceSyncConsumer:                  "NetworkFlowManagerResourceSync",
		SensorSoftRestartConsumer:                               "SensorSoftRestart",
		CertRefreshSecuredClusterTLSIssuerSensorOnlineConsumer:  "CertRefreshSecuredClusterTLSIssuerSensorOnline",
		CertRefreshSecuredClusterTLSIssuerSensorOfflineConsumer: "CertRefreshSecuredClusterTLSIssuerSensorOffline",
		CentralProxyHandlerSensorOnlineConsumer:                 "CentralProxyHandlerSensorOnline",
		CentralProxyHandlerSensorOfflineConsumer:                "CentralProxyHandlerSensorOffline",
		ImageServiceSensorOnlineConsumer:                        "ImageServiceSensorOnline",
		ImageServiceSensorOfflineConsumer:                       "ImageServiceSensorOffline",
		VirtualMachineIndexHandlerSensorOnlineConsumer:          "VirtualMachineIndexHandlerSensorOnline",
		VirtualMachineIndexHandlerSensorOfflineConsumer:         "VirtualMachineIndexHandlerSensorOffline",
		EventPipelineSensorOnlineConsumer:                       "EventPipelineSensorOnline",
		EventPipelineSensorOfflineConsumer:                      "EventPipelineSensorOffline",
	}
)

func (c ConsumerID) String() string {
	if consumerStr, ok := consumerToString[c]; ok {
		return consumerStr
	}
	return "unknown"
}
