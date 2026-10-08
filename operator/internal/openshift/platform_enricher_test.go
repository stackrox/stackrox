package openshift

import (
	"context"
	"errors"
	"testing"

	"github.com/stackrox/rox/operator/internal/utils/testutils"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chartutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestPlatformEnricher(t *testing.T) {
	const namespaceName = "secured-cluster"

	getErr := errors.New("temporary API error")
	tests := map[string]struct {
		annotations map[string]string
		getErr      error
		initialEnv  map[string]interface{}
		wantEnv     map[string]interface{}
		wantErr     error
	}{
		"OpenShift namespace": {
			annotations: map[string]string{
				namespaceUIDRangeAnnotation: "1000670000/10000",
			},
			initialEnv: map[string]interface{}{
				"managedServices": true,
				"openshift":       false,
			},
			wantEnv: map[string]interface{}{
				"managedServices": true,
				"openshift":       true,
			},
		},
		"OpenShift namespace without existing env values": {
			annotations: map[string]string{
				namespaceUIDRangeAnnotation: "1000670000/10000",
			},
			wantEnv: map[string]interface{}{"openshift": true},
		},
		"Kubernetes namespace": {
			annotations: map[string]string{"example.com/annotation": "value"},
			initialEnv: map[string]interface{}{
				"managedServices": true,
				"openshift":       false,
			},
			wantEnv: map[string]interface{}{
				"managedServices": true,
				"openshift":       false,
			},
		},
		"empty UID range annotation": {
			annotations: map[string]string{namespaceUIDRangeAnnotation: ""},
			initialEnv:  map[string]interface{}{"openshift": false},
			wantEnv:     map[string]interface{}{"openshift": false},
		},
		"namespace read error": {
			getErr:  getErr,
			wantErr: getErr,
		},
	}

	obj := &unstructured.Unstructured{}
	obj.SetNamespace(namespaceName)

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			namespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:        namespaceName,
					Annotations: tt.annotations,
				},
			}
			builder := testutils.NewFakeClientBuilder(t, namespace)
			if tt.getErr != nil {
				builder = builder.WithInterceptorFuncs(interceptor.Funcs{
					Get: func(_ context.Context, _ ctrlClient.WithWatch, _ ctrlClient.ObjectKey, _ ctrlClient.Object, _ ...ctrlClient.GetOption) error {
						return tt.getErr
					},
				})
			}

			vals := chartutil.Values{}
			if tt.initialEnv != nil {
				vals["env"] = tt.initialEnv
			}
			result, err := NewEnricher(builder.Build()).Enrich(context.Background(), k8sutil.Object(obj), vals)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, result)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantEnv, result["env"])
		})
	}
}
