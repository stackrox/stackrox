import { EdgeModel, EdgeTerminalType, Model, NodeModel } from '@patternfly/react-topology';

import { DeploymentDetails, EdgeProperties, OutEdges } from 'types/networkFlow.proto';
import { Override } from 'utils/type.utils';

export type CustomModel = Override<Model, { nodes: CustomNodeModel[]; edges: CustomEdgeModel[] }>;

// Node types

export type CustomNodeModel =
    | NamespaceNodeModel
    | DeploymentNodeModel
    | ExternalGroupNodeModel
    | ExternalEntitiesNodeModel
    | InternalEntitiesNodeModel
    | CIDRBlockNodeModel
    | ExtraneousNodeModel;

export type CustomSingleNodeModel =
    | DeploymentNodeModel
    | ExternalEntitiesNodeModel
    | InternalEntitiesNodeModel
    | CIDRBlockNodeModel;

export type NamespaceNodeModel = Override<NodeModel, { data: NamespaceData }>;

export type DeploymentNodeModel = Override<NodeModel, { data: DeploymentData }>;

export type ExternalGroupNodeModel = Override<NodeModel, { data: ExternalGroupData }>;

export type ExternalEntitiesNodeModel = Override<NodeModel, { data: ExternalEntitiesData }>;

export type InternalEntitiesNodeModel = Override<NodeModel, { data: InternalEntitiesData }>;

export type CIDRBlockNodeModel = Override<NodeModel, { data: CIDRBlockData }>;

export type ExtraneousNodeModel = Override<NodeModel, { data: ExtraneousData }>;

export type CustomGroupNodeData = NamespaceData | ExternalGroupData;

export type CustomSingleNodeData =
    | DeploymentData
    | ExternalEntitiesData
    | CIDRBlockData
    | InternalEntitiesData;

export type CustomNodeData =
    | NamespaceData
    | DeploymentData
    | ExternalGroupData
    | ExternalEntitiesData
    | InternalEntitiesData
    | CIDRBlockData;

export type BadgeData = {
    badge?: string;
    badgeColor?: string;
    badgeTextColor?: string;
    badgeBorderColor?: string;
};

export type NamespaceData = {
    type: 'NAMESPACE';
    collapsible: boolean;
    showContextMenu: boolean;
    namespace: string;
    cluster: string;
    isFilteredNamespace: boolean;
    labelIconClass?: string;
    isFadedOut: boolean;
} & BadgeData;

export type NetworkPolicyState = 'none' | 'both' | 'ingress' | 'egress';

// prettier-ignore
type NodeModelType<DataType extends NodeDataType> =
    DataType extends 'DEPLOYMENT' ? DeploymentNodeModel :
    DataType extends 'EXTERNAL_GROUP' ? ExternalGroupNodeModel :
    DataType extends 'EXTERNAL_ENTITIES' ? ExternalEntitiesNodeModel :
    DataType extends 'CIDR_BLOCK' ? CIDRBlockNodeModel :
    DataType extends 'INTERNAL_ENTITIES' ? InternalEntitiesNodeModel :
    DataType extends 'EXTRANEOUS' ? ExtraneousNodeModel :
    never;

/**
 * Returns a type guard for checking if a node is of a certain type
 *
 * @template DataType the type of node to check, should not be specified explicitly
 * @param type the expected string value of the node's `data.type` property
 * @returns a type guard for checking if a node is of a certain type
 */
export function isOfType<DataType extends NodeDataType>(
    type: DataType
): (node: CustomNodeModel) => node is NodeModelType<DataType> {
    return (node: CustomNodeModel): node is NodeModelType<DataType> => node.data.type === type;
}

/**
 * Type guard for checking if a node is of a certain type
 *
 * @template DataType the type of node to check, should not be specified explicitly
 * @param type the expected string value of the node's `data.type` property
 * @param node the node to check
 * @returns true if the node is of the specified type, false otherwise
 */
export function isNodeOfType<DataType extends NodeDataType>(
    type: DataType,
    node: CustomNodeModel
): node is NodeModelType<DataType> {
    return isOfType(type)(node);
}

export type NodeDataType =
    | 'DEPLOYMENT'
    | 'EXTERNAL_GROUP'
    | 'EXTERNAL_ENTITIES'
    | 'CIDR_BLOCK'
    | 'INTERNAL_ENTITIES'
    | 'EXTRANEOUS';

export type DeploymentData = {
    type: 'DEPLOYMENT';
    id: string;
    deployment: DeploymentDetails;
    policyIds: string[];
    networkPolicyState: NetworkPolicyState;
    showPolicyState: boolean;
    isExternallyConnected: boolean;
    showExternalState: boolean;
    isFadedOut: boolean;
    labelIconClass?: string;
} & BadgeData;

export type ExternalGroupData = {
    type: 'EXTERNAL_GROUP';
    collapsible: boolean;
    showContextMenu: boolean;
    isFadedOut: boolean;
};

export type InternalEntitiesData = {
    type: 'INTERNAL_ENTITIES';
    id: string;
    outEdges: OutEdges;
    isFadedOut: boolean;
};

export type ExternalEntitiesData = {
    type: 'EXTERNAL_ENTITIES';
    id: string;
    outEdges: OutEdges;
    isFadedOut: boolean;
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
    isFadedOut: boolean;
} & BadgeData;

export type ExtraneousData = {
    type: 'EXTRANEOUS';
    collapsible: boolean;
    showContextMenu: boolean;
    numFlows: number;
};

// Edge types

export type CustomEdgeModel = Override<
    EdgeModel,
    { source: string; target: string; data: EdgeData }
>;

export type EdgeData = {
    // the edge label shows up when this exists
    tag?: string;
    // this is so that we can easily reference the tag content
    portProtocolLabel: string;
    // this makes the PF topology library render arrows on both sides
    startTerminalType?: EdgeTerminalType;
    endTerminalType?: EdgeTerminalType;
    // previous was `properties`
    sourceToTargetProperties: EdgeProperties[];
    // this is for holding on to properties for bidirectional edges
    targetToSourceProperties?: EdgeProperties[];
    isBidirectional: boolean;
};
