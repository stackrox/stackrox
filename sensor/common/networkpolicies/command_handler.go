package networkpolicies

import "github.com/stackrox/rox/generated/internalapi/central"

// CommandHandler handles network policy-related commands.
type CommandHandler interface {
	Start()
	Stop()
	SendCommand(command *central.NetworkPoliciesCommand) bool
	Responses() <-chan *central.NetworkPoliciesResponse
}
