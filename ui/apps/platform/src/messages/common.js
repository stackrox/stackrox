const common = Object.freeze({
    severityLabels: {
        CRITICAL_SEVERITY: 'Critical',
        HIGH_SEVERITY: 'High',
        MEDIUM_SEVERITY: 'Medium',
        LOW_SEVERITY: 'Low',
    },
    clusterTypeLabels: {
        KUBERNETES_CLUSTER: 'Kubernetes Clusters',
        SWARM_CLUSTER: 'Swarm Clusters',
        OPENSHIFT_CLUSTER: 'OpenShift Clusters',
    },
    clusterVersionLabels: {
        KUBERNETES_CLUSTER: 'K8s Version',
        SWARM_CLUSTER: 'Swarm Version',
        OPENSHIFT_CLUSTER: 'OpenShift Version',
    },
    lifecycleStageLabels: {
        BUILD: 'Build',
        DEPLOY: 'Deploy',
        RUNTIME: 'Runtime',
    },
    enforcementActionLabels: {
        UNSET_ENFORCEMENT: 'None',
        FAIL_BUILD_ENFORCEMENT: 'Fail builds during continuous integration',
        SCALE_TO_ZERO_ENFORCEMENT: 'Scale to Zero Replicas',
        KILL_POD_ENFORCEMENT: 'Kill Pod',
        UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT: 'Add an Unsatisfiable Node Constraint',
    },
    accessControl: {
        NO_ACCESS: 'No Access',
        READ_ACCESS: 'Read Access',
        READ_WRITE_ACCESS: 'Read and Write Access',
    },
    resourceLabels: {
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
    },
    rbacConfigLabels: {
        SUBJECT: 'users and groups',
        SERVICE_ACCOUNT: 'service account',
        ROLE: 'role',
    },
    stackroxSupport: {
        phoneNumber: {
            withSpaces: '1 (650) 385-8329',
            withDashes: '1-650-385-8329',
        },
        email: 'support@stackrox.com',
    },
    portExposureLabels: {
        EXTERNAL: 'LoadBalancer',
        NODE: 'NodePort',
        HOST: 'HostPort',
        INTERNAL: 'ClusterIP',
    },
    // For any update to rbacPermissionLabels, please also update policy.proto
    rbacPermissionLabels: {
        DEFAULT: 'Default Access',
        ELEVATED_IN_NAMESPACE: 'Elevated Access in Namespace',
        ELEVATED_CLUSTER_WIDE: 'Elevated Access Cluster Wide',
        CLUSTER_ADMIN: 'Cluster Admin Access',
    },
    // For any update to envVarSrcLabels, please also update deployment.proto
    envVarSrcLabels: {
        RAW: 'NoObjectRef (Raw Value)',
        SECRET_KEY: 'SecretKeyRef',
        CONFIG_MAP_KEY: 'ConfigMapRef',
        FIELD: 'FieldRef',
        RESOURCE_FIELD: 'ResourceFieldRef',
    },
    policyCriteriaCategories: {
        IMAGE_REGISTRY: 'Image Registry',
        IMAGE_CONTENTS: 'Image Contents',
        CONTAINER_CONFIGURATION: 'Container Configuration',
        DEPLOYMENT_METADATA: 'Deployment Metadata',
        STORAGE: 'Storage',
        NETWORKING: 'Networking',
        PROCESS_ACTIVITY: 'Process Activity',
        KUBERNETES_ACCESS: 'Kubernetes Access',
    },
});

module.exports = common;
