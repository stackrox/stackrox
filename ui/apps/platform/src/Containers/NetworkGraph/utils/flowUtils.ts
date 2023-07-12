import uniq from 'lodash/uniq';

import { L4Protocol, NetworkEntityInfoType as EntityType } from 'types/networkFlow.proto';
import { GroupedDiffFlows } from 'types/networkPolicyService';
import { AdvancedFlowsFilterType } from '../common/AdvancedFlowsFilter/types';
import { BaselineSimulationDiffState, Flow, FlowEntityType, Peer } from '../types/flow.type';
import {
    CustomEdgeModel,
    CustomNodeModel,
    CustomSingleNodeData,
    CustomSingleNodeModel,
} from '../types/topology.type';
import { getNodeById } from './networkGraphUtils';

export const protocolLabel = {
    L4_PROTOCOL_UNKNOWN: 'UNKNOWN',
    L4_PROTOCOL_TCP: 'TCP',
    L4_PROTOCOL_UDP: 'UDP',
    L4_PROTOCOL_ICMP: 'ICMP',
    L4_PROTOCOL_RAW: 'RAW',
    L4_PROTOCOL_SCTP: 'SCTP',
    L4_PROTOCOL_ANY: 'ANY',
};

export function getAllUniquePorts(flows: Flow[]): string[] {
    const allPorts = flows.reduce((acc, curr) => {
        if (curr.children && curr.children.length) {
            return [...acc, ...curr.children.map((child) => child.port)];
        }
        return [...acc, curr.port];
    }, [] as string[]);
    const allUniquePorts = uniq(allPorts);
    return allUniquePorts;
}

export function getNumFlows(flows: Flow[]): number {
    const numFlows = flows.reduce((acc, curr) => {
        // if there are no children then it counts as 1 flow
        return acc + (curr.children && curr.children.length > 0 ? curr.children.length : 1);
    }, 0);
    return numFlows;
}

export function createUniqueFlowId({
    entityId,
    direction,
    port,
    protocol,
}: {
    entityId: string;
    direction: string;
    port: string;
    protocol: L4Protocol;
}) {
    return `${entityId}-${direction}-${port}-${protocol}`;
}

export function getUniqueIdFromFlow(flow: Flow) {
    const { entityId, direction, port, protocol } = flow;
    const id = createUniqueFlowId({ entityId, direction, port, protocol });
    return id;
}

export function getUniqueIdFromPeer(peer: Peer) {
    const { id: entityId } = peer.entity;
    const direction = peer.ingress ? 'Ingress' : 'Egress';
    const { port } = peer;
    const { protocol } = peer;
    const id = createUniqueFlowId({ entityId, direction, port: String(port), protocol });
    return id;
}

function createFlow({
    sourceNodeData,
    targetNodeData,
    direction,
    port,
    protocol,
    isSourceNodeSelected,
}: {
    sourceNodeData: CustomSingleNodeData;
    targetNodeData: CustomSingleNodeData;
    direction: string;
    port: number;
    protocol: L4Protocol;
    isSourceNodeSelected: boolean;
}) {
    const adjacentNodeData = isSourceNodeSelected ? targetNodeData : sourceNodeData;
    const { id: entityId, type } = adjacentNodeData;
    let entity = '';
    let namespace = '';
    if (adjacentNodeData.type === 'DEPLOYMENT') {
        entity = adjacentNodeData.deployment.name;
        namespace = adjacentNodeData.deployment.namespace;
    } else if (adjacentNodeData.type === 'CIDR_BLOCK') {
        entity = `${adjacentNodeData.externalSource.name}`;
    } else if (adjacentNodeData.type === 'EXTERNAL_ENTITIES') {
        entity = 'External entities';
    }
    // we need a unique id for each network flow
    const flowId = `${entity}-${namespace}-${direction}-${String(port)}-${String(protocol)}`;
    return {
        id: flowId,
        type,
        entity,
        entityId,
        namespace,
        direction,
        port: String(port),
        protocol,
        // @TODO: Need to set this depending on whether it is in the baseline or not
        isAnomalous: true,
        // @TODO: Need to create nesting structure
        children: [],
    };
}

/*
  This function takes edges and a selected id of a node and creates an array of flows
  which is a structured data type used for showing specific information in the network graph
  side panels
*/
export function getNetworkFlows(
    nodes: CustomNodeModel[],
    edges: CustomEdgeModel[],
    id: string
): Flow[] {
    const networkFlows: Flow[] = edges.reduce((acc, edge) => {
        const isSourceNodeSelected = edge.source === id;

        const sourceNode = getNodeById(nodes, edge.source) as CustomSingleNodeModel;
        const targetNode = getNodeById(nodes, edge.target) as CustomSingleNodeModel;

        const sourceNodeData = sourceNode.data;
        const targetNodeData = targetNode.data;

        const newFlows = edge.data.sourceToTargetProperties.map(({ port, protocol }): Flow => {
            const direction: string = isSourceNodeSelected ? 'Egress' : 'Ingress';
            const flow = createFlow({
                sourceNodeData,
                targetNodeData,
                direction,
                port,
                protocol,
                isSourceNodeSelected,
            });
            return flow;
        });

        const newReverseFlows = edge.data.targetToSourceProperties
            ? edge.data.targetToSourceProperties.map(({ port, protocol }): Flow => {
                  const direction: string = isSourceNodeSelected ? 'Ingress' : 'Egress';
                  const flow = createFlow({
                      sourceNodeData,
                      targetNodeData,
                      direction,
                      port,
                      protocol,
                      isSourceNodeSelected,
                  });
                  return flow;
              })
            : [];

        return [...acc, ...newFlows, ...newReverseFlows] as Flow[];
    }, [] as Flow[]);
    return networkFlows;
}

