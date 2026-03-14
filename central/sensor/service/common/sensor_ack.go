package common

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
)

type capabilityChecker interface {
	HasCapability(centralsensor.SensorCapability) bool
}

// SendSensorACK sends a SensorACK only when sensor capability support is explicitly advertised.
func SendSensorACK(ctx concurrency.Waitable, action central.SensorACK_Action, messageType central.SensorACK_MessageType, resourceID, reason string, injector MessageInjector) {
	if injector == nil {
		return
	}

	checker, ok := injector.(capabilityChecker)
	if !ok || !checker.HasCapability(centralsensor.SensorACKSupport) {
		return
	}

	if err := injector.InjectMessage(ctx, &central.MsgToSensor{
		Msg: &central.MsgToSensor_SensorAck{
			SensorAck: &central.SensorACK{
				Action:      action,
				MessageType: messageType,
				ResourceId:  resourceID,
				Reason:      reason,
			},
		},
	}); err != nil {
		log.Warnf("Failed injecting SensorACK (%v) for %v (resource_id=%s): %v", action, messageType, resourceID, err)
	}
}

// SendLegacyNodeInventoryACK sends the legacy NodeInventoryACK message supported since version 4.1.
func SendLegacyNodeInventoryACK(ctx concurrency.Waitable, clusterID, nodeName string, action central.NodeInventoryACK_Action, messageType central.NodeInventoryACK_MessageType, injector MessageInjector) {
	if injector == nil {
		return
	}

	if err := injector.InjectMessage(ctx, &central.MsgToSensor{
		Msg: &central.MsgToSensor_NodeInventoryAck{
			NodeInventoryAck: &central.NodeInventoryACK{
				ClusterId:   clusterID,
				NodeName:    nodeName,
				Action:      action,
				MessageType: messageType,
			},
		},
	}); err != nil {
		log.Warnf("Failed injecting legacy NodeInventoryACK (%v) for cluster=%s node=%s: %v", messageType, clusterID, nodeName, err)
	}
}
