// Package diff provides primitives for command 'roxctl connectivity diff'
package diff

import (
	"os"

	npgdiff "github.com/np-guard/netpol-analyzer/pkg/netpol/diff"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/roxctl/common/npg"
	"github.com/stackrox/rox/roxctl/netpol/resources"
	"k8s.io/cli-runtime/pkg/resource"
)

const (
	defaultOutputFileNamePrefix = "connectivity_diff."
	defaultOutputFormat         = "txt"
)

type diffAnalyzer interface {
	ConnDiffFromResourceInfos(infos1, infos2 []*resource.Info) (npgdiff.ConnectivityDiff, error)
	ConnectivityDiffToString(connectivityDiff npgdiff.ConnectivityDiff) (string, error)
	Errors() []npgdiff.DiffError
}

func (cmd *diffNetpolCommand) processInput() (info1 []*resource.Info, info2 []*resource.Info, warnings []error, errs []error) {
	infos1, warns1, errs1 := resources.GetK8sInfos(cmd.inputFolderPath1, cmd.stopOnFirstError, cmd.treatWarningsAsErrors)
	infos2, warns2, errs2 := resources.GetK8sInfos(cmd.inputFolderPath2, cmd.stopOnFirstError, cmd.treatWarningsAsErrors)
	return infos1, infos2, append(warns1, warns2...), append(errs1, errs2...)
}

func (cmd *diffNetpolCommand) analyzeConnectivityDiff(analyzer diffAnalyzer) (w []error, e []error) {
	info1, info2, warns, errs := cmd.processInput()
	if cmd.stopOnFirstError && (len(errs) > 0 || (len(warns) > 0 && cmd.treatWarningsAsErrors)) {
		return warns, errs
	}

	connsDiff, err := analyzer.ConnDiffFromResourceInfos(info1, info2)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return warns, append(errs, errox.NotFound.New(err.Error()))
		}
		return warns, append(errs, errors.Wrap(err, "connectivity diff analysis"))
	}
	connsDiffStr, err := analyzer.ConnectivityDiffToString(connsDiff)
	if err != nil {
		return warns, append(errs, errors.Wrap(err, "formatting connectivity diff"))
	}
	if err := cmd.outputConnsDiff(connsDiffStr); err != nil {
		return warns, append(errs, err)
	}

	w, e = npg.HandleNPGuardErrors(analyzer.Errors())
	return append(warns, w...), append(errs, e...)
}

func (cmd *diffNetpolCommand) outputConnsDiff(connsDiffStr string) error {
	if cmd.outputToFile {
		if cmd.outputFilePath == "" { // save-to-file is true, but output file path is not provided
			cmd.outputFilePath = cmd.getDefaultFileName()
		}

		if err := npg.WriteFile(cmd.outputFilePath, connsDiffStr); err != nil {
			return errors.Wrap(err, "error writing connections diff output")
		}
	}

	cmd.env.Logger().PrintfLn(connsDiffStr)
	return nil
}

func (cmd *diffNetpolCommand) getDefaultFileName() string {
	fileNamePrefix := defaultOutputFileNamePrefix
	if cmd.outputFormat == "" {
		return fileNamePrefix + defaultOutputFormat
	}
	return fileNamePrefix + cmd.outputFormat
}
