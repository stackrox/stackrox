package common

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
)

const vmIndexACKResourceIDSeparator = ":"

// SendSensorACK sends a SensorACK only when sensor capability support is explicitly advertised.
func SendSensorACK(ctx concurrency.Waitable, action central.SensorACK_Action, messageType central.SensorACK_MessageType, resourceID, reason string, injector MessageInjector) {
	if injector == nil {
		return
	}

	if !injector.HasCapability(centralsensor.SensorACKSupport) {
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

// VMIndexACKResourceID builds the correlation key for VM index ACK/NACK.
//
// The format is always "vmID:vsockCID" with the separator present even when
// one component is empty (e.g. ":100" or "vm-1:"). This makes it unambiguous
// which part is which when debugging logs or parsing the resource ID.
//
// Limitation: this pair cannot distinguish multiple in-flight reports from the
// same VM while it keeps the same CID; a stale ACK may still match the latest
// VMID:CID entry.
func VMIndexACKResourceID(vmID, vsockCID string) string {
	if vmID == "" && vsockCID == "" {
		return ""
	}
	return vmID + vmIndexACKResourceIDSeparator + vsockCID
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
