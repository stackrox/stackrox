package route

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stackrox/rox/operator/internal/common"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chartutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_injector_Enrich(t *testing.T) {
	centralCA := "fake-central-CA"

	tests := map[string]struct {
		destCAValue string
		want        string
	}{
		"should default to central CA from central-tls secret": {
			destCAValue: "",
			want:        centralCA,
		},
		"should take destinationCACertificate from the input values": {
			destCAValue: "fake-input-CA",
			want:        "fake-input-CA",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			obj := &unstructured.Unstructured{}
			obj.SetNamespace("some-ns")
			tlsSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "some-ns",
					Name:      common.CentralTLSSecretName,
				},
				Data: map[string][]byte{mtls.CACertFileName: []byte(centralCA)},
			}
			i := NewRouteInjector(fake.NewFakeClient(tlsSecret), fake.NewFakeClient(tlsSecret), logr.New(nil))
			vals := chartutil.Values{}
			if tt.destCAValue != "" {
				vals["central"] = map[string]interface{}{
					"exposure": map[string]interface{}{
						"route": map[string]interface{}{
							"reencrypt": map[string]interface{}{
								"tls": map[string]interface{}{
									"destinationCACertificate": string(tt.destCAValue),
								},
							},
						},
					},
				}
			}

			gotValues, err := i.Enrich(context.Background(), obj, vals)
			require.NoError(t, err)
			gotCA, err := gotValues.PathValue("central.exposure.route.reencrypt.tls.destinationCACertificate")
			require.NoError(t, err)

			assert.Equal(t, tt.want, gotCA)
		})
	}
}
