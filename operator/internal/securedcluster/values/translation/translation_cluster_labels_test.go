package translation

import (
	"context"
	"testing"

	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/operator/internal/utils/testutils"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestClusterLabels(t *testing.T) {
	newCentral := func(namespace string) *platform.Central {
		return &platform.Central{
			ObjectMeta: metav1.ObjectMeta{Name: "central", Namespace: namespace},
		}
	}

	tests := map[string]struct {
		scNamespace   string
		userLabels    map[string]string
		centralObject *platform.Central
		want          map[string]string
	}{
		"no Central CR anywhere: user labels are returned unchanged": {
			scNamespace: "stackrox",
			userLabels:  map[string]string{"user-label": "value"},
			want:        map[string]string{"user-label": "value"},
		},
		"Central CR in the same namespace: label is added": {
			scNamespace:   "stackrox",
			userLabels:    map[string]string{"user-label": "value"},
			centralObject: newCentral("stackrox"),
			want: map[string]string{
				"user-label":             "value",
				centralColocatedLabelKey: "true",
			},
		},
		"Central CR in a different namespace on the same cluster: label is still added": {
			// SecuredCluster and Central can be installed into different namespaces
			// even when they run on the same physical cluster.
			scNamespace:   "stackrox-secured-cluster",
			centralObject: newCentral("stackrox"),
			want: map[string]string{
				centralColocatedLabelKey: "true",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var objects []ctrlClient.Object
			if tt.centralObject != nil {
				objects = append(objects, tt.centralObject)
			}
			client := testutils.NewFakeClientBuilder(t, objects...).Build()
			translator := New(client, client)

			sc := platform.SecuredCluster{
				ObjectMeta: metav1.ObjectMeta{Namespace: tt.scNamespace},
				Spec:       platform.SecuredClusterSpec{ClusterLabels: tt.userLabels},
			}

			got := translator.clusterLabels(context.Background(), sc)
			assert.Equal(t, tt.want, got)
		})
	}
}
