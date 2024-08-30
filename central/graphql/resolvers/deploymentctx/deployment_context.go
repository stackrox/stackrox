package deploymentctx

import (
	"context"
)

// distroContextKey is the key for the distribution value in the context.
type deploymentContextKey struct{}

// deploymentContextValue holds the value of the distro in the context.
type deploymentContextValue struct {
	deployment string
}

// FromContext returns the deployment from the input context.
func FromContext(context context.Context) string {
	if context == nil {
		return ""
	}
	deploymentCtxValue := context.Value(deploymentContextKey{})
	if deploymentCtxValue == nil {
		return ""
	}
	return deploymentCtxValue.(*deploymentContextValue).deployment
}
