import { RbacConfigType } from 'constants/entityTypes';

export const severityLabels = Object.freeze({
    CRITICAL_SEVERITY: 'Critical',
    HIGH_SEVERITY: 'High',
    MEDIUM_SEVERITY: 'Medium',
    LOW_SEVERITY: 'Low',
});

export const clusterTypeLabels = Object.freeze({
    KUBERNETES_CLUSTER: 'Kubernetes Clusters',
    SWARM_CLUSTER: 'Swarm Clusters',
    OPENSHIFT_CLUSTER: 'OpenShift Clusters',
});

export const clusterVersionLabels = Object.freeze({
    KUBERNETES_CLUSTER: 'K8s Version',
    SWARM_CLUSTER: 'Swarm Version',
    OPENSHIFT_CLUSTER: 'OpenShift Version',
});

export const healthStatusLabels = Object.freeze({
    UNINITIALIZED: 'Uninitialized',
    UNAVAILABLE: 'Unavailable',
    UNHEALTHY: 'Unhealthy',
    DEGRADED: 'Degraded',
    HEALTHY: 'Healthy',
});

export const lifecycleStageLabels = Object.freeze({
    BUILD: 'Build',
    DEPLOY: 'Deploy',
    RUNTIME: 'Runtime',
});

export const enforcementActionLabels = Object.freeze({
    UNSET_ENFORCEMENT: 'None',
    FAIL_BUILD_ENFORCEMENT: 'Fail builds during continuous integration',
    SCALE_TO_ZERO_ENFORCEMENT: 'Scale to Zero Replicas',
    KILL_POD_ENFORCEMENT: 'Kill Pod',
    UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT: 'Add an Unsatisfiable Node Constraint',
});

export const accessControl = Object.freeze({
    NO_ACCESS: 'No Access',
    READ_ACCESS: 'Read Access',
    READ_WRITE_ACCESS: 'Read and Write Access',
});

// TODO research inconsistency with resourceTypes: NETWORK_POLICY versus CHECK
export const resourceLabels = Object.freeze({
    CLUSTER: 'cluster',
    NAMESPACE: 'namespace',
    NODE: 'node',
    DEPLOYMENT: 'deployment',
    SECRET: 'secret',
    CONTROL: 'control',
    CVE: 'CVE',
    COMPONENT: 'component',
    IMAGE: 'image',
    POLICY: 'policy',
    CHECK: 'check',
});

export const rbacConfigLabels: Record<RbacConfigType, string> = Object.freeze({
    SUBJECT: 'users and groups',
    SERVICE_ACCOUNT: 'service account',
    ROLE: 'role',
});

export const stackroxSupport = Object.freeze({
    phoneNumber: {
        withSpaces: '1 (650) 385-8329',
        withDashes: '1-650-385-8329',
    },
    email: 'support@stackrox.com',
});

export const portExposureLabels = Object.freeze({
    EXTERNAL: 'LoadBalancer',
    NODE: 'NodePort',
    HOST: 'HostPort',
    INTERNAL: 'ClusterIP',
    UNSET: 'Exposure type is not set',
});

// For any update to rbacPermissionLabels, please also update policy.proto
export const rbacPermissionLabels = Object.freeze({
    DEFAULT: 'Default Access',
    ELEVATED_IN_NAMESPACE: 'Elevated Access in Namespace',
    ELEVATED_CLUSTER_WIDE: 'Elevated Access Cluster Wide',
    CLUSTER_ADMIN: 'Cluster Admin Access',
});

// For any update to envVarSrcLabels, please also update deployment.proto
export const envVarSrcLabels = Object.freeze({
    RAW: 'NoObjectRef (Raw Value)',
    SECRET_KEY: 'SecretKeyRef',
    CONFIG_MAP_KEY: 'ConfigMapRef',
    FIELD: 'FieldRef',
    RESOURCE_FIELD: 'ResourceFieldRef',
});

export const policyCriteriaCategories = Object.freeze({
    IMAGE_REGISTRY: 'Image Registry',
    IMAGE_CONTENTS: 'Image Contents',
    CONTAINER_CONFIGURATION: 'Container Configuration',
    DEPLOYMENT_METADATA: 'Deployment Metadata',
    STORAGE: 'Storage',
    NETWORKING: 'Networking',
    PROCESS_ACTIVITY: 'Process Activity',
    KUBERNETES_ACCESS: 'Kubernetes Access',
    KUBERNETES_EVENTS: 'Kubernetes Events',
});
