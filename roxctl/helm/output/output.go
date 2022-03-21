package output

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/helm/charts"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/helm/internal/common"
	"helm.sh/helm/v3/pkg/chart/loader"
)

// Command for writing Helm Chart
func Command(cliEnvironment environment.Environment) *cobra.Command {
	helmOutputCmd := &helmOutputCommand{env: cliEnvironment}

	c := &cobra.Command{
		Use: fmt.Sprintf("output <%s>", common.PrettyChartNameList),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := helmOutputCmd.Construct(args, cmd); err != nil {
				return err
			}
			if err := helmOutputCmd.Validate(); err != nil {
				return err
			}

			return helmOutputCmd.outputHelmChart()
		},
	}
	c.PersistentFlags().StringVar(&helmOutputCmd.outputDir, "output-dir", "", "path to the output directory for Helm chart (default: './stackrox-<chart name>-chart')")
	c.PersistentFlags().BoolVar(&helmOutputCmd.removeOutputDir, "remove", false, "remove the output directory if it already exists")
	c.PersistentFlags().BoolVar(&helmOutputCmd.rhacs, "rhacs", false,
		fmt.Sprintf("render RHACS chart flavor (deprecated: use '--%s=%s' instead)", flags.ImageDefaultsFlagName, defaults.ImageFlavorNameRHACSRelease))

	if !buildinfo.ReleaseBuild {
		flags.AddHelmChartDebugSetting(c)
	}
	flags.AddImageDefaults(c.PersistentFlags(), &helmOutputCmd.imageFlavor)
	return c
}

// helmOutputCommand holds all configurations and metadata to execute a `helm output` command
type helmOutputCommand struct {
	// properties bound to cobra flags
	outputDir       string
	removeOutputDir bool
	rhacs           bool
	imageFlavor     string

	// values injected from either Construct, parent command or for abstracting external dependencies
	chartName               string
	flavorProvided          bool
	chartTemplatePathPrefix image.ChartPrefix
	env                     environment.Environment
	logger                  environment.Logger
}

// Construct will enhance the struct with other values coming either from os.Args, other, global flags or environment variables
func (cfg *helmOutputCommand) Construct(args []string, cmd *cobra.Command) error {
	if len(args) != 1 {
		return errors.New("incorrect number of arguments, see --help for usage information")
	}

	cfg.chartName = args[0]
	cfg.flavorProvided = cmd.Flags().Changed(flags.ImageDefaultsFlagName)

	cfg.logger = cfg.env.Logger()

	return nil
}

// Validate will validate the injected values and check whether it's possible to execute the operation with the
// provided values
func (cfg *helmOutputCommand) Validate() error {
	// Check that chart template prefix exists
	chartTemplatePathPrefix := common.ChartTemplates[cfg.chartName]
	if chartTemplatePathPrefix == "" {
		return errox.NewErrInvalidArgs("unknown chart, see --help for list of supported chart names")
	}
	cfg.chartTemplatePathPrefix = chartTemplatePathPrefix

	if cfg.outputDir == "" {
		cfg.outputDir = fmt.Sprintf("./stackrox-%s-chart", cfg.chartName)
		cfg.logger.ErrfLn("No output directory specified, using default directory %q", cfg.outputDir)
	}

	if _, err := os.Stat(cfg.outputDir); err == nil {
		if cfg.removeOutputDir {
			if err := os.RemoveAll(cfg.outputDir); err != nil {
				return errors.Wrapf(err, "failed to remove output dir %s", cfg.outputDir)
			}
			cfg.logger.ErrfLn("Removed output directory %s", cfg.outputDir)
		} else {
			cfg.logger.ErrfLn("Directory %q already exists, use --remove or select a different directory with --output-dir.", cfg.outputDir)
			return errox.AlreadyExists.New(fmt.Sprintf("directory %q already exists", cfg.outputDir))
		}
	} else if !os.IsNotExist(err) {
		return errors.Wrapf(err, "failed to check if directory %q exists", cfg.outputDir)
	}

	return nil
}

func (cfg *helmOutputCommand) outputHelmChart() error {
	// load chart template meta values
	chartMetaValues, err := cfg.getChartMetaValues(buildinfo.ReleaseBuild)
	if err != nil {
		return err
	}

	// load image with templates
	templateImage := image.GetDefaultImage()
	if flags.IsDebug() {
		templateImage = flags.GetDebugHelmImage()
	}

	// Load and render template files.
	renderedChartFiles, err := templateImage.LoadAndInstantiateChartTemplate(cfg.chartTemplatePathPrefix, chartMetaValues)
	if err != nil {
		return errors.Wrapf(err, "loading and instantiating %s helmtpl", cfg.chartName)
	}

	// Write rendered files to output directory.
	if err := os.MkdirAll(cfg.outputDir, 0755); err != nil {
		return errors.Wrapf(err, "unable to create folder %q", cfg.outputDir)
	}
	for _, f := range renderedChartFiles {
		if err := writeFile(f, cfg.outputDir); err != nil {
			return errors.Wrapf(err, "error writing file %q", f.Name)
		}
	}
	cfg.logger.ErrfLn("Written Helm chart %s to directory %q", cfg.chartName, cfg.outputDir)

	return nil
}

func (cfg *helmOutputCommand) getChartMetaValues(release bool) (*charts.MetaValues, error) {
	handleRhacsWarnings(cfg.rhacs, cfg.flavorProvided, cfg.logger)
	if cfg.rhacs {
		if cfg.flavorProvided {
			return nil, errox.InvalidArgs.New(fmt.Sprintf("flag '--rhacs' is deprecated and must not be used together with '--%s'. Remove '--rhacs' flag and specify only '--%s'", flags.ImageDefaultsFlagName, flags.ImageDefaultsFlagName))
		}
		cfg.imageFlavor = defaults.ImageFlavorNameRHACSRelease
	}
	imageFlavor, err := defaults.GetImageFlavorByName(cfg.imageFlavor, release)
	if err != nil {
		return nil, errox.InvalidArgs.New(fmt.Sprintf("'--%s': %v", flags.ImageDefaultsFlagName, err))
	}
	return charts.GetMetaValuesForFlavor(imageFlavor), nil
}

func handleRhacsWarnings(rhacs, imageFlavorProvided bool, logger environment.Logger) {
	if rhacs {
		logger.WarnfLn("'--rhacs' is deprecated, please use '--%s=%s' instead", flags.ImageDefaultsFlagName, defaults.ImageFlavorNameRHACSRelease)
	} else if !imageFlavorProvided {
		logger.WarnfLn("Default image registries have changed. Images will be taken from 'registry.redhat.io'. Specify '--%s=%s' command line argument to use images from 'stackrox.io' registries.", flags.ImageDefaultsFlagName, defaults.ImageFlavorNameStackRoxIORelease)
	}
}

func writeFile(file *loader.BufferedFile, destDir string) error {
	outputPath := filepath.Join(destDir, filepath.FromSlash(file.Name))
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return errors.Wrapf(err, "creating directory for file %q", file.Name)
	}

	perms := os.FileMode(0644)
	if filepath.Ext(file.Name) == ".sh" {
		perms = os.FileMode(0755)
	}
	return errors.Wrap(os.WriteFile(outputPath, file.Data, perms), "could not write file")
}
