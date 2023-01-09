package rbac

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func Test_FetchBindingRemovesLastAppliedConfig(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	_, err := fakeClient.RbacV1().ClusterRoleBindings().Create(context.Background(), &v1.ClusterRoleBinding{
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
	}, metav1.CreateOptions{})

	require.NoError(t, err)

	fetcher := newBindingFetcher(fakeClient)
	event, err := fetcher.generateDependentEvent(namespacedBindingID{
		name:      "test-binding",
		namespace: "",
		uid:       "test-uid",
	}, "role-uid", true)

	require.NoError(t, err)

	assert.Equal(t, event.GetBinding().GetRoleId(), "role-uid")
	annotations := event.GetBinding().GetAnnotations()
	assert.Len(t, annotations, 1)
	assert.Equal(t, annotations["SomeKey"], "SomeValue")
}
