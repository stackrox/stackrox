package output

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/helm/charts"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/roxctl/helm/internal/common"
	"helm.sh/helm/v3/pkg/chart/loader"
)

const (
	flavorDevelopment string = "development"
	flavorStackRoxIO  string = "stackrox.io"
	// flavorRHACS       string = "rhacs" // TODO(RS-380): Uncomment to enable rhacs flavor
)

var allowedFlavors set.StringSet

func init() {
	allowedFlavors = set.NewStringSet(flavorStackRoxIO)
	if !buildinfo.ReleaseBuild {
		allowedFlavors.Add(flavorDevelopment)
	}
}

func getMetaValues(flavor string, rhacs, release bool) charts.MetaValues {
	if rhacs {
		return charts.RHACSMetaValues()
	}
	switch strings.ToLower(flavor) {
	case flavorStackRoxIO:
		return charts.GetMetaValuesForFlavor(defaults.StackRoxIOReleaseImageFlavor())
	case flavorDevelopment:
		return charts.GetMetaValuesForFlavor(defaults.DevelopmentBuildImageFlavor())
	default:
		return charts.RHACSMetaValues()
	}
}

func validateFlavorFlags(rhacs bool, imageFlavor string) error {
	if rhacs && imageFlavor != "" {
		// TODO(RS-380): '--image-defaults' will be preferred (--rhacs deprecated) after we add RHACS flavor
		fmt.Fprintln(os.Stderr, "Warning: '--rhacs' has priority over '--image-defaults'")
	}
	if imageFlavor == "" || allowedFlavors.Contains(imageFlavor) {
		return nil
	}
	return fmt.Errorf("invalid value of '--image-defaults=%s', allowed values: %s", imageFlavor, allowedFlavors.ElementsString(", "))
}

// defaultFlavor provides default flavor for calls without --image-defaults
func defaultFlavor(flavor string) string {
	if flavor != "" {
		return flavor
	}
	if buildinfo.ReleaseBuild {
		return flavorStackRoxIO
	}
	return flavorDevelopment
}

func outputHelmChart(chartName string, outputDir string, removeOutputDir bool, rhacs bool, imageFlavor string, debug bool, debugChartPath string) error {
	// Lookup chart template prefix.
	chartTemplatePathPrefix := common.ChartTemplates[chartName]
	if chartTemplatePathPrefix == "" {
		return errors.New("unknown chart, see --help for list of supported chart names")
	}
	if err := validateFlavorFlags(rhacs, imageFlavor); err != nil {
		return err
	}
	metaVals := getMetaValues(defaultFlavor(imageFlavor), rhacs, buildinfo.ReleaseBuild)

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
func Command() *cobra.Command {
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
			return outputHelmChart(chartName, outputDir, removeOutputDir, rhacs, imageFlavor, debug, debugChartPath)
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
	c.PersistentFlags().StringVar(&imageFlavor, "image-defaults", "", "default container registry for container images")

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
