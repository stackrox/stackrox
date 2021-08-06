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

// Context returns a new context with the scope attached.
func Context(ctx context.Context, deploymentID string) context.Context {
	return context.WithValue(ctx, deploymentContextKey{}, &deploymentContextValue{
		deployment: deploymentID,
	})
}

// IsDeploymentScoped returns a boolean if a distro is set
func IsDeploymentScoped(ctx context.Context) bool {
	return FromContext(ctx) != ""
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
