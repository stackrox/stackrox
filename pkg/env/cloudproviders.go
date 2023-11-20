package env

// GCPCloudCredentialsSecret is the variable that specifies the name of the GCP cloud credentials secret.
var GCPCloudCredentialsSecret = RegisterSetting("ROX_GCP_CLOUD_CREDENTIALS_SECRET", WithDefault("gcpCloudCredentials"))
