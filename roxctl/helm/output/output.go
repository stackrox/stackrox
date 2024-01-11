package output

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/buildinfo"
	pkgEnv "github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/helm/charts"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	env "github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/logger"
	"github.com/stackrox/rox/roxctl/helm/internal/common"
	"helm.sh/helm/v3/pkg/chart/loader"
)

// Command for writing Helm Chart
func Command(cliEnvironment env.Environment) *cobra.Command {
	helmOutputCmd := &helmOutputCommand{env: cliEnvironment}

	c := &cobra.Command{
		Use:       fmt.Sprintf("output <%s>", common.PrettyChartNameList),
		Short:     "Output a Helm Chart.",
		ValidArgs: []string{common.ChartCentralServices, common.ChartSecuredClusterServices},
		Args:      cobra.ExactValidArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			helmOutputCmd.Construct(args[0], cmd)

			if err := helmOutputCmd.Validate(); err != nil {
				return err
			}

			return helmOutputCmd.outputHelmChart()
		},
	}
	c.PersistentFlags().StringVar(&helmOutputCmd.outputDir, "output-dir", "", "path to the output directory for Helm chart (default: './stackrox-<chart name>-chart')")
	c.PersistentFlags().BoolVar(&helmOutputCmd.removeOutputDir, "remove", false, "remove the output directory if it already exists")
	c.PersistentFlags().BoolVar(&helmOutputCmd.rhacs, "rhacs", false, "render RHACS chart flavor")
	c.PersistentFlags().BoolVar(&helmOutputCmd.telemetry, "enable-telemetry", version.IsReleaseVersion(), "whether to enable telemetry")

	deprecationNote := fmt.Sprintf("use '--%s=%s' instead", flags.ImageDefaultsFlagName, defaults.ImageFlavorNameRHACSRelease)
	utils.Must(c.PersistentFlags().MarkDeprecated("rhacs", deprecationNote))

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
	telemetry       bool

	// values injected from either Construct, parent command or for abstracting external dependencies
	chartName               string
	flavorProvided          bool
	chartTemplatePathPrefix image.ChartPrefix
	env                     env.Environment
}

// Construct will enhance the struct with other values coming either from os.Args, other, global flags or environment variables
func (cfg *helmOutputCommand) Construct(chartName string, cmd *cobra.Command) {
	cfg.chartName = chartName
	cfg.flavorProvided = cmd.Flags().Changed(flags.ImageDefaultsFlagName)
}

// Validate will validate the injected values and check whether it's possible to execute the operation with the
// provided values
func (cfg *helmOutputCommand) Validate() error {
	cfg.chartTemplatePathPrefix = common.ChartTemplates[cfg.chartName]

	if cfg.outputDir == "" {
		cfg.outputDir = fmt.Sprintf("./stackrox-%s-chart", cfg.chartName)
		cfg.env.Logger().WarnfLn("No output directory specified, using default directory %q", cfg.outputDir)
	}

	if _, err := os.Stat(cfg.outputDir); err == nil {
		if cfg.removeOutputDir {
			if err := os.RemoveAll(cfg.outputDir); err != nil {
				return errors.Wrapf(err, "failed to remove output dir %s", cfg.outputDir)
			}
			cfg.env.Logger().WarnfLn("Removed output directory %s", cfg.outputDir)
		} else {
			cfg.env.Logger().ErrfLn("Directory %q already exists, use --remove or select a different directory with --output-dir.", cfg.outputDir)
			return errox.AlreadyExists.Newf("directory %q already exists", cfg.outputDir)
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
		return errors.Wrap(err, "unable to get chart meta values")
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
	cfg.env.Logger().InfofLn("Written Helm chart %s to directory %q", cfg.chartName, cfg.outputDir)

	return nil
}

func (cfg *helmOutputCommand) getChartMetaValues(release bool) (*charts.MetaValues, error) {
	handleRhacsWarnings(cfg.rhacs, cfg.flavorProvided, cfg.env.Logger())
	if cfg.rhacs {
		if cfg.flavorProvided {
			return nil, errox.InvalidArgs.Newf("flag '--rhacs' is deprecated and must not be used together with '--%s'. Remove '--rhacs' flag and specify only '--%s'", flags.ImageDefaultsFlagName, flags.ImageDefaultsFlagName)
		}
		cfg.imageFlavor = defaults.ImageFlavorNameRHACSRelease
	}
	imageFlavor, err := defaults.GetImageFlavorByName(cfg.imageFlavor, release)
	if err != nil {
		return nil, errox.InvalidArgs.Newf("'--%s': %v", flags.ImageDefaultsFlagName, err)
	}

	values := charts.GetMetaValuesForFlavor(imageFlavor)

	// For testing purposes, running a non-release roxctl version, provide
	// TelemetryStorageKey with the test value and set --enable-telemetry=true
	// to get telemetry enabled by default in the resulting chart.
	if cfg.telemetry && (version.IsReleaseVersion() || pkgEnv.TelemetryStorageKey.Setting() != "") &&
		pkgEnv.TelemetryStorageKey.Setting() != phonehome.DisabledKey {
		values.TelemetryEnabled = true
		values.TelemetryKey = pkgEnv.TelemetryStorageKey.Setting()
		values.TelemetryEndpoint = pkgEnv.TelemetryEndpoint.Setting()
	} else {
		values.TelemetryEnabled = false
		values.TelemetryKey = phonehome.DisabledKey
	}
	return values, nil
}

func handleRhacsWarnings(rhacs, _ bool, logger logger.Logger) {
	if rhacs {
		logger.WarnfLn("'--rhacs' is deprecated, please use '--%s=%s' instead", flags.ImageDefaultsFlagName, defaults.ImageFlavorNameRHACSRelease)
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
