package utils

import "github.com/stackrox/rox/pkg/env"

var serviceOperatorCAPathSetting = env.RegisterSetting("ROX_SERVICE_OPERATOR_CA_PATH",
	env.WithDefault("/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt"))

// ServiceOperatorCAPath returns the path to the service-ca operator CA certificate.
func ServiceOperatorCAPath() string {
	return serviceOperatorCAPathSetting.Setting()
}
