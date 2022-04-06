package generate

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/apiparams"
	"github.com/stackrox/rox/pkg/istioutils"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/zipdownload"
	"github.com/stackrox/rox/roxctl/scanner/clustertype"
)

const (
	scannerGenerateAPIPath = "/api/extensions/scanner/zip"
)

type scannerGenerateCommand struct {
	// Properties that are bound to cobra flags.
	outputDir string
	apiParams apiparams.Scanner
	timeout   time.Duration
}

func (cmd *scannerGenerateCommand) construct(c *cobra.Command) {
	cmd.timeout = flags.Timeout(c)
}

func (cmd *scannerGenerateCommand) validate() error {
	// Validate supported Istio versions.
	if cmd.apiParams.IstioVersion != "" {
		for _, istioVersion := range istioutils.ListKnownIstioVersions() {
			if cmd.apiParams.IstioVersion == istioVersion {
				return nil
			}
		}

		return errors.New("unsupported Istio version")
	}

	return nil
}

func (cmd *scannerGenerateCommand) generate() error {
	cmd.apiParams.ClusterType = clustertype.Get().String()
	body, err := json.Marshal(cmd.apiParams)
	if err != nil {
		return errors.Wrap(err, "could not marshal scanner params")
	}

	err = zipdownload.GetZip(zipdownload.GetZipOptions{
		Path:       scannerGenerateAPIPath,
		Method:     http.MethodPost,
		Body:       body,
		Timeout:    cmd.timeout,
		BundleType: "scanner",
		ExpandZip:  true,
		OutputDir:  cmd.outputDir,
	})

	return errors.Wrap(err, "could not get scanner zip")
}

// Command represents the generate command.
func Command() *cobra.Command {
	scannerGenerateCmd := &scannerGenerateCommand{}

	c := &cobra.Command{
		Use:  "generate",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			scannerGenerateCmd.construct(c)

			if err := scannerGenerateCmd.validate(); err != nil {
				return err
			}

			return scannerGenerateCmd.generate()
		},
	}

	c.PersistentFlags().Var(clustertype.Value(storage.ClusterType_KUBERNETES_CLUSTER), "cluster-type", "type of cluster the scanner will run on (k8s, openshift)")

	c.Flags().StringVar(&scannerGenerateCmd.outputDir, "output-dir", "", "Output directory for scanner bundle (leave blank for default)")
	c.Flags().BoolVar(&scannerGenerateCmd.apiParams.OfflineMode, "offline-mode", false, "whether to run the scanner in offline mode (so "+
		"it doesn't reach out to the internet for updates)")
	c.Flags().StringVar(&scannerGenerateCmd.apiParams.ScannerImage, flags.FlagNameScannerImage, "", "Scanner image to use (leave blank to use server default)")
	c.Flags().StringVar(&scannerGenerateCmd.apiParams.IstioVersion, "istio-support", "",
		fmt.Sprintf(
			"Generate deployment files supporting the given Istio version. Valid versions: %s",
			strings.Join(istioutils.ListKnownIstioVersions(), ", ")))

	return c
}
