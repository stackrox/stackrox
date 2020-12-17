package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/image"
	"github.com/stackrox/rox/pkg/devbuild"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/helmutil"
	"github.com/stackrox/rox/pkg/roxctl/defaults"
	"github.com/stackrox/rox/pkg/version"
	"helm.sh/helm/v3/pkg/chart/loader"
)

func main() {
	args := os.Args[1:]

	if err := mainCmd(args); err != nil {
		fmt.Fprintf(os.Stderr, "helm templating: %v\n", err)
		os.Exit(1)
	}
}

func mainCmd(args []string) error {
	if len(args) != 3 {
		return fmt.Errorf("incorrect number of arguments, found %d, expected 3", len(args))
	}

	imageTag, collectorImageTag, outputDir := args[0], args[1], args[2]
	metaValues := map[string]interface{}{
		"Versions": version.Versions{
			CollectorVersion: collectorImageTag,
			MainVersion:      imageTag,
			ChartVersion:     version.DeriveChartVersion(imageTag),
		},
		"MainRegistry":        defaults.MainImageRegistry(),
		"CollectorRegistry":   defaults.CollectorImageRegistry(),
		"ImageTag":            imageTag,
		"CollectorImageTag":   collectorImageTag,
		"RenderAsLegacyChart": true,
	}

	if _, err := os.Stat(outputDir); err != nil {
		return errors.Wrapf(err, "directory %s expected to exist, but doesn't", outputDir)
	}

	if devbuild.IsEnabled() {
		featureFlagVals := make(map[string]interface{})
		for _, feature := range features.Flags {
			featureFlagVals[feature.EnvVar()] = strconv.FormatBool(feature.Enabled())
		}
		metaValues["FeatureFlags"] = featureFlagVals
	}

	chartTpl, err := image.GetSensorChartTemplate(image.K8sBox)
	if err != nil {
		return errors.Wrap(err, "loading sensor helmtpl")
	}

	chartFiles, err := chartTpl.InstantiateRaw(metaValues)
	if err != nil {
		return errors.Wrap(err, "instantiating sensor helmtpl")
	}

	// Apply .helmignore filtering rules, to be on the safe side (but keep .helmignore).
	chartFiles, err = helmutil.FilterFiles(chartFiles)
	if err != nil {
		return errors.Wrap(err, "filtering instantiated helm chart files")
	}

	for _, f := range chartFiles {
		if err := writeFile(f, outputDir); err != nil {
			return errors.Wrapf(err, "error writing file %q", f.Name)
		}
	}

	return nil
}

func writeFile(file *loader.BufferedFile, destDir string) error {
	outputPath := filepath.Join(destDir, filepath.FromSlash(file.Name))
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return errors.Wrapf(err, "creating directory for file %q", file.Name)
	}

	perms := os.FileMode(0644)
	if path.Ext(file.Name) == ".sh" {
		perms |= 0111
	}
	return ioutil.WriteFile(outputPath, file.Data, perms)
}
