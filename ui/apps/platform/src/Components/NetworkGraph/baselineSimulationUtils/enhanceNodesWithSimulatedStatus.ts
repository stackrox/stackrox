import remove from 'lodash/remove';
import {
    InternetEntity,
    DeploymentEntity,
    ExternalSourceEntity,
    Protocol,
    NetworkNode,
    SimulatedBaseline,
    SimulatedBaselineStatus,
    Entity,
} from 'Containers/Network/networkTypes';
import { getIsExternal } from 'utils/networkGraphUtils';

interface Property {
    port: string;
    protocol: Protocol;
    timestamp?: string;
}

interface PropertyWithSimulatedStatus extends Property {
    simulatedStatus: SimulatedBaselineStatus;
}

type OutEdges = Record<string, { properties: Property[] }>;

type OutEdgesWithSimulatedStatus = Record<string, { properties: PropertyWithSimulatedStatus[] }>;

export interface NetworkNodeBase {
    entity: DeploymentEntity | InternetEntity | ExternalSourceEntity;
    externallyConnected: boolean;
    internetAccess: boolean;
    nonIsolatedEgress: boolean;
    nonIsolatedIngress: boolean;
    outEdges: OutEdges;
    policyIds: string[];
    queryMatch: boolean;
}

export interface NetworkNodeBaseWithSimulatedStatus extends Omit<NetworkNodeBase, 'outEdges'> {
    outEdges: OutEdgesWithSimulatedStatus;
}

function getEntity(entity: Entity) {
    if (entity.type === 'DEPLOYMENT') {
        return {
            id: entity.id,
            type: entity.type,
            deployment: {
                name: entity.name,
                namespace: entity.namespace,
            },
        };
    }
    if (entity.type === 'EXTERNAL_SOURCE') {
        return {
            id: entity.id,
            type: entity.type,
            externalSource: {
                name: entity.name,
                cidr: entity.namespace,
            },
        };
    }
    return {
        id: entity.id,
        type: entity.type,
    };
}

function mergeProperties(
    prevProperties: Property[] = [],
    newProperty: PropertyWithSimulatedStatus
): (Property | PropertyWithSimulatedStatus)[] {
    const properties = remove(
        prevProperties,
        (property: Property) =>
            property.port !== newProperty.port || property.protocol !== newProperty.protocol
    );
    const result: Property[] = [...properties, newProperty];
    return result;
}

function enhanceNodesWithSimulatedStatus(
    selectedNode: NetworkNode,
    nodes: NetworkNodeBase[],
    simulatedBaselines: SimulatedBaseline[]
): (NetworkNodeBase | NetworkNodeBaseWithSimulatedStatus)[] {
    // if node doesn't exist create it
    const nodeMap = nodes.reduce((acc, curr) => {
        acc[curr.entity.id] = curr;
        return acc;
    }, {} as Record<string, NetworkNodeBase>);
    const modifiedNodeMap = simulatedBaselines.reduce(
        (acc, curr): Record<string, NetworkNodeBase> => {
            const { id } = curr.peer.entity;
            // if node doesn't exist
            if (!acc[id]) {
                const entity = getEntity(curr.peer.entity);
                const result = {
                    entity,
                    externallyConnected:
                        getIsExternal(selectedNode.type) || getIsExternal(curr.peer.entity.type),
                    internetAccess: false,
                    nonIsolatedEgress: false,
                    nonIsolatedIngress: false,
                    policyIds: [],
                    queryMatch: false,
                    outEdges: {},
                };
                acc[id] = result;
            }
            const node = curr.peer.ingress ? acc[selectedNode.id] : acc[id];
            const outEdgeId = curr.peer.ingress ? id : selectedNode.id;
            const newProperty = {
                port: curr.peer.port,
                protocol: curr.peer.protocol,
                simulatedStatus: curr.simulatedStatus,
            };
            const mergedProperties = mergeProperties(
                node.outEdges[outEdgeId]?.properties,
                newProperty
            );
            node.outEdges[outEdgeId] = { properties: mergedProperties };
            return acc;
        },
        nodeMap as Record<string, NetworkNodeBase | NetworkNodeBaseWithSimulatedStatus>
    );

    return Object.values(modifiedNodeMap);
}

export default enhanceNodesWithSimulatedStatus;
