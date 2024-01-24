// Package diff provides primitives for command 'roxctl connectivity diff'
package diff

import (
	goerrors "errors"
	"os"
	"path/filepath"

	npgdiff "github.com/np-guard/netpol-analyzer/pkg/netpol/diff"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/roxctl/common/npg"
	"github.com/stackrox/rox/roxctl/netpol/netpolerrors"
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

func getInfoObjs(path string, failFast bool, treatWarningsAsErrors bool) ([]*resource.Info, error) {
	b := resource.NewLocalBuilder().
		Unstructured()
	if !(failFast && treatWarningsAsErrors) {
		b.ContinueOnError()
	}
	//nolint:wrapcheck // we do wrap the errors later in ErrorHandler
	return b.Path(true, path).Do().IgnoreErrors().Infos()
}

func (cmd *diffNetpolCommand) processInput() (info1 []*resource.Info, info2 []*resource.Info, warnings error, errs error) {
	info1, err1 := getInfoObjs(cmd.inputFolderPath1, cmd.stopOnFirstError, cmd.treatWarningsAsErrors)
	info2, err2 := getInfoObjs(cmd.inputFolderPath2, cmd.stopOnFirstError, cmd.treatWarningsAsErrors)
	inputErrHandler := netpolerrors.NewErrHandler(cmd.treatWarningsAsErrors)
	w, e := inputErrHandler.HandleErrorPair(err1, err2)
	return info1, info2, goerrors.Join(w...), goerrors.Join(e...)
}

func (cmd *diffNetpolCommand) analyzeConnectivityDiff(analyzer diffAnalyzer) error {
	info1, info2, warnings, err := cmd.processInput()
	if err != nil {
		return err
	}
	if warnings != nil {
		cmd.env.Logger().WarnfLn("%v", warnings)
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
