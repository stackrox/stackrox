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
	switch t := msg.WhichMsg(); t {
	case central.MsgFromSensor_NetworkFlowUpdate_case:
		return "NetworkFlow"
	case central.MsgFromSensor_ScrapeUpdate_case:
		return "ScrapeUpdate"
	case central.MsgFromSensor_Event_case:
		if msg.GetEvent().GetResource() == nil {
			return "Unknown"
		}
		return strings.TrimPrefix(reflect.TypeOf(msg.GetEvent().GetResource()).Elem().Name(), "SensorEvent_")
	case central.MsgFromSensor_ClusterStatusUpdate_case:
		return "ClusterStatusUpdate"
	case central.MsgFromSensor_NetworkPoliciesResponse_case:
		return "NetworkPoliciesResponse"
	case central.MsgFromSensor_ClusterHealthInfo_case:
		return "ClusterHealthInfo"
	case central.MsgFromSensor_ClusterMetrics_case:
		return "ClusterMetrics"
	case central.MsgFromSensor_AuditLogStatusInfo_case:
		return "AuditLogStatusInfo"
	case central.MsgFromSensor_ProcessListeningOnPortUpdate_case:
		return "ProcessListeningOnPortUpdate"
	case central.MsgFromSensor_ComplianceOperatorInfo_case:
		return "ComplianceOperatorInfo"
	case central.MsgFromSensor_ComplianceResponse_case:
		return "ComplianceResponse"
	case central.MsgFromSensor_DeploymentEnhancementResponse_case:
		return "DeploymentEnhancementResponse"
	default:
		log.Errorf("UNEXPECTED:  Unknown message type: %v", t)
		return "Unknown"
	}
}
