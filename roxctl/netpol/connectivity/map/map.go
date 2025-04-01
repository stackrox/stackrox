// Package connectivitymap provides primitives for command 'roxctl netpol connectivity map'
package connectivitymap

import (
	npguard "github.com/np-guard/netpol-analyzer/pkg/netpol/connlist"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/npg"
	"github.com/stackrox/rox/roxctl/netpol/resources"
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
	if cmd.exposure {
		opts = append(opts, npguard.WithExposureAnalysis())
	}
	if cmd.explain {
		opts = append(opts, npguard.WithExplanation())
	}
	return npguard.NewConnlistAnalyzer(opts...), nil
}

func (cmd *Cmd) analyze(analyzer netpolAnalyzer) (w []error, e []error) {
	infos, warns, errs := resources.GetK8sInfos(cmd.inputFolderPath, cmd.stopOnFirstError, cmd.treatWarningsAsErrors)
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
	w, e = npg.HandleNPGuardErrors(analyzer.Errors())
	return append(warns, w...), append(errs, e...)
}

func (cmd *Cmd) ouputConnList(connsStr string) error {
	if cmd.outputToFile {
		if cmd.outputFilePath == "" { // save-to-file is true, but output file path is not provided
			cmd.outputFilePath = cmd.getDefaultFileName()
		}

		if err := npg.WriteFile(cmd.outputFilePath, connsStr); err != nil {
			return errors.Wrap(err, "error writing connlist output")
		}
	}

	cmd.env.Logger().PrintfLn(connsStr)
	return nil
}

func (cmd *Cmd) getDefaultFileName() string {
	if cmd.outputFormat == "" {
		return defaultOutputFileNamePrefix + defaultOutputFormat
	}
	return defaultOutputFileNamePrefix + cmd.outputFormat
}
