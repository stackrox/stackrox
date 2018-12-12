package renderer

import (
	"encoding/base64"
	"fmt"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/utils"
	kubernetesPkg "github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/netutil"
	"github.com/stackrox/rox/pkg/zip"
)

const (
	defaultMonitoringPort = 443
)

func init() {
	Deployers[storage.ClusterType_KUBERNETES_CLUSTER] = newKubernetes()
	Deployers[storage.ClusterType_OPENSHIFT_CLUSTER] = newKubernetes()
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

func (k *kubernetes) renderKubectl(c Config) ([]*zip.File, error) {
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

func (k *kubernetes) Render(c Config) ([]*zip.File, error) {
	// Make all items in SecretsByteMap base64 encoded
	c.SecretsBase64Map = make(map[string]string)
	for k, v := range c.SecretsByteMap {
		c.SecretsBase64Map[k] = base64.StdEncoding.EncodeToString(v)
	}
	if c.ClusterType == storage.ClusterType_KUBERNETES_CLUSTER {
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

	var renderedFiles []*zip.File
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

const instructionPrefix = `To deploy:
  - Unzip the deployment bundle.
  - If you need to add additional trusted CAs, run central/scripts/ca-setup.sh.`

const helmInstructionTemplate = instructionPrefix + `
  {{if not .K8sConfig.MonitoringType.None}}
  - Deploy Monitoring
    - Run monitoring/scripts/setup.sh
    - Run helm install --name monitoring monitoring
  {{- end}}
  - Deploy Central
    - Run central/scripts/setup.sh
    - Run helm install --name central central
  - Deploy Clairify
    - If you want to run the StackRox Clairify scanner, run helm install --name clairify clairify
`

const kubectlInstructionTemplate = instructionPrefix + `{{if not .K8sConfig.MonitoringType.None}}
  - Deploy Monitoring
    - Run monitoring/scripts/setup.sh
    - Run {{.K8sConfig.Command}} create -R -f monitoring
  {{- end}}
  - Deploy Central
    - Run central/scripts/setup.sh
    - Run {{.K8sConfig.Command}} create -R -f central
  - Deploy Clairify
    - If you want to run the StackRox Clairify scanner, run {{.K8sConfig.Command}} create -R -f clairify
`

func (k *kubernetes) Instructions(c Config) string {
	template := kubectlInstructionTemplate
	if c.K8sConfig.DeploymentFormat == v1.DeploymentFormat_HELM {
		template = helmInstructionTemplate
	}

	// If error, then its a programming error
	data, err := executeRawTemplate(template, &c)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func injectImageTags(c *Config) {
	c.K8sConfig.ClairifyImageTag = utils.GenerateImageFromString(c.K8sConfig.ClairifyImage).GetName().GetTag()
	c.K8sConfig.MainImageTag = utils.GenerateImageFromString(c.K8sConfig.MainImage).GetName().GetTag()
}
