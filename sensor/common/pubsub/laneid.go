package pubsub

type LaneID int

const (
	DefaultLane LaneID = iota
	KubernetesDispatcherEventLane
	FromCentralResolverEventLane
	UnenrichedProcessIndicatorLane
	EnrichedProcessIndicatorLane
)

var (
	laneToString = map[LaneID]string{
		DefaultLane:                        "Default",
		KubernetesDispatcherEventLane:      "KubernetesDispatcherEvent",
		FromCentralResolverEventLane:       "FromCentralResolverEvent",
		UnenrichedProcessIndicatorLane:     "UnenrichedProcessIndicator",
		EnrichedProcessIndicatorLane:       "EnrichedProcessIndicator",
	}
)

func (l LaneID) String() string {
	if laneStr, ok := laneToString[l]; ok {
		return laneStr
	}
	return "unknown"
}
