package common

import (
	"context"
	"testing"

	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/scale/scheme"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type testCaseResourceNonExistence struct {
	name                  string
	existingObjects       []ctrlClient.Object
	gvk                   schema.GroupVersionKind
	resourceName          string
	expectedAlreadyExists bool
}

const (
	someNamespace = "some-namespace"
	someName      = "some-name"
	otherName     = "other-name"
)

func TestVerifyResourceNonExistence(t *testing.T) {
	testCases := []testCaseResourceNonExistence{
		{
			name:         "central-check-plain",
			gvk:          platform.CentralGVK,
			resourceName: someName,
		},
		{
			name:         "central-check-conflicting",
			gvk:          platform.CentralGVK,
			resourceName: someName,
			existingObjects: []ctrlClient.Object{&platform.Central{
				ObjectMeta: metav1.ObjectMeta{
					Name:      someName,
					Namespace: someNamespace,
				},
			}},
			expectedAlreadyExists: true,
		},
		{
			name:         "central-check-other-name",
			gvk:          platform.CentralGVK,
			resourceName: someName,
			existingObjects: []ctrlClient.Object{&platform.Central{
				ObjectMeta: metav1.ObjectMeta{
					Name:      otherName,
					Namespace: someNamespace,
				},
			}},
		},
		{
			name:         "central-check-other-kind",
			gvk:          platform.CentralGVK,
			resourceName: someName,
			existingObjects: []ctrlClient.Object{&platform.SecuredCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      someName,
					Namespace: someNamespace,
				},
			}},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			executeTestCase(t, testCase)
		})
	}
}

func executeTestCase(t *testing.T, testCase testCaseResourceNonExistence) {
	ctx := context.Background()
	client := buildClientWithObjects(t, testCase.existingObjects)

	resource := gvkToResource(t, testCase.gvk)
	err := VerifyResourceNonExistence(ctx, client, testCase.gvk, resource, someNamespace, testCase.resourceName)
	if testCase.expectedAlreadyExists {
		assertIsAlreadyExistsError(t, err)
	} else {
		assert.NoError(t, err)
	}
}

func assertIsAlreadyExistsError(t *testing.T, err error) {
	assert.Condition(t, func() bool { return apiErrors.IsAlreadyExists(err) }, "expected AlreadyExists error, got: %v", err)
}

func gvkToResource(t *testing.T, gvk schema.GroupVersionKind) string {
	var resource string
	switch gvk.Kind {
	case platform.CentralGVK.Kind:
		resource = "centrals"
	}
	assert.NotEmpty(t, resource, "failed to resolve testCase kind to resource name")
	return resource
}

func buildClientWithObjects(t *testing.T, objects []ctrlClient.Object) ctrlClient.Client {
	sch := runtime.NewScheme()
	require.NoError(t, platform.AddToScheme(sch))
	require.NoError(t, scheme.AddToScheme(sch))
	return fake.NewClientBuilder().
		WithScheme(sch).
		WithObjects(objects...).
		Build()
}
