package central

import (
	"encoding/base64"
	"fmt"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/images/utils"
	kubernetesPkg "github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/netutil"
)

const (
	defaultMonitoringPort = 443
)

func init() {
	Deployers[v1.ClusterType_KUBERNETES_CLUSTER] = newKubernetes()
	Deployers[v1.ClusterType_OPENSHIFT_CLUSTER] = newKubernetes()

}

type kubernetes struct{}

func newKubernetes() deployer {
	return &kubernetes{}
}

const (
	prefixPath            = "/data/templates/"
	monitoringChartSuffix = "kubernetes/helm/monitoringchart"
	centralChartSuffix    = "kubernetes/helm/centralchart"
	clairifyChartSuffix   = "kubernetes/helm/clairifychart"
)

var (
	monitoringChartPath = prefixPath + monitoringChartSuffix
	centralChartPath    = prefixPath + centralChartSuffix
	clairifyChartPath   = prefixPath + clairifyChartSuffix
)

func (k *kubernetes) renderKubectl(c Config) ([]*v1.File, error) {
	renderedFiles, err := k.renderHelmFiles(c, centralChartPath, "central")
	if err != nil {
		return nil, fmt.Errorf("error rendering central files: %v", err)
	}

	clairifyRenderedFiles, err := k.renderHelmFiles(c, clairifyChartPath, "clairify")
	if err != nil {
		return nil, fmt.Errorf("error rendering clairify files: %v", err)
	}
	renderedFiles = append(renderedFiles, clairifyRenderedFiles...)

	if c.K8sConfig.MonitoringType.OnPrem() {
		monitoringFiles, err := k.renderHelmFiles(c, monitoringChartPath, "monitoring")
		if err != nil {
			return nil, fmt.Errorf("error rendering monitoring files: %v", err)
		}
		renderedFiles = append(renderedFiles, monitoringFiles...)
	}
	return renderedFiles, nil
}

func (k *kubernetes) Render(c Config) ([]*v1.File, error) {
	// Make all items in SecretsByteMap base64 encoded
	c.SecretsBase64Map = make(map[string]string)
	for k, v := range c.SecretsByteMap {
		c.SecretsBase64Map[k] = base64.StdEncoding.EncodeToString(v)
	}
	if c.ClusterType == v1.ClusterType_KUBERNETES_CLUSTER {
		c.K8sConfig.Command = "kubectl"
	} else {
		c.K8sConfig.Command = "oc"
	}

	var err error
	c.K8sConfig.Registry, err = kubernetesPkg.GetResolvedRegistry(c.K8sConfig.MainImage)
	if err != nil {
		return nil, err
	}
	injectImageTags(&c)
	c.K8sConfig.MonitoringImage = generateMonitoringImage(c.K8sConfig.MainImage)
	c.K8sConfig.MonitoringEndpoint = netutil.WithDefaultPort(c.K8sConfig.MonitoringEndpoint, defaultMonitoringPort)

	var renderedFiles []*v1.File
	if c.K8sConfig.DeploymentFormat == v1.DeploymentFormat_HELM {
		renderedFiles, err = k.renderHelm(c)
	} else {
		renderedFiles, err = k.renderKubectl(c)
	}
	if err != nil {
		return nil, err
	}
	return wrapFiles(renderedFiles, &c, dockerAuthPath)
}

const instructionTemplate = `To deploy:
  - Unzip the deployment bundle.
  - If you need to add additional trusted CAs, run central/scripts/ca-setup.sh.
  {{- if not .K8sConfig.MonitoringType.None}}
  - Deploy Monitoring
    - Run monitoring/scripts/setup.sh
  {{- if eq .K8sConfig.DeploymentFormat.String "KUBECTL"}}
    - Run {{.K8sConfig.Command}} create -R -f monitoring
  {{- else}}
    - Run helm install --name monitoring monitoring
  {{- end}}
  {{- end}}
  - Deploy Central
    - Run central/scripts/setup.sh
  {{- if eq .K8sConfig.DeploymentFormat.String "KUBECTL"}}
    - Run {{.K8sConfig.Command}} create -R -f central
  {{- else}}
    - Run helm install --name central central
  {{- end}}
  - Deploy Clairify
  {{- if eq .K8sConfig.DeploymentFormat.String "KUBECTL"}}
    - If you want to run the StackRox Clairify scanner, run {{.K8sConfig.Command}} create -R -f clairify
  {{- else}}
    - Run helm install --name clairify clairify
  {{- end}}
`

func (k *kubernetes) Instructions(c Config) string {
	// If error, then its a programming error
	data, err := executeRawTemplate(instructionTemplate, &c)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func injectImageTags(c *Config) {
	c.K8sConfig.ClairifyImageTag = utils.GenerateImageFromString(c.K8sConfig.ClairifyImage).GetName().GetTag()
	c.K8sConfig.MainImageTag = utils.GenerateImageFromString(c.K8sConfig.MainImage).GetName().GetTag()
}
