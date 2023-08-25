package logconvert

import (
	"bufio"
	"io"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
)

var (
	module     string
	levelLabel string
)

// Command defines the log-convert command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:    "log-convert",
		Short:  "Read messages line by line from stdin and log them via the default logging facilities",
		Hidden: true,
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			level, ok := logging.LevelForLabel(levelLabel)
			if !ok {
				return errox.InvalidArgs.Newf("unknown level %s", levelLabel)
			}
			if level > logging.WarnLevel {
				return errox.InvalidArgs.New("only non-destructive log levels are supported")
			}

			scanner := bufio.NewScanner(cliEnvironment.InputOutput().In())
			logger := logging.ModuleForName(module).Logger()

			for scanner.Scan() {
				logger.Log(level, scanner.Text())
			}

			switch err := scanner.Err(); err {
			case io.EOF:
				return nil
			default:
				return errors.Wrap(err, "scanner error")
			}
		}),
	}

	c.Flags().StringVar(&module, "module", "logconvert", "Specifies the module for logging purposes")
	c.Flags().StringVar(&levelLabel, "level", "info", "Specifies the log level in {error, warn, info, debug}")

	flags.HideInheritedFlags(c)

	return c
}
