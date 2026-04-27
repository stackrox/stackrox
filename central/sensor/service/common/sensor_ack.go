package common

import (
	"strings"

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
// The key prefers a VMID:CID pair when both are available:
// - VMID avoids cross-VM collisions if a CID is reused by another VM.
// - CID allows Compliance relay/UMH to correlate with CID-keyed retry/cache state.
//
// Limitation: this pair cannot distinguish multiple in-flight reports from the
// same VM while it keeps the same CID; a stale ACK may still match the latest
// VMID:CID entry.
func VMIndexACKResourceID(vmID, vsockCID string) string {
	if vmID == "" {
		return vsockCID
	}
	if vsockCID == "" {
		return vmID
	}
	return strings.Join([]string{vmID, vsockCID}, vmIndexACKResourceIDSeparator)
}

// VMIDFromACKResourceID extracts the VM ID from a composite resource ID
// produced by VMIndexACKResourceID. If the resource ID has no separator,
// it is returned as-is (assumed to be a bare VM ID or CID).
func VMIDFromACKResourceID(resourceID string) string {
	if idx := strings.Index(resourceID, vmIndexACKResourceIDSeparator); idx >= 0 {
		return resourceID[:idx]
	}
	return resourceID
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
