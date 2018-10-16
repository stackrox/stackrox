package central

import (
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/images/utils"
	kubernetesPkg "github.com/stackrox/rox/pkg/kubernetes"
)

func init() {
	Deployers[v1.ClusterType_KUBERNETES_CLUSTER] = newKubernetes()
}

type kubernetes struct{}

func newKubernetes() deployer {
	return &kubernetes{}
}

var monitoringOnPrem = []string{
	"kubernetes/monitoring/monitoring.sh",
	"kubernetes/monitoring/monitoring.yaml",
	"kubernetes/monitoring/influxdb.conf",
}

var monitoringClient = []string{
	"kubernetes/telegraf.conf",
}

func (k *kubernetes) Render(c Config) ([]*v1.File, error) {
	var err error
	c.K8sConfig.Registry, err = kubernetesPkg.GetResolvedRegistry(c.K8sConfig.PreventImage)
	if err != nil {
		return nil, err
	}
	injectImageTags(&c)
	c.K8sConfig.MonitoringImage = generateMonitoringImage(c.K8sConfig.PreventImage)

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

	if c.K8sConfig.MonitoringType.OnPrem() {
		filenames = append(filenames, monitoringOnPrem...)
		filenames = append(filenames, monitoringClient...)
	} else if c.K8sConfig.MonitoringType.StackRoxHosted() {
		filenames = append(filenames, monitoringClient...)
	}

	return renderFilenames(filenames, &c, "/data/assets/docker-auth.sh")
}

func (k *kubernetes) Instructions() string {
	return `To deploy:
  1. Unzip the deployment bundle.
  2. If you need to add additional trusted CAs, run ca-setup.sh.
  3. If you have opted into self-hosting monitoring, run monitoring/monitoring.sh
  4. Run central.sh.
  5. If you want to run the StackRox Clairify scanner, run clairify.sh.
  6. Expose Central:
       a. Using a LoadBalancer: kubectl create -f lb.yaml
       b. Using a NodePort:     kubectl create -f np.yaml
       c. Using a port forward: ./port-forward.sh 8443`
}

func injectImageTags(c *Config) {
	c.K8sConfig.ClairifyImageTag = utils.GenerateImageFromString(c.K8sConfig.ClairifyImage).GetName().GetTag()
	c.K8sConfig.PreventImageTag = utils.GenerateImageFromString(c.K8sConfig.PreventImage).GetName().GetTag()
}
