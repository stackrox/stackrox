import { ContainerImage } from './deployment.proto';
import { L4Protocol, NetworkEntityInfoType } from './networkFlow.proto';
import { EnforcementAction, LifecycleStage, Policy, PolicySeverity } from './policy.proto';
import { ProcessIndicator } from './processIndicator.proto';

// Alert is for violation page.

// An alert cannot be on more than one entity (deployment, container image, resource, etc.)
export type Alert = DeploymentAlert | ImageAlert | ResourceAlert;

export type DeploymentAlert = {
    deployment: {
        id: string;
        name: string;
        type: string;
        namespace: string;
        namespaceId: string;
        labels: Record<string, string>;
        clusterId: string;
        clusterName: string;
        containers: {
            image: ContainerImage;
            name: string;
        }[];
        annotations: Record<string, string>;
        inactive: boolean;
    };
} & BaseAlert;

export type ImageAlert = {
    image: ContainerImage;
} & BaseAlert;

export type ResourceAlert = {
    resource: {
        resourceType: AlertResourceType;
        name: string;
        clusterId: string;
        clusterName: string;
        namespace: string;
        namespaceId: string;
    };
} & BaseAlert;

export type AlertResourceType =
    | 'UNKNOWN'
    | 'SECRETS'
    | 'CONFIGMAPS'
    | 'CLUSTER_ROLES'
    | 'CLUSTER_ROLE_BINDINGS'
    | 'NETWORK_POLICIES'
    | 'SECURITY_CONTEXT_CONSTRAINTS'
    | 'EGRESS_FIREWALLS';

export function isDeploymentAlert(alert: Alert): alert is DeploymentAlert {
    return 'deployment' in alert && Boolean(alert.deployment);
}

export function isImageAlert(alert: Alert): alert is ImageAlert {
    return 'image' in alert && Boolean(alert.image);
}

export function isResourceAlert(alert: Alert): alert is ResourceAlert {
    return 'resource' in alert && Boolean(alert.resource);
}

export type BaseAlert = {
    id: string;
    policy: Policy;
    lifecycleStage: LifecycleStage;
    violations: Violation[]; // For run-time phase alert, a maximum of 40 violations are retained.
    processViolation: ProcessViolation | null;
    enforcement: AlertEnforcement | null;
    time: string; // ISO 8601 date string
    firstOccurred: string; // ISO 8601 date string
    resolvedAt: string | null; // ISO 8601 date string only if ViolationState is RESOLVED
    state: ViolationState;
    snoozeTill: string | null; // ISO 8601 date string
};

export type Violation = GenericViolation | K8sEventViolation | NetworkFlowViolation;

export type GenericViolation = {
    type: 'GENERIC';
} & BaseViolation;

export type K8sEventViolation = {
    type: 'K8S_EVENT';
    keyValueAttrs: {
        attrs: {
            key: string;
            value: string;
        }[]; // TODO import a reusable KeyValueAttribute type?
    };
} & BaseViolation;

export type NetworkFlowViolation = {
    type: 'NETWORK_FLOW';
    networkFlowInfo: NetworkFlowInfo;
} & BaseViolation;

export type NetworkFlowInfo = {
    protocol: L4Protocol;
    source: NetworkFlowInfoEntity;
    destination: NetworkFlowInfoEntity;
};

export type NetworkFlowInfoEntity = {
    name: string;
    entityType: NetworkEntityInfoType;
    deploymentNamespace: string;
    deploymentType: string;
    port: string | number; // int32 TODO verify is it just number?
};

type BaseViolation = {
    type: ViolationType;
    message: string;
    // Indicates violation time. This field differs from top-level field 'time' which represents last time the alert
    // occurred in case of multiple occurrences of the policy alert. As of 55.0, this field is set only for kubernetes
    // event violations, but may not be limited to it in future.
    time: string | null; // ISO 8601 date string
};

export type ViolationType = 'GENERIC' | 'K8S_EVENT' | 'NETWORK_FLOW' | 'NETWORK_POLICY';

export type ProcessViolation = {
    message: string;
    processes: ProcessIndicator[];
};

export type AlertEnforcement = {
    action: EnforcementAction;
    message: string;
};

export type ViolationState = 'ACTIVE' | 'SNOOZED' | 'RESOLVED' | 'ATTEMPTED';

// ListAlert is for violations list.

export type ListAlert = DeploymentListAlert | ResourceListAlert;

type DeploymentListAlert = {
    commonEntityInfo: CommonEntityInfo & {
        resourceType: 'DEPLOYMENT';
    };
    deployment: {
        id: string;
        name: string;
        clusterName: string;
        namespace: string;
        clusterId: string;
        inactive: boolean;
        namespaceId: string;
    };
} & BaseListAlert;

export type ResourceListAlert = {
    commonEntityInfo: CommonEntityInfo & {
        resourceType:
            | 'SECRETS'
            | 'CONFIGMAPS'
            | 'CLUSTER_ROLES'
            | 'CLUSTER_ROLE_BINDINGS'
            | 'NETWORK_POLICIES'
            | 'SECURITY_CONTEXT_CONSTRAINTS'
            | 'EGRESS_FIREWALLS';
    };
    resource: {
        name: string;
    };
} & BaseListAlert;

export type CommonEntityInfo = {
    clusterName: string;
    namespace: string;
    clusterId: string;
    namespaceId: string;
    resourceType: ListAlertResourceType;
};

/*
 * A special ListAlert-only enumeration of resource types.
 * Unlike AlertResourceType this also includes deployment as a type.
 * This must be kept in sync with AlertResourceType (excluding the deployment value).
 */
export type ListAlertResourceType =
    | 'DEPLOYMENT'
    | 'SECRETS'
    | 'CONFIGMAPS'
    | 'CLUSTER_ROLES'
    | 'CLUSTER_ROLE_BINDINGS'
    | 'NETWORK_POLICIES'
    | 'SECURITY_CONTEXT_CONSTRAINTS'
    | 'EGRESS_FIREWALLS';

export type BaseListAlert = {
    id: string;
    lifecycleStage: LifecycleStage;
    time: string; // ISO 8601 date string
    policy: ListAlertPolicy;
    state: ViolationState;
    enforcementCount: number;
    enforcementAction: EnforcementAction;
    commonEntityInfo: CommonEntityInfo;
};

export type ListAlertPolicy = {
    id: string;
    name: string;
    severity: PolicySeverity;
    description: string;
    categories: string[];
};
