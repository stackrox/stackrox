package openshift

import (
	"strings"
	"testing"

	"github.com/stackrox/rox/pkg/auth/authproviders"
)

// NewTestFactoryCreator returns a function that creates a new factory
// for OpenShift oauth AuthProvider backends.
func NewTestFactoryCreator(t testing.TB) authproviders.BackendFactoryCreator {
	return func(urlPathPrefix string) authproviders.BackendFactory {
		urlPathPrefix = strings.TrimRight(urlPathPrefix, "/") + "/"
		return &factory{
			callbackURLPath: urlPathPrefix + callbackRelativePath,
			newBackend:      newTestBackend(t),
		}
	}
}

func newTestBackend(_ testing.TB) newBackendFunc {
	return func(id string, callbackURL string, _ map[string]string) (*backend, error) {
		b := &backend{
			id:                  id,
			baseRedirectURLPath: callbackURL,
			openshiftConnector:  nil,
		}
		registerBackend(b)
		return b, nil
	}
}
