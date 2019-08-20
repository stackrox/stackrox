package env

var (
	// ImageClusterNSScopes is the variable that determines if ClusterNS scopes will be added to the image
	ImageClusterNSScopes = RegisterSetting("ROX_IMAGE_SAC", WithDefault("false"))
)
