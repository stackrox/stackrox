package debug

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
)

var (
	levels = getValidLevels()

	levelList = strings.Join(levels, " | ")
)

type centralDebugLogLevelCommand struct {
	// Properties that are bound to cobra flags.
	level   string
	modules []string

	// Properties that are injected or constructed.
	env          environment.Environment
	timeout      time.Duration
	retryTimeout time.Duration
}

// Command defines the debug command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	c := &cobra.Command{
		Use:   "debug",
		Short: "Commands for debugging the Central service",
	}
	c.AddCommand(logLevelCommand(cliEnvironment))
	c.AddCommand(dumpCommand(cliEnvironment))
	c.AddCommand(downloadDiagnosticsCommand(cliEnvironment))
	c.AddCommand(authzTraceCommand(cliEnvironment))
	if env.ResyncDisabled.BooleanSetting() {
		c.AddCommand(resyncCheckCommand(cliEnvironment))
	}
	return c
}

// logLevelCommand allows getting and setting the Log Level for StackRox services.
func logLevelCommand(cliEnvironment environment.Environment) *cobra.Command {
	levelCmd := &centralDebugLogLevelCommand{env: cliEnvironment}

	c := &cobra.Command{
		Use:   "log",
		Short: `"log" to get current log level; "log --level=<level>" to set log level`,
		Long:  `"log" to get current log level; "log --level=<level>" to set log level`,
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			levelCmd.timeout = flags.Timeout(c)
			levelCmd.retryTimeout = flags.RetryTimeout(c)
			if levelCmd.level == "" {
				return levelCmd.getLogLevel()
			}
			return levelCmd.setLogLevel()
		}),
	}
	c.Flags().StringVarP(&levelCmd.level, "level", "l", "",
		fmt.Sprintf("the log level to set the modules to (%s) ", levelList))
	c.Flags().StringSliceVarP(&levelCmd.modules, "modules", "m", nil, "the modules to which to apply the command")
	flags.AddTimeout(c)
	flags.AddRetryTimeout(c)
	return c
}

func (cmd *centralDebugLogLevelCommand) getLogLevel() error {
	conn, err := cmd.env.GRPCConnection(cmd.retryTimeout)
	if err != nil {
		return err
	}
	defer func() {
		_ = conn.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), cmd.timeout)
	defer cancel()

	client := v1.NewDebugServiceClient(conn)
	logResponse, err := client.GetLogLevel(ctx, &v1.GetLogLevelRequest{Modules: cmd.modules})
	if err != nil {
		return errors.Wrap(err, "could not get log level from central")
	}

	cmd.printGetLogLevelResponse(logResponse)
	return nil
}

func (cmd *centralDebugLogLevelCommand) printGetLogLevelResponse(r *v1.LogLevelResponse) {
	const rowFormat = "%-40s  %s"
	indent := ""
	if r.GetLevel() != "" {
		cmd.env.Logger().PrintfLn("Current log level is %s", r.GetLevel())
		if len(r.GetModuleLevels()) > 0 {
			cmd.env.Logger().PrintfLn("Modules with a different log level:")
			indent = "  "
		}
	}
	if len(r.GetModuleLevels()) > 0 {
		cmd.env.Logger().PrintfLn(indent+rowFormat, "Module", "Level")
		cmd.env.Logger().PrintfLn("")
		for _, modLvl := range r.GetModuleLevels() {
			cmd.env.Logger().PrintfLn(indent+rowFormat, modLvl.GetModule(), modLvl.GetLevel())
		}
	}
}

func (cmd *centralDebugLogLevelCommand) setLogLevel() error {
	conn, err := cmd.env.GRPCConnection(cmd.retryTimeout)
	if err != nil {
		return err
	}
	defer func() {
		_ = conn.Close()
	}()

	client := v1.NewDebugServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), cmd.timeout)
	defer cancel()

	_, err = client.SetLogLevel(ctx, &v1.LogLevelRequest{Level: cmd.level, Modules: cmd.modules})
	if err != nil {
		return errors.Wrap(err, "could not set log level on central")
	}

	cmd.env.Logger().PrintfLn("Successfully set log level")
	return nil
}

// getValidLevels return level strings in ascending severity order.
func getValidLevels() []string {
	sortedLevels := logging.SortedLevels()
	labels := make([]string, 0, len(sortedLevels))
	for _, lvl := range sortedLevels {
		labels = append(labels, logging.LabelForLevelOrInvalid(lvl))
	}

	return labels
}
