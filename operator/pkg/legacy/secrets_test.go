package legacy

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/chartutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Test_injector_Enrich(t *testing.T) {
	tests := map[string]struct {
		existingSecrets []string
		commonKey       string
		commonSecrets   []string
		secretMap       map[string][]string
		vals            chartutil.Values
		want            chartutil.Values
		wantErr         error
	}{
		"empty": {},
		"bad vals": {
			existingSecrets: []string{"secret1"},
			commonKey:       "imagePullSecrets",
			commonSecrets:   []string{"secret1"},
			vals: map[string]interface{}{
				"imagePullSecrets": "badger",
			},
			wantErr: fmt.Errorf("key %q maps to a string, table expected", "imagePullSecrets"),
		},
		"bad use existing": {
			existingSecrets: []string{"secret1"},
			commonKey:       "imagePullSecrets",
			commonSecrets:   []string{"secret1"},
			vals: map[string]interface{}{
				"imagePullSecrets": map[string]interface{}{
					"useExisting": "badger",
				},
			},
			wantErr: fmt.Errorf("unexpected value %q type: string", "imagePullSecrets.useExisting"),
		},
		"some secrets": {
			existingSecrets: []string{"secret1", "secret3"},
			commonKey:       "imagePullSecrets",
			commonSecrets:   []string{"secret1", "secret2"},
			secretMap: map[string][]string{
				"mainImagePullSecrets":      {"secret3", "secret4"},
				"collectorImagePullSecrets": {"secret5", "secret6"},
			},
			want: map[string]interface{}{
				"imagePullSecrets": map[string]interface{}{
					"useExisting": []interface{}{"secret1"},
				},
				"mainImagePullSecrets": map[string]interface{}{
					"useExisting": []interface{}{"secret3"},
				},
			},
		},
		"with baseline secrets": {
			existingSecrets: []string{"secret1", "secret3"},
			commonKey:       "imagePullSecrets",
			commonSecrets:   []string{"secret1", "secret2"},
			secretMap: map[string][]string{
				"mainImagePullSecrets":      {"secret3", "secret4"},
				"collectorImagePullSecrets": {"secret5", "secret6"},
			},
			vals: map[string]interface{}{
				"imagePullSecrets": map[string]interface{}{
					"useExisting": []interface{}{"secret01"},
				},
				"mainImagePullSecrets": map[string]interface{}{
					"useExisting": []interface{}{"secret02"},
				},
				"collectorImagePullSecrets": map[string]interface{}{
					"useExisting": []interface{}{"secret03"},
				},
			},
			want: map[string]interface{}{
				"imagePullSecrets": map[string]interface{}{
					"useExisting": []interface{}{"secret01", "secret1"},
				},
				"mainImagePullSecrets": map[string]interface{}{
					"useExisting": []interface{}{"secret02", "secret3"},
				},
				"collectorImagePullSecrets": map[string]interface{}{
					"useExisting": []interface{}{"secret03"},
				},
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			obj := &unstructured.Unstructured{}
			obj.SetNamespace("some-ns")
			var secrets []runtime.Object
			for _, secretName := range tt.existingSecrets {
				secrets = append(secrets, &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "some-ns",
						Name:      secretName,
					},
				})
			}
			i := NewImagePullSecretReferenceInjector(fake.NewFakeClient(secrets...), tt.commonKey, tt.commonSecrets...)
			for key, secrets := range tt.secretMap {
				i = i.WithExtraImagePullSecrets(key, secrets...)
			}
			got, err := i.Enrich(context.Background(), obj, tt.vals)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantErr, err)
		})
	}
}
