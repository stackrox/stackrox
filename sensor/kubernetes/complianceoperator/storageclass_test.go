package complianceoperator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestHasDefaultStorageClass(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		objects       []runtime.Object
		expectDefault bool
	}{
		"default storage class exists": {
			objects: []runtime.Object{
				&storagev1.StorageClass{
					ObjectMeta: metav1.ObjectMeta{
						Name: "gp3-csi",
						Annotations: map[string]string{
							defaultStorageClassAnnotationKey: "true",
						},
					},
				},
				&storagev1.StorageClass{
					ObjectMeta: metav1.ObjectMeta{
						Name: "gp2-csi",
						Annotations: map[string]string{
							defaultStorageClassAnnotationKey: "false",
						},
					},
				},
			},
			expectDefault: true,
		},
		"no default storage class": {
			objects: []runtime.Object{
				&storagev1.StorageClass{
					ObjectMeta: metav1.ObjectMeta{
						Name: "gp3-csi",
						Annotations: map[string]string{
							defaultStorageClassAnnotationKey: "false",
						},
					},
				},
			},
			expectDefault: false,
		},
		"no storage classes": {
			expectDefault: false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := fake.NewSimpleDynamicClient(scheme.Scheme, tc.objects...)

			hasDefault, err := hasDefaultStorageClass(context.Background(), client)
			require.NoError(t, err)
			assert.Equal(t, tc.expectDefault, hasDefault)
		})
	}
}

func TestValidateComplianceScanStorage(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		objects     []runtime.Object
		expectError bool
	}{
		"allows schedule when default storage class exists": {
			objects: []runtime.Object{
				&storagev1.StorageClass{
					ObjectMeta: metav1.ObjectMeta{
						Name: "default-sc",
						Annotations: map[string]string{
							defaultStorageClassAnnotationKey: "true",
						},
					},
				},
			},
		},
		"rejects schedule when no default storage class exists": {
			objects: []runtime.Object{
				&storagev1.StorageClass{
					ObjectMeta: metav1.ObjectMeta{
						Name: "gp3-csi",
					},
				},
			},
			expectError: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := fake.NewSimpleDynamicClient(scheme.Scheme, tc.objects...)
			handler := &handlerImpl{client: client}

			err := handler.validateComplianceScanStorage(context.Background())
			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "No default storage class")
				return
			}
			require.NoError(t, err)
		})
	}
}
