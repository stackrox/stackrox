package translation

import (
	"context"
	"testing"

	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/utils/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestClusterLabels(t *testing.T) {
	tests := map[string]struct {
		userLabels       map[string]string
		centralNamespace string
	}{
		"no labels": {},
		"empty labels": {
			userLabels: map[string]string{},
		},
		"user labels are preserved": {
			userLabels: map[string]string{"user-label": "value", "another-label": "another-value"},
		},
		"explicit central-colocated label is preserved without Central": {
			userLabels: map[string]string{
				"user-label":                    "value",
				"stackrox.io/central-colocated": "true",
			},
		},
		"explicit central-colocated label is preserved with Central": {
			userLabels: map[string]string{
				"user-label":                    "value",
				"stackrox.io/central-colocated": "false",
			},
			centralNamespace: "stackrox",
		},
		"Central in the same namespace does not add labels": {
			centralNamespace: "stackrox",
		},
		"Central in another namespace does not add labels": {
			userLabels:       map[string]string{"user-label": "value"},
			centralNamespace: "stackrox-central",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			objects := defaultObjectsWithNamespace("stackrox")
			if tt.centralNamespace != "" {
				objects = append(objects, &platform.Central{
					ObjectMeta: metav1.ObjectMeta{Name: "central", Namespace: tt.centralNamespace},
				})
			}
			centralLookups := 0
			client := testutils.NewFakeClientBuilder(t, objects...).WithInterceptorFuncs(interceptor.Funcs{
				List: func(ctx context.Context, client ctrlClient.WithWatch, list ctrlClient.ObjectList, opts ...ctrlClient.ListOption) error {
					if _, ok := list.(*platform.CentralList); ok {
						centralLookups++
					}
					return client.List(ctx, list, opts...)
				},
			}).Build()
			translator := New(client, client)

			sc := platform.SecuredCluster{
				ObjectMeta: metav1.ObjectMeta{Namespace: "stackrox"},
				Spec:       platform.SecuredClusterSpec{ClusterLabels: tt.userLabels},
			}
			original := sc.DeepCopy()

			got, err := translator.translate(context.Background(), sc)
			require.NoError(t, err)
			if len(tt.userLabels) == 0 {
				assert.NotContains(t, got, "clusterLabels")
			} else {
				want := make(map[string]interface{}, len(tt.userLabels))
				for key, value := range tt.userLabels {
					want[key] = value
				}
				assert.Equal(t, want, got["clusterLabels"])
			}
			assert.Zero(t, centralLookups)
			assert.Equal(t, original.Spec.ClusterLabels, sc.Spec.ClusterLabels)
		})
	}
}
