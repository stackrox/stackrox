// Package k8sclient provides the typed Kubernetes client factory.
// Separated from pkg/k8sutil to avoid pulling k8s.io/client-go/kubernetes
// (132 packages) into binaries that only need k8sutil's utilities.
package k8sclient

import (
	"github.com/stackrox/rox/pkg/logging"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var log = logging.LoggerForModule()

// MustCreateK8sClient creates a typed k8s client or panics.
func MustCreateK8sClient(config *rest.Config) kubernetes.Interface {
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Panicf("Creating Kubernetes clientset: %v", err)
	}
	return client
}
