// Package connectivitydiff provides primitives for command 'roxctl connectivity-diff'
package connectivitydiff

import (
	"os"
	"path/filepath"

	npguard "github.com/np-guard/netpol-analyzer/pkg/netpol/diff"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/roxctl/common/npg"
)

const (
	defaultOutputFileNamePrefix = "connectivity_diff."
	defaultOutputFormat         = "txt"
)

type diffAnalyzer interface {
	ConnDiffFromDirPaths(dirPath1, dirPath2 string) (npguard.ConnectivityDiff, error)
	ConnectivityDiffToString(connectivityDiff npguard.ConnectivityDiff) (string, error)
	Errors() []npguard.DiffError
}

func (cmd *diffNetpolCommand) analyzeConnectivityDiff(analyzer diffAnalyzer) error {
	connsDiff, err := analyzer.ConnDiffFromDirPaths(cmd.inputFolderPath1, cmd.inputFolderPath2)
	if err != nil { // returned with fatal error
		return errors.Wrap(err, "error in connectivity diff analysis")
	}
	severeErr, warnErr := cmd.checkAnalyzerErrors(analyzer)
	if cmd.stopOnFirstError && severeErr != nil {
		return severeErr
	}

	connsDiffStr, err := analyzer.ConnectivityDiffToString(connsDiff)
	if err != nil {
		return errors.Wrap(err, "error in formatting connectivity diff")
	}

	if err := cmd.ouputConnsDiff(connsDiffStr); err != nil {
		return err
	}

	if cmd.treatWarningsAsErrors && severeErr == nil {
		return warnErr
	}
	return severeErr
}

func (cmd *diffNetpolCommand) checkAnalyzerErrors(analyzer diffAnalyzer) (severeErr, warnErr error) {
	var severe, warning error
	for _, e := range analyzer.Errors() {
		if e.IsSevere() {
			cmd.env.Logger().ErrfLn("%s %s", e.Error(), e.Location())
			severe = npg.ErrErrors
		} else {
			cmd.env.Logger().WarnfLn("%s %s", e.Error(), e.Location())
			warning = npg.ErrWarnings
		}
	}
	return severe, warning
}

func (cmd *diffNetpolCommand) ouputConnsDiff(connsDiffStr string) error {
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
