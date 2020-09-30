package output

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/charts"
	"github.com/stackrox/rox/pkg/helmtpl"
	"github.com/stackrox/rox/pkg/helmutil"
	"helm.sh/helm/v3/pkg/chart/loader"
)

// These are actually const.

var (
	chartTemplates = map[string]string{
		"central-services": image.CentralServicesChartPrefix,
	}
	prettyChartNameList string
)

// Initialize `prettyChartNameList` for usage information.
func init() {
	chartTemplateNames := make([]string, 0, len(chartTemplates))
	for name := range chartTemplates {
		chartTemplateNames = append(chartTemplateNames, name)
	}
	sort.Strings(chartTemplateNames)
	prettyChartNameList = strings.Join(chartTemplateNames, " | ")
}

func outputHelmChart(chartName string, outputDir string) error {
	// Lookup chart template prefix.
	chartTemplatePathPrefix := chartTemplates[chartName]
	if chartTemplatePathPrefix == "" {
		return errors.New("unknown chart, see --help for list of supported chart names")
	}

	if outputDir == "" {
		outputDir = fmt.Sprintf("./stackrox-%s-chart", chartName)
		fmt.Fprintf(os.Stderr, "No output directory specified, using default directory %q.\n", outputDir)
	}

	if _, err := os.Stat(outputDir); err == nil {
		fmt.Fprintf(os.Stderr, "Directory %q already exists, use --output-dir to select a different directory.\n", outputDir)
		return fmt.Errorf("directory %q already exists", outputDir)
	} else if !os.IsNotExist(err) {
		return errors.Wrapf(err, "failed to check if directory %q exists", outputDir)
	}

	// Retrieve template files from box.
	chartTplFiles, err := image.GetFilesFromBox(image.K8sBox, chartTemplatePathPrefix)
	if err != nil {
		return errors.Wrapf(err, "fetching %s chart files from box", chartName)
	}
	chartTpl, err := helmtpl.Load(chartTplFiles)
	if err != nil {
		return errors.Wrapf(err, "loading %s helmtpl", chartName)
	}

	// Render template files.
	renderedChartFiles, err := chartTpl.InstantiateRaw(charts.DefaultMetaValues())
	if err != nil {
		return errors.Wrapf(err, "instantiating %s helmtpl", chartName)
	}

	// Apply .helmignore filtering rules, to be on the safe side (but keep .helmignore).
	renderedChartFiles, err = helmutil.FilterFiles(renderedChartFiles)
	if err != nil {
		return errors.Wrap(err, "filtering instantiated helm chart files")
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

	c := &cobra.Command{
		Use: fmt.Sprintf("output <%s>", prettyChartNameList),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("incorrect number of arguments, see --help for usage information")
			}
			chartName := args[0]
			return outputHelmChart(chartName, outputDir)

		},
	}
	c.PersistentFlags().StringVar(&outputDir, "output-dir", "", "path to the output directory for Helm chart (default: './stackrox-<chart name>-chart')")

	return c
}

func writeFile(file *loader.BufferedFile, destDir string) error {
	outputPath := filepath.Join(destDir, filepath.FromSlash(file.Name))
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return errors.Wrapf(err, "creating directory for file %q", file.Name)
	}

	perms := os.FileMode(0644)
	return ioutil.WriteFile(outputPath, file.Data, perms)
}
