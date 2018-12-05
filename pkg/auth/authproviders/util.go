package authproviders

import "github.com/stackrox/rox/generated/api/v1"

// AllUIEndpoints returns all UI endpoints for a given auth provider, with the default UI endpoint first.
func AllUIEndpoints(providerProto *v1.AuthProvider) []string {
	if providerProto.GetUiEndpoint() == "" {
		return nil
	}
	return append([]string{providerProto.GetUiEndpoint()}, providerProto.GetExtraUiEndpoints()...)
}
