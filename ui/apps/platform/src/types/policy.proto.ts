import { PermissionLevel } from './rbac.proto';

export type ListPolicy = {
    id: string;
    name: string;
    description: string;
    severity: PolicySeverity;
    disabled: boolean;
    lifecycleStages: LifecycleStage[];
    notifiers: string[];
    lastUpdated: string | null; // ISO 8601 date string
    eventSource: PolicyEventSource;
    readonly isDefault: boolean; // Indicates the policy is a default policy if true and a custom policy if false.
};

// TODO supersedes src/Containers/Violations/PatternFly/types/violationTypes.ts
export type PolicySeverity =
    | 'LOW_SEVERITY'
    | 'MEDIUM_SEVERITY'
    | 'HIGH_SEVERITY'
    | 'CRITICAL_SEVERITY';

// TODO supersedes src/Containers/Violations/PatternFly/types/violationTypes.ts
export type LifecycleStage = 'DEPLOY' | 'BUILD' | 'RUNTIME';

export type PolicyEventSource = 'NOT_APPLICABLE' | 'DEPLOYMENT_EVENT' | 'AUDIT_LOG_EVENT';

export type Policy = {
    rationale: string;
    remediation: string;
    categories: string[];
    fields: PolicyFields | null;
    // whitelists is deprecated and superseded by exlusions
    exclusions: PolicyExclusion[];
    scope: PolicyScope[];
    enforcementActions: EnforcementAction[];
    excludedImageNames: string[]; // For internal use only.
    excludedDeploymentScopes: PolicyExcludedDeployment[]; // For internal use only.
    SORT_name: string; // For internal use only.
    SORT_lifecycleStage: string; // For internal use only.
    SORT_enforcement: boolean; // For internal use only.
    policyVersion: string;
    policySections: PolicySection[];
    mitreAttackVectors: PolicyMitreAttackVector[];
    readonly criteriaLocked: boolean; // If true, the policy's criteria fields are rendered read-only.
    readonly mitreVectorsLocked: boolean; // If true, the policy's MITRE ATT&CK fields are rendered read-only.
} & ListPolicy;

export type PolicyExclusion = PolicyDeploymentExclusion | PolicyImageExclusion;

// TODO prefer initial values instead of optional properties while adding a new policy?
export type PolicyDeploymentExclusion = {
    deployment: PolicyExcludedDeployment;
    image: null;
} & PolicyBaseExclusion;

export type PolicyExcludedDeployment = {
    name: string;
    scope: PolicyScope | null;
};

export type PolicyImageExclusion = {
    deployment: null;
    image: {
        name: string;
    };
} & PolicyBaseExclusion;

// TODO prefer initial values instead of optional properties while adding a new policy?
export type PolicyBaseExclusion = {
    name: string;
    expiration: string | null; // ISO 8601 date string
};

// TODO prefer initial values instead of optional properties while adding a new policy?
export type PolicyScope = {
    cluster?: string;
    namespace?: string;
    label?: PolicyScopeLabel | null;
};

export type PolicyScopeLabel = {
    key: string;
    value: string;
};

// TODO supersedes apps/platform/src/Containers/Violations/PatternFly/types/violationTypes.ts
// FAIL_KUBE_REQUEST_ENFORCEMENT takes effect only if admission control webhook is enabled to listen on exec and port-forward events.
// FAIL_DEPLOYMENT_CREATE_ENFORCEMENT takes effect only if admission control webhook is configured to enforce on object creates/updates.
// FAIL_DEPLOYMENT_UPDATE_ENFORCEMENT takes effect only if admission control webhook is configured to enforce on object updates.
export type EnforcementAction =
    | 'UNSET_ENFORCEMENT'
    | 'SCALE_TO_ZERO_ENFORCEMENT'
    | 'UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT'
    | 'KILL_POD_ENFORCEMENT'
    | 'FAIL_BUILD_ENFORCEMENT'
    | 'FAIL_KUBE_REQUEST_ENFORCEMENT'
    | 'FAIL_DEPLOYMENT_CREATE_ENFORCEMENT'
    | 'FAIL_DEPLOYMENT_UPDATE_ENFORCEMENT';

