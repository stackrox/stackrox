package features

// SensorDeploymentBuildOptimization enables a performance improvement by skipping deployments processing when no dependency or spec changed
var SensorDeploymentBuildOptimization = registerFeature("Enables a performance improvement by skipping deployments processing when no dependency or spec changed", "ROX_DEPLOYMENT_BUILD_OPTIMIZATION", enabled)
