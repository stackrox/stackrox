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

export const policySeverities = [
    'LOW_SEVERITY',
    'MEDIUM_SEVERITY',
    'HIGH_SEVERITY',
    'CRITICAL_SEVERITY',
] as const;
export type PolicySeverity = (typeof policySeverities)[number];

export type LifecycleStage = 'DEPLOY' | 'BUILD' | 'RUNTIME';

export type PolicyEventSource = 'NOT_APPLICABLE' | 'DEPLOYMENT_EVENT' | 'AUDIT_LOG_EVENT';

type BasePolicy = {
    rationale: string;
    remediation: string;
    categories: string[];
    exclusions: PolicyExclusion[];
    scope: PolicyScope[];
    enforcementActions: EnforcementAction[];
    SORTName: string; // For internal use only.
    SORTLifecycleStage: string; // For internal use only.
    SORTEnforcement: boolean; // For internal use only.
    policyVersion: string;
    mitreAttackVectors: PolicyMitreAttackVector[];
    readonly criteriaLocked: boolean; // If true, the policy's criteria fields are rendered read-only.
    readonly mitreVectorsLocked: boolean; // If true, the policy's MITRE ATT&CK fields are rendered read-only.
} & ListPolicy;

// the policy object we use client side for ease of form manipulation
export type ClientPolicy = {
    excludedImageNames: string[]; // For internal use only.
    excludedDeploymentScopes: PolicyExcludedDeployment[]; // For internal use only.
    serverPolicySections: PolicySection[]; // For internal use only.
    policySections: ClientPolicySection[]; // value strings converted into objects
} & BasePolicy;

export type Policy = {
    policySections: PolicySection[]; // values are strings
} & BasePolicy;

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

type ClientPolicySection = {
    sectionName: string;
    policyGroups: ClientPolicyGroup[];
};

type ClientPolicyGroup = {
    fieldName: string;
    booleanOperator: PolicyBooleanOperator;
    negate: boolean;
    values: ClientPolicyValue[];
};

export type PolicyGroup = {
    fieldName: string;
    booleanOperator: PolicyBooleanOperator;
    negate: boolean;
    values: PolicyValue[];
};

export type PolicyBooleanOperator = 'OR' | 'AND';

export type PolicyValue = {
    value: string;
};

export type ValueObj = {
    source?: string;
    key?: string;
    value?: string;
};

export type ClientPolicyValue = {
    value?: ValueObj;
    arrayValue?: string[];
};

// TODO supersedes MitreAttackVectorId in src/services/MitreService.ts
export type PolicyMitreAttackVector = {
    tactic: string; // tactic id
    techniques: string[]; // technique ids
};

export type PolicyCategory = {
    id: string;
    // central/policycategory/service/service_impl.go
    // policy category must have a name between 5 and 128 characters long with no new lines or dollar signs
    name: string;
    isDefault: boolean;
};
