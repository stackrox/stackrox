package apiserver

import (
	"github.com/stackrox/rox/benchmarks/checks"
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/generated/api/v1"
	"gopkg.in/yaml.v2"
	"k8s.io/apiserver/pkg/server/options/encryptionconfig"
)

type encryptionProvider struct{}

func (c *encryptionProvider) Definition() utils.Definition {
	return utils.Definition{
		BenchmarkCheckDefinition: v1.BenchmarkCheckDefinition{
			Name:        "CIS Kubernetes v1.2.0 - 1.1.34",
			Description: "Ensure that the encryption provider is set to aescbc",
		}, Dependencies: []utils.Dependency{utils.InitKubeAPIServerConfig},
	}
}

func (c *encryptionProvider) Run() (result v1.BenchmarkCheckResult) {
	params, ok := utils.KubeAPIServerConfig.Get("experimental-encryption-provider-config")
	if !ok {
		utils.Warn(&result)
		utils.AddNotes(&result, "--experimental-encryption-provider-config is not set, which means that aescbc is not in use")
		return
	}
	output, err := utils.ReadFile(params.String())
	if err != nil {
		utils.Note(&result)
		utils.AddNotef(&result, "Could not read file '%v' to check for aescbc specification due to %+v. Please manually check", params.String(), err)
		return
	}
	var config encryptionconfig.EncryptionConfig
	if err := yaml.Unmarshal([]byte(output), &config); err != nil {
		utils.Warn(&result)
		utils.AddNotef(&result, "Could not parse file '%v' to check for aescbc specification due to %+v. Please manually check", params.String(), err)
		return
	}
	if config.Kind != "EncryptionConfig" {
		utils.Warn(&result)
		utils.AddNotef(&result, "Incorrect configuration kind '%v' in file '%v'", config.Kind, params.String())
		return
	}
	for _, resource := range config.Resources {
		for _, provider := range resource.Providers {
			if provider.AESCBC != nil {
				utils.Pass(&result)
				return
			}
		}
	}
	utils.Warn(&result)
	utils.AddNotef(&result, "Encryption provider config file '%v' does not use recommended 'aescbc' encryption", params.String())
	return
}

// NewEncryptionProvider implements CIS Kubernetes v1.2.0 1.1.34
func NewEncryptionProvider() utils.Check {
	return &encryptionProvider{}
}

func init() {
	checks.AddToRegistry(
		NewEncryptionProvider(),
	)
}
