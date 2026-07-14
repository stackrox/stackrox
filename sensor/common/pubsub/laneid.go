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
	SensorOnlineLane
	SensorOfflineLane
	SyncFinishedLane
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
		SensorOnlineLane:               "SensorOnline",
		SensorOfflineLane:              "SensorOffline",
		SyncFinishedLane:               "SyncFinished",
	}
)

func (l LaneID) String() string {
	if laneStr, ok := laneToString[l]; ok {
		return laneStr
	}
	return "unknown"
}
