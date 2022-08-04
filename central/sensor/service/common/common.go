package common

import (
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/utils"
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
	default:
		utils.Should(errors.Errorf("Unknown message type: %T", t))
		return "Unknown"
	}
}
