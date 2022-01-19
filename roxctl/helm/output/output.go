package output

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/helm/charts"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/helm/internal/common"
	"helm.sh/helm/v3/pkg/chart/loader"
)

func handleRhacsFlag(flavorName string) error {
	if flavorName == "" || flavorName == defaults.ImageFlavorNameRHACSRelease {
		return nil
	}
	return fmt.Errorf("flag '--rhacs' collides with '--image-defaults=%s'. Remove '--rhacs' flag", flavorName)
}

func getMetaValues(inputFlavorName string, rhacs, release bool, cliEnvironment environment.Environment) (*charts.MetaValues, error) {
	if !rhacs && inputFlavorName == "" {
		cliEnvironment.Logger().WarnfLn("images are taken from 'registry.redhat.io'. Use '--image-defaults=stackrox.io' to restore previous behavior")
	}
	if rhacs {
		cliEnvironment.Logger().WarnfLn("'--rhacs' is deprecated in favor of '--image-defaults=rhacs'")
		if err := handleRhacsFlag(inputFlavorName); err != nil {
			return nil, errorhelpers.NewErrInvalidArgsf("'--image-defaults': %v", err)
		}
		return charts.GetMetaValuesForFlavor(defaults.RHACSReleaseImageFlavor()), nil
	}

	flavorName := defaults.ImageFlavorNameRHACSRelease
	if !buildinfo.ReleaseBuild {
		flavorName = defaults.ImageFlavorNameDevelopmentBuild
	}
	if inputFlavorName != "" {
		flavorName = inputFlavorName
	}
	imageFlavor, err := defaults.GetImageFlavorByName(flavorName, release)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid value of '--image-defaults=%s'", flavorName)
	}
	return charts.GetMetaValuesForFlavor(imageFlavor), nil
}

func outputHelmChart(chartName string, outputDir string, removeOutputDir bool, rhacs bool, imageFlavor string, debug bool, debugChartPath string, cliEnvironment environment.Environment) error {
	// Lookup chart template prefix.
	chartTemplatePathPrefix := common.ChartTemplates[chartName]
	if chartTemplatePathPrefix == "" {
		return errors.New("unknown chart, see --help for list of supported chart names")
	}

	metaVals, err := getMetaValues(imageFlavor, rhacs, buildinfo.ReleaseBuild, cliEnvironment)
	if err != nil {
		return err
	}

	if outputDir == "" {
		outputDir = fmt.Sprintf("./stackrox-%s-chart", chartName)
		fmt.Fprintf(os.Stderr, "No output directory specified, using default directory %q.\n", outputDir)
	}

	if _, err := os.Stat(outputDir); err == nil {
		if removeOutputDir {
			if err := os.RemoveAll(outputDir); err != nil {
				return errors.Wrapf(err, "failed to remove output dir %s", outputDir)
			}
			fmt.Fprintf(os.Stderr, "Removed output directory %s\n", outputDir)
		} else {
			fmt.Fprintf(os.Stderr, "Directory %q already exists, use --remove or select a different directory with --output-dir.\n", outputDir)
			return fmt.Errorf("directory %q already exists", outputDir)
		}
	} else if !os.IsNotExist(err) {
		return errors.Wrapf(err, "failed to check if directory %q exists", outputDir)
	}

	// load image with templates
	templateImage := image.GetDefaultImage()
	if debug {
		templateImage = image.NewImage(os.DirFS(debugChartPath))
	}

	// Load and render template files.
	renderedChartFiles, err := templateImage.LoadAndInstantiateChartTemplate(chartTemplatePathPrefix, metaVals)
	if err != nil {
		return errors.Wrapf(err, "loading and instantiating %s helmtpl", chartName)
	}

	// Write rendered files to output directory.
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return errors.Wrapf(err, "unable to create folder %q", outputDir)
	}
	for _, f := range renderedChartFiles {
		if err := writeFile(f, outputDir); err != nil {
			return errors.Wrapf(err, "error writing file %q", f.Name)
		}
	}
	fmt.Fprintf(os.Stderr, "Written Helm chart %s to directory %q.\n", chartName, outputDir)

	return nil
}

// Command for writing Helm Chart.
func Command(cliEnvironment environment.Environment) *cobra.Command {
	var outputDir string
	var removeOutputDir bool
	var debug bool
	var debugChartPath string
	var rhacs bool
	var imageFlavor string

	c := &cobra.Command{
		Use: fmt.Sprintf("output <%s>", common.PrettyChartNameList),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("incorrect number of arguments, see --help for usage information")
			}
			chartName := args[0]
			return outputHelmChart(chartName, outputDir, removeOutputDir, rhacs, imageFlavor, debug, debugChartPath, cliEnvironment)
		},
	}
	c.PersistentFlags().StringVar(&outputDir, "output-dir", "", "path to the output directory for Helm chart (default: './stackrox-<chart name>-chart')")
	c.PersistentFlags().BoolVar(&removeOutputDir, "remove", false, "remove the output directory if it already exists")
	c.PersistentFlags().BoolVar(&rhacs, "rhacs", false, "render RHACS chart flavor")

	if !buildinfo.ReleaseBuild {
		defaultDebugPath := path.Join(os.Getenv("GOPATH"), "src/github.com/stackrox/stackrox/image/")
		c.PersistentFlags().BoolVar(&debug, "debug", false, "read templates from local filesystem")
		c.PersistentFlags().StringVar(&debugChartPath, "debug-path", defaultDebugPath, "path to helm templates on your local filesystem")
	}
	imageFlavorDefault := defaults.ImageFlavorNameRHACSRelease
	if !buildinfo.ReleaseBuild {
		imageFlavorDefault = defaults.ImageFlavorNameDevelopmentBuild
	}
	// Leave the third param as "", because it allows us to detect whether the user has used this flag or left it empty
	c.PersistentFlags().StringVar(&imageFlavor, "image-defaults", "", fmt.Sprintf("default container registry for container images (default: \"%s\")", imageFlavorDefault))

	return c
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
	return os.WriteFile(outputPath, file.Data, perms)
}
