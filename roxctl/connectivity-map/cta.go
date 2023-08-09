// Package connectivitymap provides primitives for command 'roxctl connectivity-map'
package connectivitymap

import (
	"os"
	"path/filepath"

	npgconnlist "github.com/np-guard/netpol-analyzer/pkg/netpol/connlist"
	npgeval "github.com/np-guard/netpol-analyzer/pkg/netpol/eval"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/roxctl/common/npg"
)

const (
	defaultOutputFileNamePrefix = "connlist."
	defaultOutputFormat         = "txt"
)

type netpolAnalyzer interface {
	ConnlistFromDirPath(dirPath string) ([]npgconnlist.Peer2PeerConnection, []npgeval.Peer, error)
	ConnectionsListToString(conns []npgconnlist.Peer2PeerConnection) (string, error)
	Errors() []npgconnlist.ConnlistError
}

func (cmd *analyzeNetpolCommand) analyzeNetpols(analyzer netpolAnalyzer) error {
	conns, _, err := analyzer.ConnlistFromDirPath(cmd.inputFolderPath)
	if err != nil {
		return errors.Wrap(err, "error in connectivity analysis")
	}
	connsStr, err := analyzer.ConnectionsListToString(conns)
	if err != nil {
		return errors.Wrap(err, "error in formatting connectivity list")
	}
	if err := cmd.ouputConnList(connsStr); err != nil {
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

func (cmd *analyzeNetpolCommand) ouputConnList(connsStr string) error {
	if cmd.outputToFile {
		if cmd.outputFilePath == "" { // save-to-file is true, but output file path is not provided
			cmd.outputFilePath = cmd.getDefaultFileName()
		}

		if err := writeFile(cmd.outputFilePath, connsStr); err != nil {
			return errors.Wrap(err, "error writing connlist output")
		}
	}

	cmd.printConnList(connsStr)
	return nil
}

func writeFile(outputPath string, content string) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return errors.Wrapf(err, "error creating directory for file %q", outputPath)
	}
	return errors.Wrap(os.WriteFile(outputPath, []byte(content), os.FileMode(0644)), "error writing file")
}

func (cmd *analyzeNetpolCommand) printConnList(connlist string) {
	cmd.env.Logger().PrintfLn(connlist)
}

func (cmd *analyzeNetpolCommand) getDefaultFileName() string {
	if cmd.outputFormat == "" {
		return defaultOutputFileNamePrefix + defaultOutputFormat
	}
	return defaultOutputFileNamePrefix + cmd.outputFormat
}
