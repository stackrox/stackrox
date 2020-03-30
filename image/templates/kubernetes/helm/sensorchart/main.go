package main

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/utils"
)

// VersionInfo contains the main and collector version
type VersionInfo struct {
	MainVersion      string
	CollectorVersion string
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
		MainVersion:      args[0],
		CollectorVersion: args[1],
	}

	tmpDir := fmt.Sprintf("/tmp/%s", version.MainVersion)
	_, err := os.Stat(tmpDir)

	if err != nil {
		return errors.Wrapf(err, "directory %s expected to exist, but doesn't", tmpDir)
	}

	chartYaml := fmt.Sprintf("%s/Chart.yaml", dir)
	sensorYaml := fmt.Sprintf("%s/templates/sensor.yaml", dir)
	admissionControllerYaml := fmt.Sprintf("%s/templates/admission-controller.yaml", dir)

	tmpl := template.Must(template.New("").Delims("!!", "!!").
		ParseFiles(chartYaml, sensorYaml, admissionControllerYaml))

	err = utils.Should(renderTemplate(chartYaml, tmpl, version, tmpDir),
		renderTemplate(sensorYaml, tmpl, version, fmt.Sprintf("%s/templates", tmpDir)),
		renderTemplate(admissionControllerYaml, tmpl, version, fmt.Sprintf("%s/templates", tmpDir)))

	if err != nil {
		return err
	}
	return nil
}

func renderTemplate(path string, tmpl *template.Template, version VersionInfo, destDir string) error {
	chartOutputFile, err := os.Create(fmt.Sprintf("%s/%s", destDir, filepath.Base(path)))
	if err != nil {
		return err
	}
	defer closeFile(chartOutputFile)

	err = tmpl.ExecuteTemplate(chartOutputFile, filepath.Base(path), version)
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
