package migratetooperator

import (
	"strings"

	"github.com/pkg/errors"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/pkg/pointers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TransformToSecuredCluster detects the configuration from the given source and
// generates a SecuredCluster custom resource. It returns the CR and a list of
// warnings for the caller to emit.
func TransformToSecuredCluster(src Source) (*platform.SecuredCluster, []string, error) {
	clusterName, err := detectClusterName(src)
	if err != nil {
		return nil, nil, errors.Wrap(err, "detecting cluster name")
	}

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

func detectClusterName(src Source) (string, error) {
	secret, err := src.Secret("helm-effective-cluster-name")
	if err != nil {
		return "", errors.Wrap(err, "looking up helm-effective-cluster-name Secret")
	}
	if secret == nil {
		return "", errors.New("Secret \"helm-effective-cluster-name\" not found")
	}
	name := secret.StringData["cluster-name"]
	if name == "" {
		if raw, ok := secret.Data["cluster-name"]; ok {
			name = string(raw)
		}
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("cluster name is empty in Secret \"helm-effective-cluster-name\"")
	}
	return name, nil
}
