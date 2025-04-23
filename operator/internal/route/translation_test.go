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
		enabled     bool
		destCAValue string
		want        string
	}{
		"should default to central CA from central-tls secret": {
			enabled:     true,
			destCAValue: "",
			want:        centralCA,
		},
		"should take destinationCACertificate from the input values": {
			enabled:     true,
			destCAValue: "fake-input-CA",
			want:        "fake-input-CA",
		},
		"should do nothing if not enabled": {
			enabled:     false,
			destCAValue: "",
			want:        "",
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
			vals := chartutil.Values{
				"central": map[string]interface{}{
					"exposure": map[string]interface{}{
						"route": map[string]interface{}{
							"reencrypt": map[string]interface{}{
								"enabled": tt.enabled,
								"tls":     map[string]interface{}{},
							},
						},
					},
				},
			}
			if tt.destCAValue != "" {
				tlsVars, err := vals.Table("central.exposure.route.reencrypt.tls")
				require.NoError(t, err)
				tlsVars["destinationCACertificate"] = tt.destCAValue
			}

			gotValues, err := i.Enrich(context.Background(), obj, vals)
			require.NoError(t, err)
			gotCA, err := gotValues.PathValue("central.exposure.route.reencrypt.tls.destinationCACertificate")
			if tt.enabled {
				require.NoError(t, err)
				assert.Equal(t, tt.want, gotCA)
			} else {
				require.Error(t, err)
			}
		})
	}
}
