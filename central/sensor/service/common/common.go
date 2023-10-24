package common

import (
	"reflect"
	"strings"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// GetMessageType returns a string version of a MsgFromSensor
func GetMessageType(msg *central.MsgFromSensor) string {
	switch t := msg.Msg.(type) {
	case *central.MsgFromSensor_NetworkFlowUpdate:
		return "NetworkFlow"
	case *central.MsgFromSensor_ScrapeUpdate:
		return "ScrapeUpdate"
	case *central.MsgFromSensor_Event:
		if msg.GetEvent().GetResource() == nil {
			return "Unknown"
		}
		return strings.TrimPrefix(reflect.TypeOf(msg.GetEvent().GetResource()).Elem().Name(), "SensorEvent_")
	case *central.MsgFromSensor_ClusterStatusUpdate:
		return "ClusterStatusUpdate"
	case *central.MsgFromSensor_NetworkPoliciesResponse:
		return "NetworkPoliciesResponse"
	case *central.MsgFromSensor_ClusterHealthInfo:
		return "ClusterHealthInfo"
	case *central.MsgFromSensor_ClusterMetrics:
		return "ClusterMetrics"
	case *central.MsgFromSensor_AuditLogStatusInfo:
		return "AuditLogStatusInfo"
	case *central.MsgFromSensor_ProcessListeningOnPortUpdate:
		return "ProcessListeningOnPortUpdate"
	case *central.MsgFromSensor_ComplianceOperatorInfo:
		return "ComplianceOperatorInfo"
	case *central.MsgFromSensor_ComplianceResponse:
		return "ComplianceResponse"
	default:
		log.Errorf("UNEXPECTED:  Unknown message type: %T", t)
		return "Unknown"
	}
}
