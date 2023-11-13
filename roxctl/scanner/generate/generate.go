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
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/istioutils"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/logger"
	"github.com/stackrox/rox/roxctl/common/zipdownload"
	"github.com/stackrox/rox/roxctl/scanner/clustertype"
)

const (
	scannerGenerateAPIPath = "/api/extensions/scanner/zip"

	istioSupportArg = "istio-support"
)

type scannerGenerateCommand struct {
	// Properties that are bound to cobra flags.
	outputDir string
	apiParams apiparams.Scanner
	timeout   time.Duration

	enablePodSecurityPolicies bool

	// Properties that are injected or constructed.
	env environment.Environment
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

		return errox.InvalidArgs.Newf(
			"unsupported Istio version %q used for argument %q. Use one of the following: [%s]",
			cmd.apiParams.IstioVersion, "--"+istioSupportArg, strings.Join(istioutils.ListKnownIstioVersions(), "|"),
		)
	}

	return nil
}

func (cmd *scannerGenerateCommand) generate(logger logger.Logger) error {
	common.LogInfoPsp(logger, cmd.enablePodSecurityPolicies)

	cmd.apiParams.ClusterType = clustertype.Get().String()
	cmd.apiParams.DisablePodSecurityPolicies = !cmd.enablePodSecurityPolicies

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
	}, cmd.env)

	return errors.Wrap(err, "could not get scanner bundle")
}

// Command represents the generate command.
func Command(cliEnvironment environment.Environment) *cobra.Command {
	scannerGenerateCmd := &scannerGenerateCommand{env: cliEnvironment}

	c := &cobra.Command{
		Use:   "generate",
		Short: "Generate the required YAML configuration files to deploy StackRox Scanner",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			scannerGenerateCmd.construct(c)

			if err := scannerGenerateCmd.validate(); err != nil {
				return err
			}

			return scannerGenerateCmd.generate(cliEnvironment.Logger())
		},
	}

	c.PersistentFlags().Var(clustertype.Value(storage.ClusterType_KUBERNETES_CLUSTER), "cluster-type", "Type of cluster the scanner will run on (k8s, openshift)")

	c.Flags().StringVar(&scannerGenerateCmd.outputDir, "output-dir", "", "Output directory for scanner bundle (leave blank for default)")
	c.Flags().StringVar(&scannerGenerateCmd.apiParams.ScannerImage, flags.FlagNameScannerImage, "", "Scanner image to use (leave blank to use server default)")
	c.Flags().StringVar(&scannerGenerateCmd.apiParams.IstioVersion, istioSupportArg, "",
		fmt.Sprintf(
			"Generate deployment files supporting the given Istio version. Valid versions: %s",
			strings.Join(istioutils.ListKnownIstioVersions(), ", ")))
	c.PersistentFlags().BoolVar(&scannerGenerateCmd.enablePodSecurityPolicies, "enable-pod-security-policies", true, "Create PodSecurityPolicy resources (for pre-v1.25 Kubernetes)")

	flags.AddTimeout(c)
	flags.AddRetryTimeout(c)

	return c
}
