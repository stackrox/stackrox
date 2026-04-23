package migratetooperator

import (
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/pkg/pointers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TransformToSecuredCluster detects the configuration from the given source and
// generates a SecuredCluster custom resource. It returns the CR and a list of
// warnings for the caller to emit.
func TransformToSecuredCluster(src Source, clusterName string) (*platform.SecuredCluster, []string, error) {
	cr := &platform.SecuredCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "platform.stackrox.io/v1alpha1",
			Kind:       "SecuredCluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "stackrox-secured-cluster-services",
		},
		Spec: platform.SecuredClusterSpec{
			ClusterName: pointers.String(clusterName),
		},
	}
	return cr, nil, nil
}
