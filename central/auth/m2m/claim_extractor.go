package m2m

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/tokens"
)

// The claimExtractor interface is unexported and references the IDToken type of this package, hence the mock is
// generated into this package (rather than the mocks subpackage) using -self_package to avoid an import cycle.
// The destination is a _test.go file, so the mock is only compiled as part of the tests and does not leak into
// production code.
//
//go:generate mockgen -package m2m -self_package github.com/stackrox/rox/central/auth/m2m -destination mock_claim_extractor_test.go -source claim_extractor.go
type claimExtractor interface {
	ExtractRoxClaims(claims map[string][]string) (tokens.RoxClaims, error)
	ExtractClaims(idToken *IDToken) (map[string][]string, error)
}

func newClaimExtractorForConfigType(configType storage.AuthMachineToMachineConfig_Type) claimExtractor {
	switch configType {
	case storage.AuthMachineToMachineConfig_GITHUB_ACTIONS:
		return &githubClaimExtractor{}
	case storage.AuthMachineToMachineConfig_KUBE_SERVICE_ACCOUNT:
		return &kubeClaimExtractor{}
	default:
		return &genericClaimExtractor{}
	}
}
