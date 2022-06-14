package common

import (
	"github.com/stackrox/rox/roxctl/common/logger"
)

// LogInfoPspEnabled writes informational message about PodSecurityPolicies being enabled to the provided logger.
func LogInfoPspEnabled(logger logger.Logger) {
	logger.InfofLn("Deployment bundle includes PodSecurityPolicies (PSPs). This is incompatible with Kubernetes >= v1.25.")
	logger.InfofLn("Use --enable-pod-security-policies=false to disable PodSecurityPolicies.")
	logger.InfofLn("For the time being PodSecurityPolicies remain enabled by default in deployment bundles and need to be disabled explicitly for Kubernetes >= v1.25.")
}

// LogInfoPspDisabled writes informational message about PodSecurityPolicies being disabled to the provided logger.
func LogInfoPspDisabled(logger logger.Logger) {
	logger.InfofLn("Deployment bundle does not include PodSecurityPolicies (PSPs).")
	logger.InfofLn("This is incompatible with pre-v1.25 Kubernetes installations having the PodSecurityPolicy Admission Controller plugin enabled.")
	logger.InfofLn("Use --enable-pod-security-policies if PodSecurityPolicies are required for your Kubernetes environment.")
}
