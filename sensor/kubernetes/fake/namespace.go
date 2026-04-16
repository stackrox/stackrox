package fake

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	namespacePool                 = newPool()
	namespacesWithDeploymentsPool = newPool()
)

func getNamespace(name, id string) *corev1.Namespace {
	return &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			UID:  idOrNewUID(id),
			CreationTimestamp: metav1.Time{
				Time: time.Now(),
			},
			Labels:      createRandMap(16, 3),
			Annotations: createRandMap(16, 3),
		},
		Status: corev1.NamespaceStatus{},
	}
}

func getNamespaces(numNamespaces int, ids []string) []*corev1.Namespace {
	namespaces := make([]*corev1.Namespace, 0, numNamespaces)

	namespaces = append(namespaces, getNamespace("default", getID(ids, 0)))
	namespacePool.add("default")
	for i := 0; i < numNamespaces-1; i++ {
		name := randStringWithLength(16)
		namespacePool.add(name)
		namespaces = append(namespaces, getNamespace(name, getID(ids, i+1)))
	}
	return namespaces
}
