import { Model, NodeModel } from '@patternfly/react-topology';

import { ListenPort } from 'types/networkFlow.proto';
import { Override } from 'utils/type.utils';

export type CustomModel = Override<Model, { nodes?: CustomNodeModel[] }>;

export type CustomNodeModel =
    | NamespaceNodeModel
    | DeploymentNodeModel
    | ExternalNodeModel
    | ExternalEntitiesNodeModel
    | CIDRBlockNodeModel
    | ExtraneousNodeModel;

export type NamespaceNodeModel = Override<NodeModel, { data: NamespaceData }>;

export type DeploymentNodeModel = Override<NodeModel, { data: DeploymentData }>;

export type ExternalNodeModel = Override<NodeModel, { data: ExternalData }>;

export type ExternalEntitiesNodeModel = Override<NodeModel, { data: ExternalEntitiesData }>;

export type CIDRBlockNodeModel = Override<NodeModel, { data: CIDRBlockData }>;

export type ExtraneousNodeModel = Override<NodeModel, { data: ExtraneousData }>;

export type CustomNodeData =
    | NamespaceData
    | DeploymentData
    | ExternalData
    | ExternalEntitiesData
    | CIDRBlockData;

export type NamespaceData = {
    type: 'NAMESPACE';
    collapsible: boolean;
    showContextMenu: boolean;
};

export type NetworkPolicyState = 'none' | 'both' | 'ingress' | 'egress';

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
};

export type ExternalData = {
    type: 'EXTERNAL';
    collapsible: boolean;
    showContextMenu: boolean;
};

export type ExternalEntitiesData = {
    type: 'EXTERNAL_ENTITIES';
    id: string;
};

export type CIDRBlockData = {
    type: 'CIDR_BLOCK';
    id: string;
    externalSource: {
        cidr?: string;
        default: boolean;
        name: string;
    };
};

export type ExtraneousData = {
    type: 'EXTRANEOUS';
    collapsible: boolean;
    showContextMenu: boolean;
    flows: number;
};
