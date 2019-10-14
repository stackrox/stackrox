package v2

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/apiparams"
	"github.com/stackrox/rox/pkg/renderer"
	"github.com/stackrox/rox/pkg/roxctl/defaults"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/scanner/generate/run"
)

// Command represents the generate V2 command.
func Command() *cobra.Command {
	params := apiparams.Scanner{
		ScannerV2Config: renderer.ScannerV2Config{
			Enable:          true,
			PersistenceType: renderer.PersistencePVC,
		},
	}
	var opts run.Options

	c := &cobra.Command{
		Use:   "v2",
		Short: "Generate V2 creates the required YAML files to deploy StackRox Scanner V2 (preview).",
		Long:  "Generate V2 creates the required YAML files to deploy StackRox Scanner V2 (preview).",
		RunE: func(c *cobra.Command, _ []string) error {
			return run.Run(c, &params, opts)
		},
	}

	c.Flags().StringVar(&opts.OutputDir, "output-dir", "", "Output directory for Scanner V2 bundle (leave blank for default)")
	c.Flags().StringVar(&params.ScannerV2Image, "image", "", "Scanner V2 image to use (leave blank to use server default)")
	c.Flags().StringVar(&params.ScannerV2DBImage, "db-image", "", "Scanner V2 DB image to use (leave blank to use server default)")

	// Scanner-V2 Persistence flags
	c.Flags().Var(&flags.PersistenceTypeWrapper{PersistenceType: &params.ScannerV2Config.PersistenceType}, "persistence-type", "scanner-v2 persistence type (pvc, hostpath, none)")

	c.Flags().StringVar(&params.ScannerV2Config.External.Name, "pvc-name", defaults.ScannerV2PVName(), "external volume name (only relevant for persistence type pvc)")
	c.Flags().StringVar(&params.ScannerV2Config.External.StorageClass, "pvc-storage-class", "", "scanner-v2 storage class name (optional if you have a default StorageClass configured) (only relevant for persistence type pvc)")
	c.Flags().Uint32Var(&params.ScannerV2Config.External.Size, "pvc-size", defaults.ScannerV2PVSize(), "size of scanner v2 persistent volume (in Gi) (only relevant for persistence type pvc)")

	c.Flags().StringVar(&params.ScannerV2Config.HostPath.HostPath, "hostpath", defaults.ScannerV2HostPath(), "path on the host (only relevant for persistence type hostpath)")
	c.Flags().StringVar(&params.ScannerV2Config.HostPath.NodeSelectorKey, "hostpath-node-selector-key", "", "hostpath node selector key (e.g. kubernetes.io/hostname) (only relevant for persistence type hostpath)")
	c.Flags().StringVar(&params.ScannerV2Config.HostPath.NodeSelectorValue, "hostpath-node-selector-value", "", "hostpath node selector value (only relevant for persistence type hostpath)")

	return c
}
