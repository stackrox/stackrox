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
    }
});

module.exports = common;
