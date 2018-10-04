const common = Object.freeze({
    severityLabels: {
        CRITICAL_SEVERITY: 'Critical',
        HIGH_SEVERITY: 'High',
        MEDIUM_SEVERITY: 'Medium',
        LOW_SEVERITY: 'Low'
    },
    clusterTypeLabels: {
        KUBERNETES_CLUSTER: 'Kubernetes Clusters',
        SWARM_CLUSTER: 'Swarm Clusters',
        OPENSHIFT_CLUSTER: 'OpenShift Clusters'
    },
    lifecycleStageLabels: {
        BUILD_TIME: 'Build',
        DEPLOY_TIME: 'Deploy',
        RUN_TIME: 'Runtime'
    },
    enforcementActionLabels: {
        UNSET_ENFORCEMENT: 'None',
        SCALE_TO_ZERO_ENFORCEMENT: 'Scale to Zero Replicas',
        KILL_POD_ENFORCEMENT: 'Kill Pod',
        UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT: 'Add an Unsatisfiable Node Constraint'
    }
});

module.exports = common;
