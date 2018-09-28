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
		"kubernetes/ca-setup.sh",
		"kubernetes/central.sh",
		"kubernetes/central.yaml",
		"kubernetes/clairify.sh",
		"kubernetes/clairify.yaml",
		"kubernetes/delete-ca.sh",
		"kubernetes/lb.yaml",
		"kubernetes/np.yaml",
		"kubernetes/port-forward.sh",
	}

	return renderFilenames(filenames, c)
}
func (k *kubernetes) Instructions() string {
	return `To deploy:
  1. Unzip the deployment bundle.
  2. If you need to add additional trusted CAs, run ca-setup.sh.
  3. Run central.sh.
  4. If you want to run the StackRox Clairify scanner, run clairify.sh.
  5. Expose Central:
       a. Using a LoadBalancer: kubectl create -f lb.yaml
       b. Using a NodePort:     kubectl create -f np.yaml
       c. Using a port forward: ./port-forward.sh 8443`
}

func injectImageTags(c *Config) {
	c.K8sConfig.ClairifyImageTag = utils.GenerateImageFromString(c.K8sConfig.ClairifyImage).GetName().GetTag()
	c.K8sConfig.PreventImageTag = utils.GenerateImageFromString(c.K8sConfig.PreventImage).GetName().GetTag()
}
