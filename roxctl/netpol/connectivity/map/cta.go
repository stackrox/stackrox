// Package connectivitymap provides primitives for command 'roxctl netpol connectivity map'
package connectivitymap

import (
	"os"
	"path/filepath"

	npguard "github.com/np-guard/netpol-analyzer/pkg/netpol/connlist"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/npg"
)

const (
	defaultOutputFileNamePrefix = "connlist."
	defaultOutputFormat         = "txt"
)

type netpolAnalyzer interface {
	ConnlistFromDirPath(dirPath string) ([]npguard.Peer2PeerConnection, []npguard.Peer, error)
	ConnectionsListToString(conns []npguard.Peer2PeerConnection) (string, error)
	Errors() []npguard.ConnlistError
}

// Cmd represents netpol connectivity map command
type Cmd struct {
	// Properties that are bound to cobra flags.
	stopOnFirstError      bool
	treatWarningsAsErrors bool
	inputFolderPath       string
	outputFilePath        string
	removeOutputPath      bool
	outputToFile          bool
	focusWorkload         string
	outputFormat          string

	// injected or constructed values
	env environment.Environment
}

// NewCmd constructs the command
func NewCmd(env environment.Environment) *Cmd {
	return &Cmd{env: env}
}

func (cmd *Cmd) construct(args []string) (netpolAnalyzer, error) {
	cmd.inputFolderPath = args[0]
	var opts []npguard.ConnlistAnalyzerOption
	if cmd.env != nil && cmd.env.Logger() != nil {
		opts = append(opts, npguard.WithLogger(npg.NewLogger(cmd.env.Logger())))
	}
	if cmd.stopOnFirstError {
		opts = append(opts, npguard.WithStopOnError())
	}
	if cmd.focusWorkload != "" {
		opts = append(opts, npguard.WithFocusWorkload(cmd.focusWorkload))
	}
	if cmd.outputFormat != "" {
		opts = append(opts, npguard.WithOutputFormat(cmd.outputFormat))
	}
	if cmd.outputFilePath != "" {
		cmd.outputToFile = true
	}
	return npguard.NewConnlistAnalyzer(opts...), nil
}

func (cmd *Cmd) validate() error {
	if err := cmd.setupPath(cmd.outputFilePath); err != nil {
		return errors.Wrap(err, "failed to set up file path")
	}
	return nil
}

func (cmd *Cmd) setupPath(path string) error {
	if _, err := os.Stat(path); err == nil && !cmd.removeOutputPath {
		return errox.AlreadyExists.Newf("path %s already exists. Use --remove to overwrite or select a different path.", path)
	} else if !os.IsNotExist(err) {
		return errors.Wrapf(err, "failed to check if path %s exists", path)
	}
	return nil
}

// RunE executes the command and returns potential errors
func (cmd *Cmd) RunE(_ *cobra.Command, args []string) error {
	cmd.env.Logger().WarnfLn("This is a Technology Preview feature. Red Hat does not recommend using Technology Preview features in production.")
	analyzer, err := cmd.construct(args)
	if err != nil {
		return err
	}
	if err := cmd.validate(); err != nil {
		return err
	}
	return cmd.analyzeNetpols(analyzer)
}

// AddFlags is for parsing flags and storing their values
func (cmd *Cmd) AddFlags(c *cobra.Command) *cobra.Command {
	c.Flags().BoolVar(&cmd.treatWarningsAsErrors, "strict", false, "treat warnings as errors")
	c.Flags().BoolVar(&cmd.stopOnFirstError, "fail", false, "fail on the first encountered error")
	c.Flags().BoolVar(&cmd.removeOutputPath, "remove", false, "remove the output path if it already exists")
	c.Flags().BoolVar(&cmd.outputToFile, "save-to-file", false, "whether to save connections list output into default file")
	c.Flags().StringVarP(&cmd.outputFilePath, "output-file", "f", "", "save connections list output into specific file")
	c.Flags().StringVarP(&cmd.focusWorkload, "focus-workload", "", "", "focus on connections of specified workload name in the output")
	c.Flags().StringVarP(&cmd.outputFormat, "output-format", "o", defaultOutputFormat, "configure the connections list in specific format, supported formats: txt|json|md|dot|csv")
	return c
}

func (cmd *Cmd) analyzeNetpols(analyzer netpolAnalyzer) error {
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

func (cmd *Cmd) ouputConnList(connsStr string) error {
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

func (cmd *Cmd) printConnList(connlist string) {
	cmd.env.Logger().PrintfLn(connlist)
}

func (cmd *Cmd) getDefaultFileName() string {
	if cmd.outputFormat == "" {
		return defaultOutputFileNamePrefix + defaultOutputFormat
	}
	return defaultOutputFileNamePrefix + cmd.outputFormat
}
