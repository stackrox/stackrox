type EnforcementAction =
    | 'UNSET_ENFORCEMENT'
    | 'SCALE_TO_ZERO_ENFORCEMENT'
    | 'UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT'
    | 'KILL_POD_ENFORCEMENT'
    | 'FAIL_BUILD_ENFORCEMENT'
    | 'FAIL_KUBE_REQUEST_ENFORCEMENT'
    | 'FAIL_DEPLOYMENT_CREATE_ENFORCEMENT'
    | 'FAIL_DEPLOYMENT_UPDATE_ENFORCEMENT';

type LifecycleStage = 'BUILD' | 'DEPLOY' | 'RUNTIME';

type PolicySeverity = 'CRITICAL_SEVERITY' | 'HIGH_SEVERITY' | 'MEDIUM_SEVERITY' | 'LOW_SEVERITY';

export type Violation = {
    id: string;
    commonEntityInfo: {
        clusterId: string;
        clusterName: string;
        namespace: string;
        namespaceId: string;
        resourceType: string;
    };
    deployment?: {
        clusterId: string;
        clusterName: string;
        id: string;
        inactive: boolean;
        name: string;
        namespace: string;
        namespaceId: string;
    };
    enforcementAction: EnforcementAction;
    enforcementCount: number;
    lifecycleStage: LifecycleStage;
    policy: {
        categories: string[];
        description: string;
        id: string;
        name: string;
        severity: PolicySeverity;
    };
    state: 'ACTIVE' | 'INACTIVE';
    tags: string[];
    time: string;
};

export default Violation;
