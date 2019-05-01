package renderer

import (
	"encoding/base64"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/image"
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

func (k *kubernetes) renderKubectl(c Config) ([]*zip.File, error) {
	renderedFiles, err := k.renderHelmFiles(c, image.GetCentralChart(), "central")
	if err != nil {
		return nil, errors.Wrap(err, "error rendering central files")
	}

	scannerRenderedFiles, err := k.renderHelmFiles(c, image.GetScannerChart(), "scanner")
	if err != nil {
		return nil, errors.Wrap(err, "error rendering scanner files")
	}
	renderedFiles = append(renderedFiles, scannerRenderedFiles...)

	if c.K8sConfig.Monitoring.Type.OnPrem() {
		monitoringFiles, err := k.renderHelmFiles(c, image.GetMonitoringChart(), "monitoring")
		if err != nil {
			return nil, errors.Wrap(err, "error rendering monitoring files")
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
	if err := injectImageTags(&c); err != nil {
		return nil, err
	}
	monitoringImage, err := generateMonitoringImage(c.K8sConfig.MainImage, c.K8sConfig.MonitoringImage)
	if err != nil {
		return nil, errors.Wrapf(err, "error parsing monitoring image: ")
	}
	c.K8sConfig.Monitoring.Image = monitoringImage
	c.K8sConfig.Monitoring.Endpoint = netutil.WithDefaultPort(c.K8sConfig.Monitoring.Endpoint, defaultMonitoringPort)

	var renderedFiles []*zip.File
	if c.K8sConfig.DeploymentFormat == v1.DeploymentFormat_HELM {
		renderedFiles, err = k.renderHelm(c)
	} else {
		renderedFiles, err = k.renderKubectl(c)
	}
	if err != nil {
		return nil, err
	}
	renderedFiles = append(renderedFiles, dockerAuthFile)
	return wrapFiles(renderedFiles, &c)
}

const instructionPrefix = `To deploy:
  - Unzip the deployment bundle.
  - If you need to add additional trusted CAs, run central/scripts/ca-setup.sh.`

const helmInstructionTemplate = instructionPrefix + `
  {{if not .K8sConfig.Monitoring.Type.None}}
  - Deploy Monitoring
    - Run monitoring/scripts/setup.sh
    - Run helm install --name monitoring monitoring
  {{- end}}
  - Deploy Central
    - Run central/scripts/setup.sh
    - Run helm install --name central central
  - Deploy Scanner
    - If you want to run the StackRox scanner, run helm install --name scanner scanner
`

const kubectlInstructionTemplate = instructionPrefix + `{{if not .K8sConfig.Monitoring.Type.None}}
  - Deploy Monitoring
    - Run monitoring/scripts/setup.sh
    - Run {{.K8sConfig.Command}} create -R -f monitoring
  {{- end}}
  - Deploy Central
    - Run central/scripts/setup.sh
    - Run {{.K8sConfig.Command}} create -R -f central
  - Deploy Scanner
    - If you want to run the StackRox scanner, run {{.K8sConfig.Command}} create -R -f scanner
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

func getTag(imageStr string) (string, error) {
	imageName, err := utils.GenerateImageFromString(imageStr)
	if err != nil {
		return "", err
	}
	return imageName.GetName().GetTag(), nil
}

func injectImageTags(c *Config) error {
	var err error
	c.K8sConfig.ScannerImageTag, err = getTag(c.K8sConfig.ScannerImage)
	if err != nil {
		return err
	}
	c.K8sConfig.MainImageTag, err = getTag(c.K8sConfig.MainImage)
	if err != nil {
		return err
	}
	return nil
}
