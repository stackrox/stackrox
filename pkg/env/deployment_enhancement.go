package env

import "time"

var (
	// CentralDeploymentEnhancementTimeout allows to configure the time Central waits for Sensor to answer to a
	// DeploymentEnhancementRequest.
	CentralDeploymentEnhancementTimeout = registerDurationSetting("ROX_CENTRAL_DEPLOYMENT_ENHANCE_TIMEOUT", 30*time.Second)

	// SensorDeploymentEnhancementQueueSize configures the size of the buffered channel that incoming
	// DeploymentEnhancementRequests are queued in
	SensorDeploymentEnhancementQueueSize = RegisterIntegerSetting("ROX_SENSOR_DEPLOYMENT_ENHANCEMENT_QUEUE_SIZE", 50)
)
