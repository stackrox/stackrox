package main

import (
	"fmt"
	"os"
	"text/template"

	"github.com/pkg/errors"
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

	tmpl := template.Must(template.New("").Delims("!!", "!!").
		ParseFiles(fmt.Sprintf("%s/templates/sensor.yaml", dir),
			fmt.Sprintf("%s/Chart.yaml", dir)))

	chartOutputFile, err := os.Create(fmt.Sprintf("%s/Chart.yaml", tmpDir))
	if err != nil {
		return err
	}
	defer closeFile(chartOutputFile)

	err = tmpl.ExecuteTemplate(chartOutputFile, "Chart.yaml", version)
	if err != nil {
		return err
	}

	sensorOutputFile, err := os.Create(fmt.Sprintf("%s/templates/sensor.yaml", tmpDir))
	if err != nil {
		return err
	}
	defer closeFile(sensorOutputFile)

	err = tmpl.ExecuteTemplate(sensorOutputFile, "sensor.yaml", version)
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
