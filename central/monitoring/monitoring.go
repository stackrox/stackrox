package monitoring

const (
	// PasswordPath is the path where the password is stored within the contianer
	PasswordPath = "/run/secrets/stackrox.io/monitoring/password"

	// CAPath is where the monitoring CA is stored within the container
	CAPath = "/run/secrets/stackrox.io/monitoring/ca.pem"
)
