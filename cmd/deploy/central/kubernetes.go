package central

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/images/utils"
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
	injectImageTags(&c)

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

func injectImageTags(c *Config) {
	c.K8sConfig.ClairifyImageTag = utils.GenerateImageFromString(c.K8sConfig.ClairifyImage).GetName().GetTag()
	c.K8sConfig.PreventImageTag = utils.GenerateImageFromString(c.K8sConfig.PreventImage).GetName().GetTag()

}
