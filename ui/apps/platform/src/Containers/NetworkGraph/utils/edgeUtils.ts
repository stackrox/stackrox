import { EdgeProperties, L4Protocol } from 'types/networkFlow.proto';
import { EdgeTerminalType } from '@patternfly/react-topology';
import { CustomEdgeModel, EdgeData } from '../types/topology.type';
import { protocolLabel } from './flowUtils';

function getPortProtocolLabel(port: number, protocol: L4Protocol): string {
    return `${port} ${protocolLabel[protocol]}`;
}

export function getPortProtocolEdgeLabel(properties: EdgeProperties[]): string {
    const { port, protocol } = properties[0];
    const singlePortLabel = getPortProtocolLabel(port, protocol);
    return `${properties.length === 1 ? singlePortLabel : properties.length}`;
}

function filterDNSEdges(properties) {
    return !(
        (properties.port === 53 || properties.port === 5353) &&
        properties.protocol === 'L4_PROTOCOL_UDP'
    );
}

export function removeDNSEdges(edges: CustomEdgeModel[]): CustomEdgeModel[] {
    const modifiedEdges: CustomEdgeModel[] = [];
    edges.forEach((edge) => {
        const filteredSourceToTargetProperties =
            edge.data.sourceToTargetProperties.filter(filterDNSEdges);
        const filteredTargetToSourceProperties =
            edge.data?.targetToSourceProperties?.filter(filterDNSEdges) || [];
        const combinedProperties = [
            ...filteredSourceToTargetProperties,
            ...filteredTargetToSourceProperties,
        ];

        if (combinedProperties.length !== 0) {
            const portProtocolLabel = getPortProtocolEdgeLabel([
                ...filteredSourceToTargetProperties,
                ...filteredTargetToSourceProperties,
            ]);
            const modifiedData: EdgeData = {
                sourceToTargetProperties: filteredSourceToTargetProperties,
                targetToSourceProperties: filteredTargetToSourceProperties,
                isBidirectional:
                    filteredTargetToSourceProperties.length !== 0 &&
                    filteredSourceToTargetProperties.length !== 0,
                portProtocolLabel,
                tag: portProtocolLabel,
            };
            if (filteredSourceToTargetProperties.length !== 0) {
                modifiedData.endTerminalType = 'directional' as EdgeTerminalType;
            } else {
                modifiedData.endTerminalType = 'none' as EdgeTerminalType;
            }
            if (filteredTargetToSourceProperties.length !== 0) {
                modifiedData.startTerminalType = 'directional' as EdgeTerminalType;
            } else {
                modifiedData.startTerminalType = 'none' as EdgeTerminalType;
            }
            const modifiedEdge = {
                ...edge,
                data: modifiedData,
            };

            modifiedEdges.push(modifiedEdge);
        }
    });
    return modifiedEdges;
}
