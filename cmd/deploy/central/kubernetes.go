package central

import (
	"github.com/stackrox/rox/generated/api/v1"
	kubernetesPkg "github.com/stackrox/rox/pkg/kubernetes"
)

func init() {
	Deployers[v1.ClusterType_KUBERNETES_CLUSTER] = newKubernetes()
}

type kubernetes struct {
}

func newKubernetes() deployer {
	return &kubernetes{}
}

func (k *kubernetes) Render(c Config) ([]*v1.File, error) {
	var err error
	c.K8sConfig.Registry, err = kubernetesPkg.GetResolvedRegistry(c.K8sConfig.PreventImage)
	if err != nil {
		return nil, err
	}

	filenames := []string{
		"kubernetes/central.sh",
		"kubernetes/central.yaml",
		"kubernetes/clairify.sh",
		"kubernetes/clairify.yaml",
		"kubernetes/lb.yaml",
		"kubernetes/port-forward.sh",
	}

	return renderFilenames(filenames, c)
}
