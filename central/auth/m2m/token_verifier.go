package m2m

import (
	"context"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/httputil/proxy"
)

type IDToken struct {
	Claims   func(any) error
	Subject  string
	Audience []string
}

// The tokenVerifier interface is unexported and references the IDToken type of this package, hence the mock is
// generated into this package (rather than the mocks subpackage) using -self_package to avoid an import cycle.
// The destination is a _test.go file, so the mock is only compiled as part of the tests and does not leak into
// production code.
//
//go:generate mockgen -package m2m -self_package github.com/stackrox/rox/central/auth/m2m -destination mock_token_verifier_test.go -source token_verifier.go
type tokenVerifier interface {
	Audience() string
	VerifyIDToken(ctx context.Context, rawIDToken string) (*IDToken, error)
}

func tokenVerifierFromConfigData(
	ctx context.Context,
	configType storage.AuthMachineToMachineConfig_Type,
	issuer string,
	audience string,
) (tokenVerifier, error) {
	tlsConfig, err := tlsConfigWithCustomCertPool()
	if err != nil {
		return nil, errors.Wrap(err, "creating TLS config for token verification")
	}
	roundTripper := proxy.RoundTripper(proxy.WithTLSConfig(tlsConfig))
	if configType == storage.AuthMachineToMachineConfig_KUBE_SERVICE_ACCOUNT {
		// By default k8s requires authentication to fetch the OIDC resources for service account tokens
		// https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#service-account-issuer-discovery
		roundTripper = &authenticatedRoundTripper{roundTripper, readK8sSAToken}
	}
	provider, err := oidc.NewProvider(
		oidc.ClientContext(ctx, &http.Client{Timeout: time.Minute, Transport: roundTripper}),
		issuer,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "creating OIDC provider for issuer %q", issuer)
	}

	return &genericTokenVerifier{provider: provider, audience: audience}, nil
}
