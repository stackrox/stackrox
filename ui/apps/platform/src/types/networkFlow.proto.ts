// TODO verify if any properties can be optional or have null as value.

export type NetworkFlow = {
    props: NetworkFlowProperties;
    lastSeenTimestamp: string; // ISO 8601 date string
};

export type NetworkFlowProperties = {
    srcEntity: NetworkEntityInfo;
    dstEntity: NetworkEntityInfo;
    dstPort: number; // uint32 may be 0 if not applicable (e.g., icmp)
    l4protocol: L4Protocol;
};

export type NetworkEndpoint = {
    props: NetworkEndpointProperties;
    lastActiveTimestamp: string; // ISO 8601 date string
};

export type NetworkEndpointProperties = {
    entity: NetworkEntityInfo;
    port: number; // uint32
    l4protocol: L4Protocol;
};

export type NetworkEntity = {
    info: NetworkEntityInfo;
    scope: NetworkEntityScope | null;
};

/*
 * Represents known cluster network peers to which the flows must be scoped.
 * In future, to restrict flows to more granular entities, such as deployment,
 * scope could include deployment ID.
 * Note: The highest scope level is cluster.
 */
export type NetworkEntityScope = {
    clusterId: string;
};

export type NetworkEntityInfo =
    | DeploymentNetworkEntityInfo
    | ExternalSourceNetworkEntityInfo
    | InternetNetworkEntityInfo
    | InternalNetworkEntitiesInfo;

export type DeploymentDetails = {
    name: string;
    namespace: string;
    cluster: string; // deprecated
    listenPorts: ListenPort[];
};

export type DeploymentNetworkEntityInfo = {
    deployment: DeploymentDetails;
    type: 'DEPLOYMENT';
} & BaseNetworkEntityInfo;

export type ListenPort = {
    port: number; // uint32
    l4protocol: L4Protocol;
};

type BaseNetworkFlowProperties = {
    dstPort: number;
    l4protocol: L4Protocol;
};

export type ExternalNetworkFlowProperties = BaseNetworkFlowProperties &
    (
        | {
              srcEntity: DeploymentNetworkEntityInfo;
              dstEntity: ExternalSourceNetworkEntityInfo;
          }
        | {
              srcEntity: ExternalSourceNetworkEntityInfo;
              dstEntity: DeploymentNetworkEntityInfo;
          }
    );

type ExternalNetworkFlow = {
    props: ExternalNetworkFlowProperties;
    lastSeenTimestamp: string; // ISO 8601 date string
    clusterId: string;
};

export type ExternalNetworkFlowsResponse = {
    entity: ExternalSourceNetworkEntityInfo;
    flows: ExternalNetworkFlow[];
    totalFlows: number;
};

export type ExternalNetworkFlowsMetadataResponse = {
    entities: ExternalNetworkFlowsMetadata[];
    totalEntities: number;
};

export type ExternalNetworkFlowsMetadata = {
    entity: ExternalSourceNetworkEntityInfo;
    flowsCount: number;
};

export type ExternalSourceNetworkEntityInfo = {
    externalSource: {
        name: string;
        cidr?: string;
        default: boolean; // `default` indicates whether the external source is user-generated or system-generated.
        discovered: boolean; // `discovered` indicates whether the external source was detected from network traffic.
    };
    type: 'EXTERNAL_SOURCE';
} & BaseNetworkEntityInfo;

export type InternetNetworkEntityInfo = {
    type: 'INTERNET';
} & BaseNetworkEntityInfo;

export type InternalNetworkEntitiesInfo = {
    type: 'INTERNAL_ENTITIES';
} & BaseNetworkEntityInfo;

type BaseNetworkEntityInfo = {
    type: NetworkEntityInfoType;
    id: string;
};

export type NetworkEntityInfoType =
    | 'UNKNOWN_TYPE'
    | 'DEPLOYMENT'
    | 'INTERNET'
    | 'LISTEN_ENDPOINT'
    | 'EXTERNAL_SOURCE'
    | 'INTERNAL_ENTITIES';

export type L4Protocol =
    | 'L4_PROTOCOL_UNKNOWN'
    | 'L4_PROTOCOL_TCP'
    | 'L4_PROTOCOL_UDP'
    | 'L4_PROTOCOL_ICMP'
    | 'L4_PROTOCOL_RAW'
    | 'L4_PROTOCOL_SCTP'
    | 'L4_PROTOCOL_ANY'; // -1

// network flow graph nodes
export type Node = {
    entity: NetworkEntityInfo;
    internetAccess: boolean;
    policyIds: string[];
    nonIsolatedIngress: boolean;
    nonIsolatedEgress: boolean;
    queryMatch: boolean;
    outEdges: OutEdges;
};

export type OutEdges = Record<
    string,
    {
        properties: EdgeProperties[];
    }
>;

export type EdgeProperties = {
    port: number;
    protocol: L4Protocol;
    lastActiveTimestamp: string | null;
};
