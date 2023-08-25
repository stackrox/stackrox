package renderer

import (
	"fmt"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	helmTemplate "github.com/stackrox/rox/pkg/helm/template"
	"github.com/stackrox/rox/pkg/roxctl"
	"github.com/stackrox/rox/pkg/templates"
	"github.com/stackrox/rox/pkg/utils"
)

// generateReadme generates a README file.
func generateReadme(c *Config, mode mode) (string, error) {
	return instructions(*c, mode)
}

func instructionPrefix(deploymentFormat v1.DeploymentFormat) string {
	prefix := "To deploy:\n"
	if roxctl.InMainImage() {
		prefix += "  - Unzip the deployment bundle.\n"
	}
	caSetupPath := "scripts/ca-setup.sh"
	if deploymentFormat == v1.DeploymentFormat_KUBECTL {
		caSetupPath = "central/scripts/ca-setup.sh"
	}
	prefix += fmt.Sprintf("  - If you need to add additional trusted CAs, run %s.\n", caSetupPath)
	return prefix
}

const (
	instructionSuffix = `

For administrator login, select the "Login with username/password" option on
the login page, and log in with username "admin" and the password found in the
"password" file located in the same directory as this README.
`

	kubectlInstructionTemplate = `
  - Deploy Central
    - Run central/scripts/setup.sh
    - Run {{.K8sConfig.Command}} create -R -f central
`

	kubectlScannerTemplate = `
  - Deploy Scanner
     If you want to run the StackRox Scanner:
     - Run scanner/scripts/setup.sh
     - Run {{.K8sConfig.Command}} create -R -f scanner
`
	kubectlCentralDBTemplate = `
  - Deploy Central DB
     To deploy Postgres Central DB and prepare for Central upgrade:
     - Run scripts/setup.sh
     - Run ./deploy-central-db.sh
`
	recommendHelmInstallationTemplate = `
PLEASE NOTE: The recommended way to deploy StackRox is by using Helm. If you have
Helm 3.1+ installed, please consider choosing this deployment route instead. For your
convenience, all required files have been written to the helm/ subdirectory, along with
a README file detailing the Helm-based deployment process.`

	newHelmInstructionTemplate = `
{{- $chartRef := "chart/" -}}
{{- if eq .K8sConfig.DeploymentFormat.String "HELM_VALUES" }}
  - If you haven't done so yet, add the StackRox Helm Chart repository locally:
      helm repo add stackrox https://charts.stackrox.io
{{- $chartRef = "stackrox/central-services" -}}
{{- end }}
  - Deploy Central and Scanner
    - Choose one of the following options for image pull secret setup:
      - Run scripts/setup.sh. This will prompt for Docker credentials for the chosen image
        registries. Then, pass
          --set imagePullSecrets.useExisting="stackrox;stackrox-scanner"
        to the following helm invocation.
      - Add the arguments
          --set imagePullSecrets.username=<username> --set imagePullSecrets.password=<password>
        to the following helm invocation, in order to explicitly configure image pull credentials.
{{- if .K8sConfig.ImageOverrides.MainRegistry }}
      - If the chosen image registry {{ quote .K8sConfig.ImageOverrides.MainRegistry }} does not require image pull secrets, add
        the arguments
          --set imagePullSecrets.allowNone=true
        to the following helm invocation.
{{- end }}
    - Run
        helm install -n stackrox --create-namespace stackrox-central-services {{ $chartRef }}
      passing any additional arguments per the above instructions.
      For installation of 4.1 and laster, it is required to add --set central.persistence.none=true to stop creating
      new persistent storage to attach to Central.
{{- if eq .K8sConfig.DeploymentFormat.String "HELM_VALUES" }}
      If you prefer reading the Helm chart from a directory on your local disk instead of from
      the stackrox upstream repository, replace {{ $chartRef }} with the path to the
      chart.
{{- end }}
`
)

// instructions returns instructions based on the config, which get echoed to standard error,
// as well as go into the README.
func instructions(c Config, mode mode) (string, error) {
	var template string
	if c.K8sConfig.DeploymentFormat == v1.DeploymentFormat_HELM || c.K8sConfig.DeploymentFormat == v1.DeploymentFormat_HELM_VALUES {
		if mode != renderAll {
			return "", fmt.Errorf("mode %s not supported for helm", mode)
		}
		template = newHelmInstructionTemplate
	} else if c.K8sConfig.DeploymentFormat == v1.DeploymentFormat_KUBECTL {
		if mode == scannerOnly {
			template = kubectlScannerTemplate
		} else if mode == centralDBOnly {
			template = kubectlCentralDBTemplate
		} else {
			template = kubectlInstructionTemplate + kubectlScannerTemplate + recommendHelmInstallationTemplate
		}
	} else {
		return "", errors.Errorf("invalid deployment format %v", c.K8sConfig.DeploymentFormat)
	}

	tpl, err := helmTemplate.InitTemplate("temp").Parse(template)
	if err != nil {
		return "", utils.ShouldErr(err)
	}
	data, err := templates.ExecuteToBytes(tpl, &c)
	if err != nil {
		return "", utils.ShouldErr(err)
	}

	instructions := string(data)
	if mode == renderAll {
		prefix := instructionPrefix(c.K8sConfig.DeploymentFormat)
		instructions = prefix + instructions + instructionSuffix
	}

	return instructions, nil
}