/*
  This function takes network flows and filters the data based on a text search value 
  for entity name and some advanced filters specific to network flows. These include:
  flow types, directionality, protocols, and ports
*/
export function filterNetworkFlows(
    flows: Flow[],
    entityNameFilter: string,
    advancedFilters: AdvancedFlowsFilterType
): Flow[] {
    const filteredFlows = flows.filter((flow) => {
        let matchedEntityName = false;
        let matchedDirectionality = true;
        const matchedProtocol = true;
        let matchedPort = true;

        // check filtering by entity name
        if (flow.entity.includes(entityNameFilter)) {
            matchedEntityName = true;
        }

        // check filtering by directionality
        if (advancedFilters.directionality.length) {
            const isIngressFiltered =
                advancedFilters.directionality.includes('ingress') && flow.direction === 'Ingress';
            const isEgressFiltered =
                advancedFilters.directionality.includes('egress') && flow.direction === 'Egress';
            matchedDirectionality = isIngressFiltered || isEgressFiltered;
        }

        // check filtering by protocols
        if (advancedFilters.protocols.length) {
            const isTCPFiltered =
                advancedFilters.protocols.includes('L4_PROTOCOL_TCP') &&
                flow.protocol === 'L4_PROTOCOL_TCP';
            const isUDPFiltered =
                advancedFilters.protocols.includes('L4_PROTOCOL_UDP') &&
                flow.protocol === 'L4_PROTOCOL_UDP';
            matchedDirectionality = isTCPFiltered || isUDPFiltered;
        }

        // check filtering by ports
        if (advancedFilters.ports.length && !advancedFilters.ports.includes(flow.port)) {
            matchedPort = false;
        }

        return matchedEntityName && matchedDirectionality && matchedProtocol && matchedPort;
    });
    return filteredFlows;
}

/*
  This function takes network flows and transforms it to peers
*/
export function transformFlowsToPeers(flows: Flow[]): Peer[] {
    return flows.map((flow) => {
        const { entityId, type, entity, namespace, direction, port, protocol } = flow;
        let backendType: EntityType;
        if (type === 'CIDR_BLOCK') {
            backendType = 'EXTERNAL_SOURCE';
        } else if (type === 'EXTERNAL_ENTITIES') {
            backendType = 'INTERNET';
        } else {
            backendType = 'DEPLOYMENT';
        }
        const peer = {
            entity: {
                id: entityId,
                name: entity,
                namespace,
                type: backendType,
            },
            ingress: direction === 'Ingress',
            port: Number(port),
            protocol,
        };
        return peer;
    });
}

export function createFlowsFromGroupedDiffFlows(
    groupedDiffFlow: GroupedDiffFlows,
    baselineSimulationDiffState: BaselineSimulationDiffState
): Flow[] {
    const { entity, properties } = groupedDiffFlow;
    const flows = properties.map(({ ingress, port, protocol }) => {
        const direction = ingress ? 'Ingress' : 'Egress';
        let entityName = '';
        let namespace = '';
        let type: FlowEntityType = 'DEPLOYMENT';
        if (entity.type === 'DEPLOYMENT') {
            entityName = entity.deployment.name;
            namespace = entity.deployment.namespace;
            type = 'DEPLOYMENT';
        } else if (entity.type === 'INTERNET') {
            entityName = 'External entities';
            type = 'EXTERNAL_ENTITIES';
        } else if (entity.type === 'EXTERNAL_SOURCE') {
            entityName = entity.externalSource.name;
            type = 'CIDR_BLOCK';
        }
        const id = createUniqueFlowId({
            entityId: entity.id,
            direction,
            port: String(port),
            protocol,
        });
        const flow: Flow = {
            id,
            type,
            entity: entityName,
            entityId: entity.id,
            namespace,
            direction,
            port: String(port),
            protocol,
            isAnomalous: false,
            children: [],
            baselineSimulationDiffState,
        };
        return flow;
    });
    return flows;
}

export function getNumExtraneousEgressFlows(nodes: CustomNodeModel[]): number {
    const extraneousEgressNode = nodes.find((node) => {
        return node.id === 'extraneous-egress-flows';
    });
    if (!extraneousEgressNode || extraneousEgressNode.visible === false) {
        return 0;
    }
    const numAllowedEgressFlows =
        extraneousEgressNode?.data.type === 'EXTRANEOUS' ? extraneousEgressNode?.data.numFlows : 0;
    return numAllowedEgressFlows;
}

export function getNumExtraneousIngressFlows(nodes: CustomNodeModel[]): number {
    const extraneousIngressNode = nodes.find((node) => {
        return node.id === 'extraneous-ingress-flows';
    });
    if (!extraneousIngressNode || extraneousIngressNode.visible === false) {
        return 0;
    }
    const numAllowedIngressFlows =
        extraneousIngressNode?.data.type === 'EXTRANEOUS'
            ? extraneousIngressNode?.data.numFlows
            : 0;
    return numAllowedIngressFlows;
}
