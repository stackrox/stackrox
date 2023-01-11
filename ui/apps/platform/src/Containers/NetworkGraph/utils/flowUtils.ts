import { Controller } from '@patternfly/react-topology';
import { uniq } from 'lodash';
import { AdvancedFlowsFilterType } from '../common/AdvancedFlowsFilter/types';
import { Flow } from '../types/flow.type';
import { CustomEdgeModel, CustomNodeData } from '../types/topology.type';

const protocolLabel = {
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
        return acc + (curr.children && curr.children.length ? curr.children.length : 1);
    }, 0);
    return numFlows;
}

/*
  This function takes edges and a selected id of a node and creates an array of flows
  which is a structured data type used for showing specific information in the network graph
  side panels
*/
export function getNetworkFlows(
    edges: CustomEdgeModel[],
    controller: Controller,
    id: string
): Flow[] {
    const networkFlows: Flow[] = edges.reduce((acc, edge) => {
        // filter out edges not connected to node with selected id
        if ((edge.source !== id && edge.target !== id) || !edge.source || !edge.target) {
            return acc;
        }
        const adjacentNodeId = edge.source !== id ? edge.source : edge.target;
        const adjacentNode = controller.getNodeById(adjacentNodeId);
        const adjacentNodeData: CustomNodeData = adjacentNode?.getData();
        const result = edge.data.properties.map(({ port, protocol }): Flow => {
            const direction: string = edge.source === id ? 'Egress' : 'Ingress';
            const type = adjacentNodeData.type === 'DEPLOYMENT' ? 'Deployment' : 'External';
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
            const flowId = `${entity}-${namespace}-${direction}-${port}-${protocol}`;
            return {
                id: flowId,
                type,
                entity,
                namespace,
                direction,
                port: String(port),
                protocol: protocolLabel[protocol],
                // @TODO: Need to set this depending on whether it is in the baseline or not
                isAnomalous: true,
                // @TODO: Need to create nesting structure
                children: [],
            };
        });
        return [...acc, ...result] as Flow[];
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
        let matchedFlowType = true;
        let matchedDirectionality = true;
        const matchedProtocol = true;
        let matchedPort = true;

        // check filtering by entity name
        if (flow.entity.includes(entityNameFilter)) {
            matchedEntityName = true;
        }

        // check filtering by flow type
        if (advancedFilters.flows.length) {
            const isAnomalousFiltered =
                advancedFilters.flows.includes('anomalous') && flow.isAnomalous;
            const isBaselineFiltered =
                advancedFilters.flows.includes('baseline') && !flow.isAnomalous;
            matchedFlowType = isAnomalousFiltered || isBaselineFiltered;
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
                advancedFilters.protocols.includes('TCP') && flow.protocol === 'TCP';
            const isUDPFiltered =
                advancedFilters.protocols.includes('UDP') && flow.protocol === 'UDP';
            matchedDirectionality = isTCPFiltered || isUDPFiltered;
        }

        // check filtering by ports
        if (advancedFilters.ports.length && !advancedFilters.ports.includes(flow.port)) {
            matchedPort = false;
        }

        return (
            matchedEntityName &&
            matchedFlowType &&
            matchedDirectionality &&
            matchedProtocol &&
            matchedPort
        );
    });
    return filteredFlows;
}
