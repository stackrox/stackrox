const common = Object.freeze({
    severityLabels: {
        CRITICAL_SEVERITY: 'Critical',
        HIGH_SEVERITY: 'High',
        MEDIUM_SEVERITY: 'Medium',
        LOW_SEVERITY: 'Low'
    },
    clusterTypeLabels: {
        KUBERNETES_CLUSTER: 'Kubernetes Clusters',
        DOCKER_EE_CLUSTER: 'Docker EE Clusters',
        SWARM_CLUSTER: 'Swarm Clusters',
        OPENSHIFT_CLUSTER: 'OpenShift Clusters'
    },
    categoriesLabels: {
        IMAGE_ASSURANCE: 'Image Assurance',
        CONTAINER_CONFIGURATION: 'Container Configuration',
        PRIVILEGES_CAPABILITIES: 'Privileges and Capabilities'
    }
});

module.exports = common;
