package debug

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
)

var (
	levels = getValidLevels()

	levelList = strings.Join(levels, " | ")
)

type centralDebugCommand struct {
	// Properties that are bound to cobra flags.

	// Properties that are injected or constructed.
	env     environment.Environment
	timeout time.Duration
}

// Command defines the debug command tree
func Command(cliEnvironment environment.Environment) *cobra.Command {
	centralDebugCmd := &centralDebugCommand{env: cliEnvironment}
	c := &cobra.Command{
		Use: "debug",
	}
	c.AddCommand(centralDebugCmd.logLevelCommand())
	c.AddCommand(DumpCommand(cliEnvironment))
	c.AddCommand(DownloadDiagnosticsCommand(cliEnvironment))
	c.AddCommand(AuthzTraceCommand(cliEnvironment))

	flags.AddTimeout(c)
	return c
}

// LogLevelCommand allows getting and setting the Log Level for StackRox services.
func (cmd *centralDebugCommand) logLevelCommand() *cobra.Command {
	var (
		level   string
		modules []string
	)

	c := &cobra.Command{
		Use:   "log",
		Short: `"log" to get current log level; "log --level=<level>" to set log level`,
		Long:  `"log" to get current log level; "log --level=<level>" to set log level`,
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			cmd.timeout = flags.Timeout(c)
			if level == "" {
				return cmd.getLogLevel(modules)
			}
			return cmd.setLogLevel(level, modules)
		}),
	}
	c.Flags().StringVarP(&level, "level", "l", "",
		fmt.Sprintf("the log level to set the modules to (%s) ", levelList))
	c.Flags().StringSliceVarP(&modules, "modules", "m", nil, "the modules to which to apply the command")
	return c
}

func (cmd *centralDebugCommand) getLogLevel(modules []string) error {
	conn, err := cmd.env.GRPCConnection()
	if err != nil {
		return err
	}
	defer func() {
		_ = conn.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), cmd.timeout)
	defer cancel()

	client := v1.NewDebugServiceClient(conn)
	logResponse, err := client.GetLogLevel(ctx, &v1.GetLogLevelRequest{Modules: modules})
	if err != nil {
		return err
	}

	cmd.printGetLogLevelResponse(logResponse)
	return nil
}

func (cmd *centralDebugCommand) printGetLogLevelResponse(r *v1.LogLevelResponse) {
	const rowFormat = "%-40s  %s"
	indent := ""
	if r.GetLevel() != "" {
		fmt.Printf("Current log level is %s\n", r.GetLevel())
		if len(r.GetModuleLevels()) > 0 {
			fmt.Println("Modules with a different log level:")
			indent = "  "
		}
	}
	if len(r.GetModuleLevels()) > 0 {
		fmt.Printf(indent+rowFormat+"\n", "Module", "Level")
		for _, modLvl := range r.GetModuleLevels() {
			fmt.Printf(indent+rowFormat+"\n", modLvl.GetModule(), modLvl.GetLevel())
		}
	}
}

func (cmd *centralDebugCommand) setLogLevel(level string, modules []string) error {
	conn, err := cmd.env.GRPCConnection()
	if err != nil {
		return err
	}
	defer func() {
		_ = conn.Close()
	}()

	client := v1.NewDebugServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), cmd.timeout)
	defer cancel()

	_, err = client.SetLogLevel(ctx, &v1.LogLevelRequest{Level: level, Modules: modules})
	if err != nil {
		return err
	}

	fmt.Println("Successfully set log level")
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
