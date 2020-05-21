package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/devbuild"
	"github.com/stackrox/rox/pkg/features"
	renderutils "github.com/stackrox/rox/pkg/renderer/utils"
	"github.com/stackrox/rox/pkg/utils"
	"gopkg.in/yaml.v2"
)

// VersionInfo contains the main and collector version
type VersionInfo struct {
	ImageTag          string
	CollectorImageTag string
}

type envVars struct {
	Env []envVar `yaml:"envVars,omitempty"`
}

type envVar struct {
	Name  string `yaml:"name,omitempty"`
	Value string `yaml:"value,omitempty"`
}

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
	dir := args[2]

	version := VersionInfo{
		ImageTag:          args[0],
		CollectorImageTag: fmt.Sprintf("%s-latest", args[1]),
	}

	tmpDir := fmt.Sprintf("/tmp/%s", version.ImageTag)
	_, err := os.Stat(tmpDir)

	if err != nil {
		return errors.Wrapf(err, "directory %s expected to exist, but doesn't", tmpDir)
	}

	if devbuild.IsEnabled() {
		ev := envVars{}
		for _, feature := range features.Flags {
			ev.Env = append(ev.Env, envVar{feature.EnvVar(), strconv.FormatBool(feature.Enabled())})
		}

		yamlBytes, err := yaml.Marshal(ev)
		if err != nil {
			return errors.Wrap(err, "feature flags cannot be processed")
		}

		err = ioutil.WriteFile(fmt.Sprintf("%s/feature-flag-values.yaml", tmpDir), yamlBytes, 0644)
		if err != nil {
			return errors.Wrap(err, "feature flags cannot be processed")
		}
	}

	chartYaml := fmt.Sprintf("%s/Chart.yaml", dir)
	sensorYaml := fmt.Sprintf("%s/templates/sensor.yaml", dir)
	admissionControllerYaml := fmt.Sprintf("%s/templates/admission-controller.yaml", dir)
	valuesYaml := fmt.Sprintf("%s/values.yaml", dir)

	tmpl := template.Must(template.New("").Delims("!!", "!!").
		Funcs(renderutils.BuiltinFuncs).Funcs(sprig.TxtFuncMap()).
		ParseFiles(chartYaml, sensorYaml, admissionControllerYaml, valuesYaml))

	err = utils.Should(renderTemplate(chartYaml, tmpl, version, tmpDir),
		renderTemplate(sensorYaml, tmpl, version, fmt.Sprintf("%s/templates", tmpDir)),
		renderTemplate(admissionControllerYaml, tmpl, version, fmt.Sprintf("%s/templates", tmpDir)),
		renderTemplate(valuesYaml, tmpl, nil, tmpDir),
	)

	if err != nil {
		return err
	}

	return nil
}

func renderTemplate(path string, tmpl *template.Template, params interface{}, destDir string) error {
	chartOutputFile, err := os.Create(fmt.Sprintf("%s/%s", destDir, filepath.Base(path)))
	if err != nil {
		return err
	}
	defer closeFile(chartOutputFile)

	err = tmpl.ExecuteTemplate(chartOutputFile, filepath.Base(path), params)
	if err != nil {
		return err
	}
	return nil
}

func closeFile(f *os.File) {
	err := f.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error closing file %s: %v\n", f.Name(), err)
		os.Exit(1)
	}
}
