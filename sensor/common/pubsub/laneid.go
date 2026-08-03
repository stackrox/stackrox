package pubsub

type LaneID int

const (
	DefaultLane LaneID = iota
	KubernetesDispatcherEventLane
	FromCentralResolverEventLane
	UnenrichedProcessIndicatorLane
	EnrichedProcessIndicatorLane
	DetectorProcessIndicatorLane
	DetectorNetworkFlowLane
	DetectorFileAccessLane
	DetectorAuditLogLane
	DetectorDeploymentLane
	DetectorScanResultLane
	DetectorDeployAlertOutputLane
	ResolvedResourceEventLane
	SoftRestartLane
	ResourceSyncFinishedLane
	CentralReachableLane
	SensorOfflineLane
	HandshakeSyncFinishedLane
)

var (
	laneToString = map[LaneID]string{
		DefaultLane:                    "Default",
		KubernetesDispatcherEventLane:  "KubernetesDispatcherEvent",
		FromCentralResolverEventLane:   "FromCentralResolverEvent",
		UnenrichedProcessIndicatorLane: "UnenrichedProcessIndicator",
		EnrichedProcessIndicatorLane:   "EnrichedProcessIndicator",
		DetectorProcessIndicatorLane:   "DetectorProcessIndicator",
		DetectorNetworkFlowLane:        "DetectorNetworkFlow",
		DetectorFileAccessLane:         "DetectorFileAccess",
		DetectorAuditLogLane:           "DetectorAuditLog",
		DetectorDeploymentLane:         "DetectorDeployment",
		DetectorScanResultLane:         "DetectorScanResult",
		DetectorDeployAlertOutputLane:  "DetectorDeployAlertOutput",
		ResolvedResourceEventLane:      "ResolvedResourceEvent",
		SoftRestartLane:                "SoftRestart",
		ResourceSyncFinishedLane:       "ResourceSyncFinished",
		CentralReachableLane:           "CentralReachable",
		SensorOfflineLane:              "SensorOffline",
		HandshakeSyncFinishedLane:      "HandshakeSyncFinished",
	}
)

func (l LaneID) String() string {
	if laneStr, ok := laneToString[l]; ok {
		return laneStr
	}
	return "unknown"
}
