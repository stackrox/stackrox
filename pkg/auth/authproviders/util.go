package authproviders

import (
	"github.com/stackrox/rox/generated/storage"
)

// AllUIEndpoints returns all UI endpoints for a given auth provider, with the default UI endpoint first.
func AllUIEndpoints(providerProto *storage.AuthProvider) []string {
	if providerProto.GetUiEndpoint() == "" {
		return nil
	}
	return append([]string{providerProto.GetUiEndpoint()}, providerProto.GetExtraUiEndpoints()...)
}
