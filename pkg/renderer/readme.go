package renderer

import (
	"fmt"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/roxctl"
)

// generateReadme generates a README file.
func generateReadme(c *Config, mode mode) (string, error) {
	return instructions(*c, mode)
}

func instructionPrefix() string {
	prefix := "To deploy:\n"
	if roxctl.InMainImage() {
		prefix += "  - Unzip the deployment bundle.\n"
	}
	prefix += "  - If you need to add additional trusted CAs, run central/scripts/ca-setup.sh.\n"
	return prefix
}

const (
	instructionSuffix = `

For administrator login, select the "Login with username/password" option on
the login page, and log in with username "admin" and the password found in the
"password" file located in the same directory as this README.
`
	helmInstructionTemplate = `
  {{if not .K8sConfig.Monitoring.Type.None}}
  - Deploy Monitoring
    - Run monitoring/scripts/setup.sh
    - Run helm install --name monitoring monitoring
  {{- end}}
  - Deploy Central
    - Run central/scripts/setup.sh
    - Run helm install --name central central
  - Deploy Scanner
    {{ $scannerName := "" -}}
    {{ if .K8sConfig.ScannerV2Config.Enable -}}
    {{ $scannerName = "scannerv2" }}
    {{ else }}
    {{ $scannerName = "scanner" }}
    {{ end -}}
    - Run {{ $scannerName }}/scripts/setup.sh
    - If you want to run the StackRox scanner, run helm install --name {{ $scannerName }} {{ $scannerName }}
`

	kubectlInstructionTemplate = `{{if not .K8sConfig.Monitoring.Type.None}}
  - Deploy Monitoring
    - Run monitoring/scripts/setup.sh
    - Run {{.K8sConfig.Command}} create -R -f monitoring
  {{- end}}
  - Deploy Central
    - Run central/scripts/setup.sh
    - Run {{.K8sConfig.Command}} create -R -f central
`

	kubectlScannerTemplate = `
  {{ $scannerName := "" -}}
  {{ if .K8sConfig.ScannerV2Config.Enable -}}
  {{ $scannerName = "scannerv2" }}
  {{ else }}
  {{ $scannerName = "scanner" }}
  {{ end -}}
  - Deploy Scanner {{ if .K8sConfig.ScannerV2Config.Enable -}}V2{{ end }}
     If you want to run the StackRox scanner:
     - Run {{$scannerName}}/scripts/setup.sh
     - Run {{.K8sConfig.Command}} create -R -f {{$scannerName}}
	`
)

// instructions returns instructions based on the config, which get echoed to standard error,
// as well as go into the README.
func instructions(c Config, mode mode) (string, error) {
	var template string
	if c.K8sConfig.DeploymentFormat == v1.DeploymentFormat_HELM {
		if mode != renderAll {
			return "", fmt.Errorf("mode %s not supported for helm", mode)
		}
		template = helmInstructionTemplate
	} else {
		if mode == scannerOnly {
			template = kubectlScannerTemplate
		} else {
			template = kubectlInstructionTemplate + kubectlScannerTemplate
		}
	}

	data, err := executeRawTemplate(template, &c)
	if err != nil {
		errorhelpers.PanicOnDevelopment(err)
		return "", err
	}

	instructions := string(data)
	if mode == renderAll {
		prefix := instructionPrefix()
		instructions = prefix + instructions + instructionSuffix
	}

	return instructions, nil
}
