package openshift

import (
	"strings"
	"testing"

	"github.com/stackrox/rox/pkg/auth/authproviders"
)

// NewTestFactoryFunc returns a function that creates a new factory for OpenShift oauth authprovider backends.
func NewTestFactoryFunc(t testing.TB) func(urlPathPrefix string) authproviders.BackendFactory {
	return func(urlPathPrefix string) authproviders.BackendFactory {
		urlPathPrefix = strings.TrimRight(urlPathPrefix, "/") + "/"
		return &factory{
			callbackURLPath: urlPathPrefix + callbackRelativePath,
			newBackendFunc:  newTestBackendFunc(t),
		}
	}
}

func newTestBackendFunc(_ testing.TB) func(id string, callbackURL string, config map[string]string) (*backend, error) {
	return func(id string, callbackURL string, _ map[string]string) (*backend, error) {
		b := &backend{
			id:                  id,
			baseRedirectURLPath: callbackURL,
			openshiftConnector:  nil,
		}
		return b, nil
	}
}
