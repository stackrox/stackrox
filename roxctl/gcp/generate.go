package gcp

import (
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/renderer"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/roxctl/central/deploy"
)

// Generate defines the gcp generate command.
func Generate() *cobra.Command {
	c := &cobra.Command{
		Use:   "generate",
		Short: "Generate helm chart for GCP marketplace",
		Long:  "Generate helm chart from GCP marketplace values yaml",
		RunE: func(c *cobra.Command, _ []string) error {
			valuesFilename, _ := c.Flags().GetString("values-file")
			outputDir, _ := c.Flags().GetString("output-dir")
			values, err := loadValues(valuesFilename)
			if err != nil {
				return err
			}

			return generate(outputDir, values)
		},
	}
	c.PersistentFlags().String("values-file", "/data/final_values.yaml", "path to the input yaml values file")
	c.PersistentFlags().String("output-dir", "", "path to the output helm chart directory")

	return c
}

func generate(outputDir string, values *Values) error {
	config := renderer.Config{
		Version:        version.GetMainVersion(),
		OutputDir:      outputDir,
		GCPMarketplace: true,
		ClusterType:    storage.ClusterType_KUBERNETES_CLUSTER,
		K8sConfig: &renderer.K8sConfig{
			AppName: values.Name,
			CommonConfig: renderer.CommonConfig{
				MainImage:       values.MainImage,
				ScannerImage:    values.ScannerImage,
				MonitoringImage: values.MonitoringImage,
			},
			ConfigType:       v1.DeploymentFormat_HELM,
			DeploymentFormat: v1.DeploymentFormat_HELM,
			LoadBalancerType: v1.LoadBalancerType_NONE,
			OfflineMode:      false,
		},
		Password:    values.Password,
		LicenseData: []byte(values.License),
	}

	return deploy.OutputZip(config)
}
