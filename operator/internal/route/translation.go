package route

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/stackrox/rox/operator/internal/values/translation"
	"github.com/stackrox/rox/pkg/k8sutil"
	"helm.sh/helm/v3/pkg/chartutil"
	corev1 "k8s.io/api/core/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	tlsSecretCAKey = "ca.pem"
	tlsSecretName  = "central-tls"
)

// NewRouteInjector returns an object which injects the Central certificate authority into route chart values.
func NewRouteInjector(client ctrlClient.Reader, log logr.Logger) *routeInjector {
	return &routeInjector{
		client: client,
		log:    log,
	}
}

type routeInjector struct {
	client ctrlClient.Reader
	log    logr.Logger
}

var _ translation.Enricher = &routeInjector{}

// Enrich injects the Central certificate authority into the reencrypt route.
func (i *routeInjector) Enrich(ctx context.Context, obj k8sutil.Object, vals chartutil.Values) (chartutil.Values, error) {
	namespaceName := obj.GetNamespace()
	tlsSecret := &corev1.Secret{}

	if err := i.client.Get(ctx, ctrlClient.ObjectKey{Name: tlsSecretName, Namespace: namespaceName}, tlsSecret); err != nil {
		return nil, fmt.Errorf("getting secret %s/%s: %w", namespaceName, tlsSecretName, err)
	}

	routeVals := chartutil.Values{
		"central": map[string]interface{}{
			"exposure": map[string]interface{}{
				"route": map[string]interface{}{
					"reencrypt": map[string]interface{}{
						"tls": map[string]interface{}{
							"destinationCACertificate": string(tlsSecret.Data[tlsSecretCAKey]),
						},
					},
				},
			},
		},
	}
	return chartutil.CoalesceTables(vals, routeVals), nil
}
