// Package diff provides primitives for command 'roxctl connectivity diff'
package diff

import (
	"os"
	"path/filepath"

	npgdiff "github.com/np-guard/netpol-analyzer/pkg/netpol/diff"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/roxctl/common/npg"
	"k8s.io/cli-runtime/pkg/resource"
)

const (
	defaultOutputFileNamePrefix = "connectivity_diff."
	defaultOutputFormat         = "txt"
)

var (
	ErrYAMLMalformed = errors.New("YAML document is malformed")
	ErrYAMLIsNotK8s  = errors.New("YAML document does not represent a K8s resource")
)

type diffAnalyzer interface {
	ConnDiffFromResourceInfos(infos1, infos2 []*resource.Info) (npgdiff.ConnectivityDiff, error)
	ConnectivityDiffToString(connectivityDiff npgdiff.ConnectivityDiff) (string, error)
	Errors() []npgdiff.DiffError
}

func getInfoObj(path string, failFast bool) ([]*resource.Info, error) {
	b := resource.NewLocalBuilder().
		Unstructured().
		FilenameParam(false,
			&resource.FilenameOptions{Filenames: []string{path}, Recursive: true}).
		Flatten()
	if !failFast {
		b.ContinueOnError()
	}
	return b.Do().Infos()
}

func (cmd *diffNetpolCommand) analyzeConnectivityDiff(analyzer diffAnalyzer) error {
	errHandler := NewErrHandler(cmd.treatWarningsAsErrors)
	info1, err1 := getInfoObj(cmd.inputFolderPath1, cmd.stopOnFirstError)
	info2, err2 := getInfoObj(cmd.inputFolderPath2, cmd.stopOnFirstError)
	if err := errHandler.HandleErrors(err1, err2); err != nil {
		return err
	}

	connsDiff, err := analyzer.ConnDiffFromResourceInfos(info1, info2)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return errox.NotFound.Newf(err.Error())
		}
		return errors.Wrap(err, "error in connectivity diff analysis")
	}
	connsDiffStr, err := analyzer.ConnectivityDiffToString(connsDiff)
	if err != nil {
		return errors.Wrap(err, "error in formatting connectivity diff")
	}
	if err := cmd.outputConnsDiff(connsDiffStr); err != nil {
		return err
	}
	var roxerr error
	for _, e := range analyzer.Errors() {
		if e.IsSevere() {
			cmd.env.Logger().ErrfLn("%s %s", e.Error(), e.Location())
			roxerr = npg.ErrErrors
		} else {
			cmd.env.Logger().WarnfLn("%s %s", e.Error(), e.Location())
			if cmd.treatWarningsAsErrors && roxerr == nil {
				roxerr = npg.ErrWarnings
			}
		}
	}
	return roxerr
}

func (cmd *diffNetpolCommand) outputConnsDiff(connsDiffStr string) error {
	if cmd.outputToFile {
		if cmd.outputFilePath == "" { // save-to-file is true, but output file path is not provided
			cmd.outputFilePath = cmd.getDefaultFileName()
		}

		if err := writeFile(cmd.outputFilePath, connsDiffStr); err != nil {
			return errors.Wrap(err, "error writing connections diff output")
		}
	}

	cmd.printConnsDiff(connsDiffStr)
	return nil
}

func writeFile(outputPath string, content string) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return errors.Wrapf(err, "error creating directory for file %q", outputPath)
	}
	return errors.Wrap(os.WriteFile(outputPath, []byte(content), os.FileMode(0644)), "error writing file")
}

func (cmd *diffNetpolCommand) printConnsDiff(connsDiff string) {
	cmd.env.Logger().PrintfLn(connsDiff)
}

func (cmd *diffNetpolCommand) getDefaultFileName() string {
	fileNamePrefix := defaultOutputFileNamePrefix
	if cmd.outputFormat == "" {
		return fileNamePrefix + defaultOutputFormat
	}
	return fileNamePrefix + cmd.outputFormat
}
