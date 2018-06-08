package env

import "os"

const (
	defaultNamespace = `stackrox`
)

var (
	// Namespace is the namespace in which sensors and benchmark services are deployed (k8s).
	Namespace = Setting(namespace{})
	// ServiceAccount designates the account under which sensors and benchmarks operate (k8s).
	ServiceAccount = Setting(serviceAccount{})
	// ImagePullSecrets are secrets used for pulling images (k8s).
	ImagePullSecrets = Setting(imagePullSecrets{})
)

type namespace struct{}

func (namespace) EnvVar() string {
	return `ROX_PREVENT_NAMESPACE`
}

func (ns namespace) Setting() string {
	if n, ok := os.LookupEnv(ns.EnvVar()); ok {
		return n
	}

	return defaultNamespace
}

type serviceAccount struct{}

func (serviceAccount) EnvVar() string {
	return `ROX_PREVENT_SERVICE_ACCOUNT`
}

func (sa serviceAccount) Setting() string {
	return os.Getenv(sa.EnvVar())
}

type imagePullSecrets struct{}

func (imagePullSecrets) EnvVar() string {
	return `ROX_PREVENT_IMAGE_PULL_CONFIG`
}

// Values interpreted as comma separated list of secret names.
func (ips imagePullSecrets) Setting() string {
	return os.Getenv(ips.EnvVar())
}
