package events

import (
	"github.com/stackrox/rox/sensor/common/pubsub"
)

// ResponseType identifies which kind of Central response the handler
// should compose from a result event.
type ResponseType int

const (
	ApplyScanConfigResponse ResponseType = iota
	DeleteScanConfigResponse
)

// ComplianceCentralResultEvent carries the outcome of a compliance
// scan config operation from the manager back to the handler.
type ComplianceCentralResultEvent struct {
	RequestID string
	Error     error
	Type      ResponseType
}

func (e *ComplianceCentralResultEvent) Topic() pubsub.Topic {
	return pubsub.ComplianceCentralResultTopic
}
func (e *ComplianceCentralResultEvent) Lane() pubsub.LaneID {
	return pubsub.ComplianceCentralResultLane
}
