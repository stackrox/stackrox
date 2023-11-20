package env

// AWSCloudCredentialsSecret is the variable that specifies the name of the AWS cloud credentials secret.
var AWSCloudCredentialsSecret = RegisterSetting("ROX_AWS_CLOUD_CREDENTIALS_SECRET", WithDefault("awsCloudCredentials"))

// GCPCloudCredentialsSecret is the variable that specifies the name of the GCP cloud credentials secret.
var GCPCloudCredentialsSecret = RegisterSetting("ROX_GCP_CLOUD_CREDENTIALS_SECRET", WithDefault("gcp-cloud-credentials"))
