package printer

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/protocompat"
)

// GenerateNetworkFlowViolation constructs violation message for network flow violations.
// Note that network flow violation messages are NOT generated via the usual path, which is
// to write a printer and embed that in printer.go
func GenerateNetworkFlowViolation(networkFlow *augmentedobjs.NetworkFlowDetails) (*storage.Alert_Violation, error) {
	var messageBuilder strings.Builder
	var err error
	if networkFlow.NotInNetworkBaseline {
		_, err = messageBuilder.WriteString("Unexpected")
	} else {
		_, err = messageBuilder.WriteString("Expected")
	}
	if err != nil {
		return nil, err
	}

	_, err = messageBuilder.WriteString(
		fmt.Sprintf(
			" network flow found in deployment. Source name: '%s'. Destination name: '%s'. Destination port: '%s'. Protocol: '%s'.",
			networkFlow.SrcEntityName,
			networkFlow.DstEntityName,
			fmt.Sprint(networkFlow.DstPort),
			networkFlow.L4Protocol.String()))
	if err != nil {
		return nil, err
	}

	return &storage.Alert_Violation{
		Message: messageBuilder.String(),
		MessageAttributes: &storage.Alert_Violation_NetworkFlowInfo_{
			NetworkFlowInfo: &storage.Alert_Violation_NetworkFlowInfo{
				Source: &storage.Alert_Violation_NetworkFlowInfo_Entity{
					Name:                networkFlow.SrcEntityName,
					EntityType:          networkFlow.SrcEntityType,
					DeploymentNamespace: networkFlow.SrcDeploymentNamespace,
					DeploymentType:      networkFlow.SrcDeploymentType,
				},
				Destination: &storage.Alert_Violation_NetworkFlowInfo_Entity{
					Name:                networkFlow.DstEntityName,
					EntityType:          networkFlow.DstEntityType,
					DeploymentNamespace: networkFlow.DstDeploymentNamespace,
					DeploymentType:      networkFlow.DstDeploymentType,
					Port:                int32(networkFlow.DstPort),
				},
				Protocol: networkFlow.L4Protocol,
			},
		},
		Type: storage.Alert_Violation_NETWORK_FLOW,
		Time: protocompat.ConvertTimeToTimestampOrNil(&networkFlow.LastSeenTimestamp),
	}, nil
}
