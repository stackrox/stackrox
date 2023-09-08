import { ViolationState } from 'types/alert.proto';
import { EnforcementAction, LifecycleStage, ListPolicy, Policy } from 'types/policy.proto';

export type { EnforcementAction, LifecycleStage, Policy };

interface ListBaseAlert {
    id: string;
    commonEntityInfo: {
        clusterId: string;
        clusterName: string;
        namespace: string;
        namespaceId: string;
        resourceType: string;
    };
    enforcementAction: EnforcementAction;
    enforcementCount: number;
    lifecycleStage: LifecycleStage;
    policy: ListPolicy;
    state: ViolationState;
    time: string;
}

interface ListDeploymentAlert extends ListBaseAlert {
    deployment: {
        clusterId: string;
        clusterName: string;
        id: string;
        inactive: boolean;
        name: string;
        namespace: string;
        namespaceId: string;
    };
}

interface ListResourceAlert extends ListBaseAlert {
    resource: {
        name: string;
        resourceType: string;
    };
}

export type ListAlert = ListDeploymentAlert | ListResourceAlert;

export type NetworkFlowInfo = {
    protocol: string;
    source: {
        name: string;
        entityType: string;
        deploymentNamespace: string;
        deploymentType: string;
        port: string | number;
    };
    destination: {
        name: string;
        entityType: string;
        deploymentNamespace: string;
        deploymentType: string;
        port: string | number;
    };
};

export type Violation = {
    keyValueAttrs: {
        attrs: {
            key: string;
            value: string;
        }[];
    };
    message: string;
    time: string;
    type: string;
    networkFlowInfo?: NetworkFlowInfo;
};

export type ProcessViolation = {
    message: string;
    processes: {
        id: string;
    }[];
};

export type Alert = {
    id: string;
    deployment?: {
        clusterId: string;
        clusterName: string;
        id: string;
        inactive: boolean;
        name: string;
        namespace: string;
        namespaceId: string;
    };
    commonEntityInfo?: {
        clusterId: string;
        clusterName: string;
        namespace: string;
        namespaceId: string;
        resourceType: string;
    };
    resource?: {
        name: string;
        clusterName: string;
        resourceType: string;
    };
    enforcement?: {
        action: string;
        message: string;
    };
    firstOccurred: string;
    lifecycleStage: LifecycleStage;
    policy: Policy;
    state: ViolationState;
    time: string;
    violations: Violation[];
    processViolation?: ProcessViolation;
    resolvedAt?: string;
    snoozeTill?: string;
};
