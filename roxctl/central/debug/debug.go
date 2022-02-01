package debug

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/util"
)

var (
	levels = getValidLevels()

	levelList = strings.Join(levels, " | ")
)

// Command defines the debug command tree
func Command() *cobra.Command {
	c := &cobra.Command{
		Use: "debug",
	}
	c.AddCommand(LogLevelCommand())
	c.AddCommand(DumpCommand())
	c.AddCommand(DownloadDiagnosticsCommand())
	c.AddCommand(AuthzTraceCommand())

	flags.AddTimeout(c)
	return c
}

// LogLevelCommand allows getting and setting the Log Level for StackRox services.
func LogLevelCommand() *cobra.Command {
	var (
		level   string
		modules []string
	)

	c := &cobra.Command{
		Use:   "log",
		Short: `"log" to get current log level; "log --level=<level>" to set log level`,
		Long:  `"log" to get current log level; "log --level=<level>" to set log level`,
		RunE: util.RunENoArgs(func(c *cobra.Command) error {
			timeout := flags.Timeout(c)
			if level == "" {
				return getLogLevel(modules, timeout)
			}
			return setLogLevel(level, modules, timeout)
		}),
	}
	c.Flags().StringVarP(&level, "level", "l", "",
		fmt.Sprintf("the log level to set the modules to (%s) ", levelList))
	c.Flags().StringSliceVarP(&modules, "modules", "m", nil, "the modules to which to apply the command")
	return c
}

func getLogLevel(modules []string, timeout time.Duration) error {
	conn, err := common.GetGRPCConnection()
	if err != nil {
		return err
	}
	defer func() {
		_ = conn.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client := v1.NewDebugServiceClient(conn)
	logResponse, err := client.GetLogLevel(ctx, &v1.GetLogLevelRequest{Modules: modules})
	if err != nil {
		return err
	}

	printGetLogLevelResponse(logResponse)
	return nil
}

func printGetLogLevelResponse(r *v1.LogLevelResponse) {
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

func setLogLevel(level string, modules []string, timeout time.Duration) error {
	conn, err := common.GetGRPCConnection()
	if err != nil {
		return err
	}
	defer func() {
		_ = conn.Close()
	}()

	client := v1.NewDebugServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
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
