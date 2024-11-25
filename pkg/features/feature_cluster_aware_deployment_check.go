package features

// ClusterAwareDeploymentCheck enables roxctl deployment check to check deployments on the cluster level.
var ClusterAwareDeploymentCheck = registerFeature("Enables cluster level check for the 'roxctl deployment check' command.", "ROX_CLUSTER_AWARE_DEPLOYMENT_CHECK", enabled)
