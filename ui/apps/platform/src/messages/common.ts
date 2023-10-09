import { AccessControlEntityType, RbacConfigType } from 'constants/entityTypes';
import { VulnerabilitySeverity } from 'types/cve.proto';
import {
    EnforcementAction,
    LifecycleStage,
    PolicyEventSource,
    PolicySeverity,
} from 'types/policy.proto';

export const severityLabels: Record<PolicySeverity, string> = Object.freeze({
    CRITICAL_SEVERITY: 'Critical',
    HIGH_SEVERITY: 'High',
    MEDIUM_SEVERITY: 'Medium',
    LOW_SEVERITY: 'Low',
});

export const vulnerabilitySeverityLabels: Record<VulnerabilitySeverity, string> = Object.freeze({
    CRITICAL_VULNERABILITY_SEVERITY: 'Critical',
    IMPORTANT_VULNERABILITY_SEVERITY: 'Important',
    MODERATE_VULNERABILITY_SEVERITY: 'Moderate',
    LOW_VULNERABILITY_SEVERITY: 'Low',
    UNKNOWN_VULNERABILITY_SEVERITY: 'Unknown',
});

export const clusterTypeLabels = Object.freeze({
    KUBERNETES_CLUSTER: 'Kubernetes Clusters',
    SWARM_CLUSTER: 'Swarm Clusters',
    OPENSHIFT_CLUSTER: 'OpenShift Clusters',
    OPENSHIFT4_CLUSTER: 'OpenShift Clusters',
});

export const clusterVersionLabels = Object.freeze({
    KUBERNETES_CLUSTER: 'K8s Version',
    SWARM_CLUSTER: 'Swarm Version',
    OPENSHIFT_CLUSTER: 'OpenShift Version',
    OPENSHIFT4_CLUSTER: 'OpenShift Version',
});

export const healthStatusLabels = Object.freeze({
    UNINITIALIZED: 'Uninitialized',
    UNAVAILABLE: 'Unavailable',
    UNHEALTHY: 'Unhealthy',
    DEGRADED: 'Degraded',
    HEALTHY: 'Healthy',
});

export const lifecycleStageLabels: Record<LifecycleStage, string> = Object.freeze({
    BUILD: 'Build',
    DEPLOY: 'Deploy',
    RUNTIME: 'Runtime',
});

export const enforcementActionLabels: Record<EnforcementAction, string> = Object.freeze({
    UNSET_ENFORCEMENT: 'None',
    FAIL_BUILD_ENFORCEMENT: 'Fail builds during continuous integration',
    SCALE_TO_ZERO_ENFORCEMENT: 'Scale to Zero Replicas',
    KILL_POD_ENFORCEMENT: 'Kill Pod',
    FAIL_KUBE_REQUEST_ENFORCEMENT: 'Fail Kubernetes API Request',
    FAIL_DEPLOYMENT_CREATE_ENFORCEMENT: 'Block Deployment Create',
    FAIL_DEPLOYMENT_UPDATE_ENFORCEMENT: 'Block Deployment Update',
    UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT: 'Add an Unsatisfiable Node Constraint',
});

export const eventSourceLabels: Record<PolicyEventSource, string> = Object.freeze({
    NOT_APPLICABLE: 'N/A',
    DEPLOYMENT_EVENT: 'Deployment',
    AUDIT_LOG_EVENT: 'Audit log',
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
    IMAGE_CVE: 'Image CVE',
    NODE_CVE: 'Node CVE',
    CLUSTER_CVE: 'Platform CVE',
    COMPONENT: 'component',
    NODE_COMPONENT: 'node component',
    IMAGE_COMPONENT: 'image component',
    IMAGE: 'image',
    POLICY: 'policy',
    CHECK: 'check',
    ROLE: 'role',
});

export const rbacConfigLabels: Record<RbacConfigType, string> = Object.freeze({
    SUBJECT: 'users and groups',
    SERVICE_ACCOUNT: 'service account',
    ROLE: 'role',
});

export const accessControlLabels: Record<AccessControlEntityType, string> = {
    ACCESS_SCOPE: 'Access scope',
    AUTH_PROVIDER: 'Auth provider',
    PERMISSION_SET: 'Permission set',
    ROLE: 'Role',
};

export const portExposureLabels = Object.freeze({
    ROUTE: 'Route',
    EXTERNAL: 'LoadBalancer',
    NODE: 'NodePort',
    HOST: 'HostPort',
    INTERNAL: 'ClusterIP',
    UNSET: 'Exposure type is not set',
});

export const mountPropagationLabels = Object.freeze({
    NONE: 'None',
    HOST_TO_CONTAINER: 'Host to Container',
    BIDIRECTIONAL: 'Bidirectional',
});

export const seccompProfileTypeLabels = Object.freeze({
    UNCONFINED: 'Unconfined',
    RUNTIME_DEFAULT: 'Runtime Default',
    LOCALHOST: 'Localhost',
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
    IMAGE_REGISTRY: 'Image registry',
    IMAGE_CONTENTS: 'Image contents',
    CONTAINER_CONFIGURATION: 'Container configuration',
    DEPLOYMENT_METADATA: 'Deployment metadata',
    STORAGE: 'Storage',
    NETWORKING: 'Networking',
    PROCESS_ACTIVITY: 'Process activity',
    KUBERNETES_ACCESS: 'Kubernetes access',
    KUBERNETES_EVENTS: 'Kubernetes events',
});

// For any update to severityRatings, please also update cve.proto,
// pkg/booleanpolicy/value_regex.go, and Containers/Policies/Wizard/Form/utils.js.
export const severityRatings = Object.freeze({
    UNKNOWN: 'Unknown',
    LOW: 'Low',
    MODERATE: 'Moderate',
    IMPORTANT: 'Important',
    CRITICAL: 'Critical',
});
