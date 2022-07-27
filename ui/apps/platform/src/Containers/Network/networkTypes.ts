export type EntityType = 'DEPLOYMENT' | 'INTERNET' | 'EXTERNAL_SOURCE';

export type Protocol = 'L4_PROTOCOL_TCP' | 'L4_PROTOCOL_UDP' | 'L4_PROTOCOL_ANY';

export type ConnectionState = 'active' | 'allowed';

export type BaselineStatus = 'BASELINE' | 'ANOMALOUS' | 'BLOCKED';

type Traffic = 'bidirectional' | 'ingress' | 'egress';

export type Entity = {
    id: string;
    type: EntityType;
    name: string;
    namespace: string;
};

export type FlattenedPeer = {
    entity: Entity;
    port: string;
    protocol: Protocol;
    ingress: boolean;
    state?: ConnectionState;
};

export type Peer = {
    entity: Entity;
    portsAndProtocols: {
        port: string;
        protocol: Protocol;
        ingress: boolean;
    }[];
    ingress: boolean;
    egress: boolean;
    state?: ConnectionState;
};

export type FlattenedNetworkBaseline = {
    peer: FlattenedPeer;
    status: BaselineStatus;
};

export type FlattenedBlockedFlows = FlattenedNetworkBaseline;

export type NetworkBaseline = {
    peer: Peer;
    status: BaselineStatus;
};

export type NetworkFlow = {
    deploymentId: string;
    namespace: string;
    entityType: EntityType;
    entityName: string;
    portsAndProtocols: {
        port: string;
        protocol: Protocol;
    }[];
    connection: ConnectionState;
    traffic: Traffic;
};

export type PortsAndProtocols = {
    lastActiveTimestamp?: string;
    port: number;
    protocol: string;
    traffic: 'bidirectional' | 'ingress' | 'egress';
};

export type Edge = {
    classes?: string;
    data: {
        source?: string;
        target?: string;
        destNodeId: string;
        destNodeNamespace: string;
        destNodeName: string;
        destNodeType?: string;
        sourceNodeId?: string;
        sourceNodeName?: string;
        sourceNodeNamespace?: string;
        targetNodeId?: string;
        targetNodeName?: string;
        targetNodeNamespace?: string;
        isActive: boolean;
        isAllowed: boolean;
        portsAndProtocols: PortsAndProtocols[];
        traffic: 'bidirectional' | 'ingress' | 'egress';
        type: 'deployment' | 'external';
    };
};

type AllFilterState = 0;
type AllowedFilterState = 1;
type ActiveFilterState = 2;

export type FilterState = AllFilterState | AllowedFilterState | ActiveFilterState;

export type NetworkNode = {
    cluster: string;
    deploymentId: string;
    edges: Edge[];
    egress: string[];
    externallyConnected: boolean;
    id: string;
    ingress: string[];
    internetAccess: boolean;
    isActive: boolean;
    listenPorts: {
        port: number;
        l4protocol: string;
    }[];
    name: string;
    nonIsolatedEgress: boolean;
    nonIsolatedIngress: boolean;
    outEdges: {
        [key: string]: {
            properties: {
                port: number;
                protocol: string;
                lastActiveTimestamp: string;
            }[];
        };
    };
    parent: string;
    policyIds: string[];
    queryMatch: boolean;
    type: string;
};

export type NetworkPolicyModification = {
    applyYaml: string;
    toDelete: {
        namespace: string;
        name: string;
    }[];
};

export type SimulatedBaselineStatus = 'ADDED' | 'REMOVED' | 'UNMODIFIED';

export type Properties = {
    port: string;
    protocol: Protocol;
    ingress: boolean;
};

export type SimulatedBaseline = {
    peer: {
        entity: Entity;
        port: string;
        protocol: Protocol;
        ingress: boolean;
        state: ConnectionState;
    };
    simulatedStatus: 'ADDED' | 'REMOVED' | 'UNMODIFIED';
};

export type DeploymentEntity = {
    id: string;
    type: 'DEPLOYMENT';
    deployment: {
        name: string;
        namespace: string;
    };
};

export type ExternalSourceEntity = {
    id: string;
    type: 'EXTERNAL_SOURCE';
    externalSource: {
        name: string;
        cidr: string;
    };
};

export type InternetEntity = {
    id: string;
    type: 'INTERNET';
};

export type AddedRemovedBaselineResponse = {
    entity: DeploymentEntity | ExternalSourceEntity | InternetEntity;
    properties: [Properties];
};

export type ReconciledBaselineResponse = {
    entity: DeploymentEntity | ExternalSourceEntity | InternetEntity;
    added: [Properties];
    removed: [Properties];
    unchanged: [Properties];
};

export type BaselineComparisonsResponse = {
    added: AddedRemovedBaselineResponse[];
    removed: AddedRemovedBaselineResponse[];
    reconciled: ReconciledBaselineResponse[];
};
