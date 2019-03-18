package networkpolicies

import (
	"context"

	"github.com/stackrox/rox/central/sensor/service/common"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
)

// Controller handles application of network policies in remote clusters.
type Controller interface {
	ApplyNetworkPolicies(ctx context.Context, mod *v1.NetworkPolicyModification) error
	ProcessNetworkPoliciesResponse(resp *central.NetworkPoliciesResponse) error
}

// NewController creates and returns a new controller for network policies.
func NewController(injector common.MessageInjector, stopSig concurrency.ReadOnlyErrorSignal) Controller {
	return newController(injector, stopSig)
}
