package env

import "os"

const (
	defaultOpenshiftAPI = `false`
)

var (
	// OpenshiftAPI is whether or not the k8s listener should talk via the openshift API
	OpenshiftAPI = Setting(openshift{})
)

type openshift struct{}

func (openshift) EnvVar() string {
	return `ROX_OPENSHIFT_API`
}

func (o openshift) Setting() string {
	if n, ok := os.LookupEnv(o.EnvVar()); ok {
		return n
	}
	return defaultOpenshiftAPI
}
