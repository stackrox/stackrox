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

type tokenVerifier interface {
	VerifyIDToken(ctx context.Context, rawIDToken string) (*IDToken, error)
}

func tokenVerifierFromConfig(ctx context.Context, config *storage.AuthMachineToMachineConfig) (tokenVerifier, error) {
	tlsConfig, err := tlsConfigWithCustomCertPool()
	if err != nil {
		return nil, errors.Wrap(err, "creating TLS config for token verification")
	}
	roundTripper := proxy.RoundTripper(proxy.WithTLSConfig(tlsConfig))
	if config.GetType() == storage.AuthMachineToMachineConfig_KUBE_SERVICE_ACCOUNT {
		// By default k8s requires authentication to fetch the OIDC resources for service account tokens
		// https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#service-account-issuer-discovery
		roundTripper = &authenticatedRoundTripper{roundTripper, readK8sSAToken}
	}
	provider, err := oidc.NewProvider(
		oidc.ClientContext(ctx, &http.Client{Timeout: time.Minute, Transport: roundTripper}),
		config.GetIssuer(),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "creating OIDC provider for issuer %q", config.GetIssuer())
	}

	return &genericTokenVerifier{provider: provider}, nil
}
