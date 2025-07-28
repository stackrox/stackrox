package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func TestResourceWithNamePredicate(t *testing.T) {
	predicate := &ResourceWithNamePredicate[*corev1.ConfigMap]{
		Name: "test-configmap",
	}

	t.Run("Create", func(t *testing.T) {
		matchingCM := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "test-configmap"}}
		nonMatchingCM := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "other-configmap"}}

		assert.True(t, predicate.Create(event.TypedCreateEvent[*corev1.ConfigMap]{Object: matchingCM}))
		assert.False(t, predicate.Create(event.TypedCreateEvent[*corev1.ConfigMap]{Object: nonMatchingCM}))
	})

	t.Run("Update", func(t *testing.T) {
		oldCM := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "test-configmap", ResourceVersion: "1"}}
		newCM := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "test-configmap", ResourceVersion: "2"}}
		sameCM := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "test-configmap", ResourceVersion: "1"}}
		wrongNameCM := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "other-configmap", ResourceVersion: "2"}}

		assert.True(t, predicate.Update(event.TypedUpdateEvent[*corev1.ConfigMap]{ObjectOld: oldCM, ObjectNew: newCM}))
		assert.False(t, predicate.Update(event.TypedUpdateEvent[*corev1.ConfigMap]{ObjectOld: oldCM, ObjectNew: sameCM}))
		assert.False(t, predicate.Update(event.TypedUpdateEvent[*corev1.ConfigMap]{ObjectOld: oldCM, ObjectNew: wrongNameCM}))
	})

	t.Run("Delete", func(t *testing.T) {
		matchingCM := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "test-configmap"}}
		nonMatchingCM := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "other-configmap"}}

		assert.True(t, predicate.Delete(event.TypedDeleteEvent[*corev1.ConfigMap]{Object: matchingCM}))
		assert.False(t, predicate.Delete(event.TypedDeleteEvent[*corev1.ConfigMap]{Object: nonMatchingCM}))
	})

	t.Run("Generic always false", func(t *testing.T) {
		cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "test-configmap"}}
		assert.False(t, predicate.Generic(event.TypedGenericEvent[*corev1.ConfigMap]{Object: cm}))
	})
}
