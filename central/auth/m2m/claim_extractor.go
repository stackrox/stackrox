package m2m

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/tokens"
)

type claimExtractor interface {
	ExtractRoxClaims(claims map[string][]string) (tokens.RoxClaims, error)
	ExtractClaims(idToken *IDToken) (map[string][]string, error)
}

func newClaimExtractorFromConfig(config *storage.AuthMachineToMachineConfig) claimExtractor {
	switch config.GetType() {
	case storage.AuthMachineToMachineConfig_GITHUB_ACTIONS:
		return &githubClaimExtractor{}
	case storage.AuthMachineToMachineConfig_KUBE_SERVICE_ACCOUNT:
		return &kubeClaimExtractor{}
	default:
		return &genericClaimExtractor{}
	}
}
