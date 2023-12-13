package translation

import (
	"context"
	"fmt"
	"testing"

	"github.com/operator-framework/helm-operator-plugins/pkg/values"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chartutil"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type enricher struct {
	key string
	val string
	err error
}

func (e enricher) Enrich(_ context.Context, _ k8sutil.Object, vals chartutil.Values) (chartutil.Values, error) {
	vals[e.key] = e.val
	return vals, e.err
}

func TestWithEnrichment(t *testing.T) {
	rootCause := fmt.Errorf("%s", "boom") // silence "no variadic arguments"
	tests := map[string]struct {
		translator values.Translator
		enrichers  []Enricher
		want       chartutil.Values
		wantErr    error
	}{
		"translator error": {
			translator: values.TranslatorFunc(func(_ context.Context, obj *unstructured.Unstructured) (chartutil.Values, error) {
				return nil, rootCause
			}),
			wantErr: rootCause,
		},
		"no enrichment": {
			translator: values.TranslatorFunc(func(_ context.Context, obj *unstructured.Unstructured) (chartutil.Values, error) {
				return map[string]interface{}{"foo": "bar"}, nil
			}),
			want: map[string]interface{}{"foo": "bar"},
		},
		"with enrichment": {
			translator: values.TranslatorFunc(func(_ context.Context, obj *unstructured.Unstructured) (chartutil.Values, error) {
				return map[string]interface{}{"foo": "val1", "bar": "val2"}, nil
			}),
			enrichers: []Enricher{
				enricher{key: "bar", val: "val3"},
				enricher{key: "baz", val: "val4"},
			},
			want: map[string]interface{}{"foo": "val1", "bar": "val3", "baz": "val4"},
		},
		"enrichment error": {
			translator: values.TranslatorFunc(func(_ context.Context, obj *unstructured.Unstructured) (chartutil.Values, error) {
				return map[string]interface{}{"foo": "val1", "bar": "val2"}, nil
			}),
			enrichers: []Enricher{
				enricher{key: "bar", val: "val3"},
				enricher{err: rootCause},
			},
			wantErr: fmt.Errorf("helm values enricher with index 1 (translation.enricher) failed: %w", rootCause),
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			wrapped := WithEnrichment(tt.translator, tt.enrichers...)
			vals, err := wrapped.Translate(context.Background(), nil)
			require.Equal(t, tt.want, vals)
			require.Equal(t, tt.wantErr, err)
		})
	}
}
