import { EdgeModel, EdgeTerminalType, Model, NodeModel } from '@patternfly/react-topology';

import { EdgeProperties, ListenPort, OutEdges } from 'types/networkFlow.proto';
import { Override } from 'utils/type.utils';

export type CustomModel = Override<
    Model,
    { nodes: CustomNodeModel[]; edges: CustomEdgeModel[]; updateCount: number }
>;

// Node types

export type CustomNodeModel =
    | NamespaceNodeModel
    | DeploymentNodeModel
    | ExternalGroupNodeModel
    | ExternalEntitiesNodeModel
    | CIDRBlockNodeModel
    | ExtraneousNodeModel;

export type NamespaceNodeModel = Override<NodeModel, { data: NamespaceData }>;

export type DeploymentNodeModel = Override<NodeModel, { data: DeploymentData }>;

export type ExternalGroupNodeModel = Override<NodeModel, { data: ExternalGroupData }>;

export type ExternalEntitiesNodeModel = Override<NodeModel, { data: ExternalEntitiesData }>;

export type CIDRBlockNodeModel = Override<NodeModel, { data: CIDRBlockData }>;

export type ExtraneousNodeModel = Override<NodeModel, { data: ExtraneousData }>;

export type CustomGroupNodeData = NamespaceData | ExternalGroupData;

export type CustomSingleNodeData = DeploymentData | ExternalEntitiesData | CIDRBlockData;

export type CustomNodeData =
    | NamespaceData
    | DeploymentData
    | ExternalGroupData
    | ExternalEntitiesData
    | CIDRBlockData;

export type NamespaceData = {
    type: 'NAMESPACE';
    collapsible: boolean;
    showContextMenu: boolean;
};

export type NetworkPolicyState = 'none' | 'both' | 'ingress' | 'egress';

export type NodeDataType =
    | 'DEPLOYMENT'
    | 'EXTERNAL_GROUP'
    | 'EXTERNAL_ENTITIES'
    | 'CIDR_BLOCK'
    | 'EXTRANEOUS';

export type DeploymentData = {
    type: 'DEPLOYMENT';
    id: string;
    deployment: {
        cluster: string;
        listenPorts: ListenPort[];
        name: string;
        namespace: string;
    };
    policyIds: string[];
    networkPolicyState: NetworkPolicyState;
    showPolicyState: boolean;
    isExternallyConnected: boolean;
    showExternalState: boolean;
};

export type ExternalGroupData = {
    type: 'EXTERNAL_GROUP';
    collapsible: boolean;
    showContextMenu: boolean;
};

export type ExternalEntitiesData = {
    type: 'EXTERNAL_ENTITIES';
    id: string;
    outEdges: OutEdges;
};

export type CIDRBlockData = {
    type: 'CIDR_BLOCK';
    id: string;
    externalSource: {
        cidr?: string;
        default: boolean;
        name: string;
    };
    outEdges: OutEdges;
};

export type ExtraneousData = {
    type: 'EXTRANEOUS';
    collapsible: boolean;
    showContextMenu: boolean;
    numFlows: number;
};

// Edge types

export type CustomEdgeModel = Override<EdgeModel, { data: EdgeData }>;

export type EdgeData = {
    // the edge label shows up when this exists
    tag?: string;
    // this is so that we can easily reference the tag content
    portProtocolLabel: string;
    properties: EdgeProperties[];
    startTerminalType?: EdgeTerminalType;
    endTerminalType?: EdgeTerminalType;
    isBidirectional: boolean;
};