export type PolicySection = {
    sectionName: string;
    policyGroups: PolicyGroup[];
};

export type ValueObj = {
    source?: string;
    key?: string;
    value?: string;
};

export type PolicyGroup = {
    fieldName: string;
    booleanOperator: PolicyBooleanOperator;
    negate: boolean;
    values: PolicyValue[];
};

export type PolicyBooleanOperator = 'OR' | 'AND';

export type PolicyValue = {
    value: string | ValueObj;
};

// TODO supersedes MitreAttackVectorId in src/services/MitreService.ts
export type PolicyMitreAttackVector = {
    tactic: string; // tactic id
    techniques: string[]; // technique ids
};

export type PolicyFields = {
    imageName: PolicyImageName | null;

    // Registry metadata
    imageAgeDays?: string; // int64
    lineRule: DockerfileLineRuleField | null;

    // Scan Metadata
    cvss: NumericalPolicy | null;
    cve: string;

    component: PolicyComponent | null;
    scanAgeDays?: string; // int64

    noScanExists?: boolean; // Whether to alert if no scan exists for an image.

    env: PolicyKeyValue | null;
    command: string;
    args: string;
    directory: string;
    user: string;

    volumePolicy: VolumePolicy | null;

    portPolicy: PortPolicy | null;
    requiredLabel: PolicyKeyValue | null;
    requiredAnnotation: PolicyKeyValue | null;
    disallowedAnnotation: PolicyKeyValue | null;

    privileged?: boolean;
    dropCapabilities: string[];
    addCapabilities: string[];

    containerResourcePolicy: ResourcePolicy | null;
    processPolicy: ProcessPolicy | null;

    readOnlyRootFs?: boolean;
    fixedBy: string;

    portExposurePolicy: PortExposurePolicy | null;
    permissionPolicy: PermissionPolicy | null;
    hostMountPolicy: HostMountPolicy | null;
    whitelistEnabled?: boolean;

    requiredImageLabel: PolicyKeyValue | null;
    disallowedImageLabel: PolicyKeyValue | null;
};

export type PolicyImageName = {
    registry: string; // e.g. docker.io
    remote: string; // e.g. stackrox/container-summarizer
    tag: string; // e.g. latest
};

export type DockerfileLineRuleField = {
    instruction: string;
    value: string;
};

export type NumericalPolicy = {
    op: PolicyComparator;
    value: number; // float
};

export type PolicyComparator =
    | 'LESS_THAN'
    | 'LESS_THAN_OR_EQUALS'
    | 'EQUALS'
    | 'GREATER_THAN_OR_EQUALS'
    | 'GREATER_THAN';

export type PolicyComponent = {
    name: string;
    version: string;
};

export type PolicyKeyValue = {
    key: string;
    value: string;
    envVarSource: EnvVarSource;
};

// TODO import from types/deployment.proto.ts
export type EnvVarSource =
    | 'UNSET'
    | 'RAW'
    | 'SECRET_KEY'
    | 'CONFIG_MAP_KEY'
    | 'FIELD'
    | 'RESOURCE_FIELD'
    | 'UNKNOWN';

export type VolumePolicy = {
    name: string;
    source: string;
    destination: string;
    readOnly?: boolean;
    type: string;
};

export type PortPolicy = {
    port: number; // int32
    protocol: string;
};

export type ResourcePolicy = {
    cpuResourceRequest: NumericalPolicy | null;
    cpuResourceLimit: NumericalPolicy | null;
    memoryResourceRequest: NumericalPolicy | null;
    memoryResourceLimit: NumericalPolicy | null;
};

export type ProcessPolicy = {
    name: string;
    args: string;
    ancestor: string;
    uid: string;
};

export type PortExposurePolicy = {
    exposureLevels: PortExposureLevel[];
};

// TODO import from types/deployment.proto.ts
export type PortExposureLevel = 'UNSET' | 'EXTERNAL' | 'NODE' | 'INTERNAL' | 'HOST';

// K8S RBAC Permission level configuration.
export type PermissionPolicy = {
    permissionLevel: PermissionLevel;
};

export type HostMountPolicy = {
    readOnly?: boolean;
};
