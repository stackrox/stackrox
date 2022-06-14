package networkpolicies

import (
	"context"

	"github.com/stackrox/stackrox/central/sensor/service/common"
	"github.com/stackrox/stackrox/generated/internalapi/central"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/concurrency"
)

// Controller handles application of network policies in remote clusters.
type Controller interface {
	// ApplyNetworkPolicies takes a network policy modification and applies it. In case of success, the returned network
	// policy modification is the one to undo the changes.
	ApplyNetworkPolicies(ctx context.Context, mod *storage.NetworkPolicyModification) (*storage.NetworkPolicyModification, error)

	ProcessNetworkPoliciesResponse(resp *central.NetworkPoliciesResponse) error
}

// NewController creates and returns a new controller for network policies.
func NewController(injector common.MessageInjector, stopSig concurrency.ReadOnlyErrorSignal) Controller {
	return newController(injector, stopSig)
}
