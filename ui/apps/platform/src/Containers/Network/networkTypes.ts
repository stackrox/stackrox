type EntityType = 'DEPLOYMENT' | 'INTERNET' | 'EXTERNAL_SOURCE';

type Protocol = 'L4_PROTOCOL_TCP' | 'L4_PROTOCOL_UDP' | 'L4_PROTOCOL_ANY';

type ConnectionState = 'active' | 'allowed';

export type BaselineStatus = 'BASELINE' | 'ANOMALOUS';

type Traffic = 'bidirectional' | 'ingress' | 'egress';

export type FlattenedPeer = {
    entity: {
        id: string;
        type: EntityType;
        name: string;
        namespace?: string;
    };
    port: string;
    protocol: Protocol;
    ingress: boolean;
    state: ConnectionState;
};

export type Peer = {
    entity: {
        id: string;
        type: EntityType;
        name: string;
        namespace?: string;
    };
    portsAndProtocols: {
        port: string;
        protocol: Protocol;
        ingress: boolean;
    }[];
    ingress: boolean;
    egress: boolean;
    state: ConnectionState;
};

export type FlattenedNetworkBaseline = {
    peer: FlattenedPeer;
    status: BaselineStatus;
};

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
        isDisallowed?: boolean;
        portsAndProtocols: PortsAndProtocols[];
        traffic: 'bidirectional' | 'ingress' | 'egress';
        type: 'deployment' | 'external';
    };
};
