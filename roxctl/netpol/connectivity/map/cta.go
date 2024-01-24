// Package connectivitymap provides primitives for command 'roxctl netpol connectivity map'
package connectivitymap

import (
	goerrors "errors"
	"fmt"
	"os"
	"path/filepath"

	npguard "github.com/np-guard/netpol-analyzer/pkg/netpol/connlist"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/npg"
	"github.com/stackrox/rox/roxctl/netpol/netpolerrors"
	"k8s.io/cli-runtime/pkg/resource"
)

const (
	defaultOutputFileNamePrefix = "connlist."
	defaultOutputFormat         = "txt"
)

type netpolAnalyzer interface {
	ConnlistFromResourceInfos(info []*resource.Info) ([]npguard.Peer2PeerConnection, []npguard.Peer, error)
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

func getInfoObj(path string, failFast, treatWarningsAsErrors bool) ([]*resource.Info, error) {
	b := resource.NewLocalBuilder().
		Unstructured().
		FilenameParam(false,
			&resource.FilenameOptions{Filenames: []string{path}, Recursive: true}).
		Flatten()
	// only for the combination of --fail & --strict, should not run with ContinueOnError, and stop on first warning.
	// the only error which is not warning returned from this call is errox.NotFound, for which it already fails fast.
	if !(failFast && treatWarningsAsErrors) {
		b.ContinueOnError()
	}
	//nolint:wrapcheck // we do wrap the errors later in `errHandler.HandleErrors`
	return b.Do().Infos()
}

func (cmd *Cmd) analyzeNetpols(analyzer netpolAnalyzer) error {
	warns, errs := cmd.analyze(analyzer)
	if cmd.treatWarningsAsErrors {
		return goerrors.Join(goerrors.Join(warns...), goerrors.Join(errs...))
	}
	for _, warn := range warns {
		cmd.env.Logger().WarnfLn("%v", warn)
	}
	return goerrors.Join(errs...)
}

func (cmd *Cmd) analyze(analyzer netpolAnalyzer) (w []error, e []error) {
	errHandler := netpolerrors.NewErrHandler(cmd.treatWarningsAsErrors)
	infos, err := getInfoObj(cmd.inputFolderPath, cmd.stopOnFirstError, cmd.treatWarningsAsErrors)
	warns, errs := errHandler.HandleError(err)
	if cmd.stopOnFirstError && (len(errs) > 0 || (len(warns) > 0 && cmd.treatWarningsAsErrors)) {
		return warns, errs
	}

	conns, _, err := analyzer.ConnlistFromResourceInfos(infos)
	if err != nil {
		return warns, append(errs, errors.Wrap(err, "connectivity analysis"))
	}
	connsStr, err := analyzer.ConnectionsListToString(conns)
	if err != nil {
		return warns, append(errs, errors.Wrap(err, "formatting connectivity list"))
	}
	if err := cmd.ouputConnList(connsStr); err != nil {
		return warns, append(errs, errors.Wrap(err, "writing connectivity result"))
	}
	var roxerr error
	for _, err := range analyzer.Errors() {
		if err.IsSevere() {
			errs = append(errs, fmt.Errorf("%s %s", err.Error(), err.Location()))
			roxerr = npg.ErrErrors
		} else {
			warns = append(warns, fmt.Errorf("%s %s", err.Error(), err.Location()))
			if cmd.treatWarningsAsErrors && roxerr == nil {
				roxerr = npg.ErrWarnings
			}
		}
	}
	if roxerr != nil {
		errs = append(errs, roxerr)
	}
	return warns, errs
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
