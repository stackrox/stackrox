package translation

import (
	"context"
	"fmt"

	"github.com/operator-framework/helm-operator-plugins/pkg/values"
	"github.com/stackrox/rox/pkg/k8sutil"
	"helm.sh/helm/v3/pkg/chartutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Enricher reads helm values produced for an object by a Translator or another enricher, and returns new values.
// Note: mutates provided vals.
type Enricher interface {
	Enrich(ctx context.Context, obj k8sutil.Object, vals chartutil.Values) (chartutil.Values, error)
}

// WithEnrichment chains a given translator with zero or more enrichers.
func WithEnrichment(translator values.Translator, enrichers ...Enricher) values.Translator {
	return values.TranslatorFunc(func(ctx context.Context, unstructured *unstructured.Unstructured) (chartutil.Values, error) {
		values, err := translator.Translate(ctx, unstructured)
		if err != nil {
			return nil, err
		}
		for i, enricher := range enrichers {
			enriched, err2 := enricher.Enrich(ctx, unstructured, values)
			if err2 != nil {
				return nil, fmt.Errorf("helm values enricher with index %d (%T) failed: %w", i, enricher, err2)
			}
			values = enriched
		}
		return values, nil
	})
}
