package generate

import (
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/renderer"
	"github.com/stackrox/rox/pkg/roxctl"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/zip"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/logger"
	"github.com/stackrox/rox/roxctl/common/zipdownload"
)

const (
	centralDBCertGeneratePath = "/api/extensions/certgen/centraldb"
)

var (
	errCreateCentralDBBundleSupported = errox.InvariantViolation.New("central does not support central db bundle functionality")
	centralDBCertBundle               = set.NewFrozenStringSet(mtls.CACertFileName, mtls.CentralDBCertFileName, mtls.CentralDBKeyFileName)
)

type generateCommand struct {
	// Properties that are bound to cobra flags.
	config *renderer.Config

	// Properties that are injected or constructed.
	env environment.Environment

	// timeout to make Central API call
	timeout time.Duration
}

// Command represents the generate command.
func Command(cliEnvironment environment.Environment) *cobra.Command {
	cmd := &generateCommand{config: &cfg, env: cliEnvironment}

	c := &cobra.Command{
		Use:    "generate",
		Short:  "generate a Central DB bundle",
		Long:   "generate a Central DB bundle which contains all required YAML files and scripts to deploy the central DB",
		Hidden: true,
	}

	if !buildinfo.ReleaseBuild {
		flags.AddHelmChartDebugSetting(c)
	}
	c.PersistentFlags().BoolVar(&cmd.config.EnablePodSecurityPolicies, "enable-pod-security-policies", true, "Create PodSecurityPolicy resources (for pre-v1.25 Kubernetes)")
	c.PersistentPreRunE = func(*cobra.Command, []string) error {
		cmd.construct(c)
		return cmd.populateMTLS()
	}

	c.AddCommand(k8s(cliEnvironment))
	c.AddCommand(openshift(cliEnvironment))

	return c
}

func (cmd *generateCommand) populateMTLS() error {
	logger := cmd.env.Logger()
	logger.InfofLn("Populating Central DB Certificate from bundle...")
	fileMap, err := zipdownload.GetZipFiles(zipdownload.GetZipOptions{
		Path:       centralDBCertGeneratePath,
		Method:     http.MethodPost,
		Timeout:    cmd.timeout,
		BundleType: "central-db",
		ExpandZip:  true,
	}, logger)
	if err != nil {
		return err
	}
	err = verifyCentralDBBundleFiles(fileMap)
	if err != nil {
		return err
	}
	cmd.config.SecretsByteMap = map[string][]byte{
		"ca.pem":              fileMap[mtls.CACertFileName].Content,
		"central-db-cert.pem": fileMap[mtls.CentralDBCertFileName].Content,
		"central-db-key.pem":  fileMap[mtls.CentralDBKeyFileName].Content,
		"central-db-password": []byte(renderer.CreatePassword()),
	}
	return nil
}

func (cmd *generateCommand) construct(c *cobra.Command) {
	cmd.timeout = flags.Timeout(c)
}

func generateBundleWrapper(config renderer.Config) (*zip.Wrapper, error) {
	rendered, err := render(config)
	if err != nil {
		return nil, err
	}

	wrapper := zip.NewWrapper()
	wrapper.AddFiles(rendered...)
	return wrapper, errors.Wrap(err, "could not get scanner bundle")
}

func OutputZip(logger logger.Logger, config renderer.Config) error {
	logger.InfofLn("Generating Central DB bundle...")
	common.LogInfoPsp(logger, config.EnablePodSecurityPolicies)
	wrapper, err := generateBundleWrapper(config)
	if err != nil {
		return err
	}
	var outputPath string
	if roxctl.InMainImage() {
		bytes, err := wrapper.Zip()
		if err != nil {
			return errors.Wrap(err, "error generating zip file")
		}
		_, err = os.Stdout.Write(bytes)
		if err != nil {
			return errors.Wrap(err, "couldn't write zip file")
		}
	} else {
		var err error
		outputPath, err = wrapper.Directory(config.OutputDir)
		if err != nil {
			return errors.Wrap(err, "error generating directory for Central output")
		}
	}

	logger.InfofLn("Done!")

	if outputPath != "" {
		logger.InfofLn("Wrote central bundle to %q", outputPath)
	}

	return nil
}

func render(config renderer.Config) ([]*zip.File, error) {
	flavor, err := defaults.GetImageFlavorByName(config.K8sConfig.ImageFlavorName, buildinfo.ReleaseBuild)
	if err != nil {
		return nil, common.ErrInvalidCommandOption.CausedByf("'--%s': %v", flags.ImageDefaultsFlagName, err)
	}

	return renderer.RenderCentralDBOnly(config, flavor)
}

func verifyCentralDBBundleFiles(fm map[string]*zip.File) error {
	var errs errorhelpers.ErrorList

	checkList := centralDBCertBundle.Unfreeze()
	for k, v := range fm {
		if 0 == len(v.Content) {
			errs.AddError(errors.Errorf("empty file in Central DB certificate bundle: %s", v.Name))
		}
		if !centralDBCertBundle.Contains(k) {
			errs.AddError(errors.Errorf("unexpected file in Central DB certificate bundle: %s", k))
		}
		checkList.Remove(k)
	}
	if checkList.Cardinality() != 0 {
		errs.AddError(errors.Errorf("missing file(s) in Central DB certificate bundle %s", checkList.ElementsString(",")))
	}
	return errs.ToError()
}
