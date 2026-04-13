package rbac

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"
)

var (
	roleUID            = "role-uid"
	clusterRoleBinding = &v1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-binding",
			UID:  "test-uid",
			Annotations: map[string]string{
				"SomeKey": "SomeValue",
				"kubectl.kubernetes.io/last-applied-configuration": "{\"some_prop\": \"value\"}",
			},
		},
		Subjects: nil,
		RoleRef: v1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "test-role",
		},
	}
	roleBinding = &v1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ns-binding",
			UID:       "test--ns-uid",
			Namespace: "test-ns",
			Annotations: map[string]string{
				"SomeKey": "SomeValue",
				"kubectl.kubernetes.io/last-applied-configuration": "{\"some_prop\": \"value\"}",
			},
		},
		Subjects: nil,
		RoleRef: v1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "test-role",
		},
	}

	clusterRoleBindingID = namespacedBindingID{
		name:      "test-binding",
		namespace: "",
		uid:       "test-uid",
	}

	roleBindingID = namespacedBindingID{
		name:      "test-ns-binding",
		namespace: "test-ns",
		uid:       "test-ns-uid",
	}
)

func Test_FetchBindingRemovesLastAppliedConfig(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1.AddToScheme(scheme)
	dynClient := dynamicfake.NewSimpleDynamicClient(scheme)
	fetcher := newBindingFetcher(dynClient)

	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(clusterRoleBinding)
	require.NoError(t, err)
	unstructuredCRB := &unstructured.Unstructured{Object: unstructuredObj}
	_, err = dynClient.Resource(client.ClusterRoleBindingGVR).Create(context.Background(), unstructuredCRB, metav1.CreateOptions{})
	require.NoError(t, err)

	event, err := fetcher.generateDependentEvent(clusterRoleBindingID, roleUID, true)
	require.NoError(t, err)

	assert.Equal(t, event.GetBinding().GetRoleId(), roleUID)
	annotations := event.GetBinding().GetAnnotations()
	require.Len(t, annotations, 1)
	assert.Equal(t, annotations["SomeKey"], "SomeValue")

	unstructuredObj, err = runtime.DefaultUnstructuredConverter.ToUnstructured(roleBinding)
	require.NoError(t, err)
	unstructuredRB := &unstructured.Unstructured{Object: unstructuredObj}
	_, err = dynClient.Resource(client.RoleBindingGVR).Namespace("test-ns").Create(context.Background(), unstructuredRB, metav1.CreateOptions{})
	require.NoError(t, err)

	event, err = fetcher.generateDependentEvent(roleBindingID, roleUID, true)
	require.NoError(t, err)

	assert.Equal(t, event.GetBinding().GetRoleId(), roleUID)
	annotations = event.GetBinding().GetAnnotations()
	require.Len(t, annotations, 1)
	assert.Equal(t, annotations["SomeKey"], "SomeValue")
}

func Test_FetchBindingErrors(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1.AddToScheme(scheme)
	fakeDynClient := dynamicfake.NewSimpleDynamicClient(scheme)
	fetcher := newBindingFetcher(fakeDynClient)
	fetcher.numRetries = 2

	t.Run("ClusterRoleBinding fails once", func(tt *testing.T) {
		reactor := newPretendReactor(clusterRoleBinding, 1, "get", "clusterrolebindings")
		fakeDynClient.PrependReactor(reactor.verb, reactor.resource, reactor.react)
		event, err := fetcher.generateDependentEvent(clusterRoleBindingID, roleUID, true)
		require.NoError(tt, err)

		assert.Equal(tt, event.GetBinding().GetRoleId(), roleUID)
		annotations := event.GetBinding().GetAnnotations()
		require.Len(tt, annotations, 1)
		assert.Equal(tt, annotations["SomeKey"], "SomeValue")
	})

	t.Run("ClusterRoleBinding fails twice", func(tt *testing.T) {
		reactor := newPretendReactor(clusterRoleBinding, 2, "get", "clusterrolebindings")
		fakeDynClient.PrependReactor(reactor.verb, reactor.resource, reactor.react)
		event, err := fetcher.generateDependentEvent(clusterRoleBindingID, roleUID, true)
		require.Error(tt, err)

		assert.Nil(tt, event)
	})

	t.Run("RoleBinding fails once", func(tt *testing.T) {
		reactor := newPretendReactor(roleBinding, 1, "get", "rolebindings")
		fakeDynClient.PrependReactor(reactor.verb, reactor.resource, reactor.react)
		event, err := fetcher.generateDependentEvent(roleBindingID, roleUID, false)
		require.NoError(tt, err)

		assert.Equal(tt, event.GetBinding().GetRoleId(), roleUID)
		annotations := event.GetBinding().GetAnnotations()
		require.Len(tt, annotations, 1)
		assert.Equal(tt, annotations["SomeKey"], "SomeValue")
	})

	t.Run("RoleBinding fails twice", func(tt *testing.T) {
		reactor := newPretendReactor(roleBinding, 2, "get", "rolebindings")
		fakeDynClient.PrependReactor(reactor.verb, reactor.resource, reactor.react)
		event, err := fetcher.generateDependentEvent(roleBindingID, roleUID, false)
		require.Error(tt, err)

		assert.Nil(tt, event)
	})
}

type pretendReactor struct {
	verb              string
	resource          string
	numErrors         int
	currentCallNumber int
	obj               runtime.Object
}

func (pr *pretendReactor) react(_ k8stesting.Action) (bool, runtime.Object, error) {
	defer func() {
		pr.currentCallNumber++
	}()
	if pr.currentCallNumber >= pr.numErrors {
		return true, pr.obj, nil
	}
	return true, pr.obj, errors.New("some error")
}

func newPretendReactor(obj runtime.Object, numErrors int, verb, resource string) *pretendReactor {
	return &pretendReactor{
		verb:              verb,
		resource:          resource,
		numErrors:         numErrors,
		currentCallNumber: 0,
		obj:               obj,
	}
}
