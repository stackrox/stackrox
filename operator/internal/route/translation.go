package route

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/stackrox/rox/operator/internal/common"
	"github.com/stackrox/rox/operator/internal/utils"
	"github.com/stackrox/rox/operator/internal/values/translation"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/mtls"
	"helm.sh/helm/v3/pkg/chartutil"
	corev1 "k8s.io/api/core/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// NewRouteInjector returns an object which injects the Central certificate authority into route chart values.
// It takes a context and controller client.
func NewRouteInjector(client ctrlClient.Client, direct ctrlClient.Reader, log logr.Logger) *routeInjector {
	return &routeInjector{
		client: client,
		direct: direct,
		log:    log,
	}
}

type routeInjector struct {
	client ctrlClient.Client
	direct ctrlClient.Reader
	log    logr.Logger
}

var _ translation.Enricher = &routeInjector{}

// Enrich injects the Central certificate authority into the reencrypt route.
func (i *routeInjector) Enrich(ctx context.Context, obj k8sutil.Object, vals chartutil.Values) (chartutil.Values, error) {
	destCAPath := "central.exposure.route.reencrypt.tls.destinationCACertificate"
	if destCA, err := vals.PathValue(destCAPath); destCA != "" && err == nil {
		return vals, nil
	}

	namespaceName := obj.GetNamespace()
	tlsSecret := &corev1.Secret{}
	key := ctrlClient.ObjectKey{Name: common.TLSSecretName, Namespace: namespaceName}
	if err := utils.GetWithFallbackToUncached(ctx, i.client, i.direct, key, tlsSecret); err != nil {
		return nil, fmt.Errorf("getting secret %s/%s: %w", namespaceName, common.TLSSecretName, err)
	}

	routeVals := chartutil.Values{
		"central": map[string]interface{}{
			"exposure": map[string]interface{}{
				"route": map[string]interface{}{
					"reencrypt": map[string]interface{}{
						"tls": map[string]interface{}{
							"destinationCACertificate": string(tlsSecret.Data[mtls.CACertFileName]),
						},
					},
				},
			},
		},
	}
	return chartutil.CoalesceTables(vals, routeVals), nil
}
