package pubsub

type LaneID int

const (
	DefaultLane LaneID = iota
)

var (
	laneToString = map[LaneID]string{
		DefaultLane: "Default",
	}
)

func (l LaneID) String() string {
	if laneStr, ok := laneToString[l]; ok {
		return laneStr
	}
	return "unknown"
}
