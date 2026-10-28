package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

//nolint:forbidigo // The entry point supplies standard streams to the injectable command runner.
func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := run(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "ci-select:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, output, diagnostic io.Writer) error {
	root := &cobra.Command{
		Use:               "ci-select",
		Short:             "Select CI tests from changed paths",
		Args:              cobra.NoArgs,
		SilenceUsage:      true,
		SilenceErrors:     true,
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
		RunE: func(_ *cobra.Command, _ []string) error {
			return errors.New("a subcommand is required (validate, resolve, or gha)")
		},
	}
	root.AddCommand(newValidateCommand(), newResolveCommand(), newGHACommand())
	// A nil argument list would make Cobra read the process arguments.
	if args == nil {
		args = []string{}
	}
	root.SetArgs(args)
	root.SetOut(output)
	root.SetErr(diagnostic)
	return errors.Wrap(root.ExecuteContext(ctx), "execute command")
}

func newValidateCommand() *cobra.Command {
	var rulesPath string
	command := &cobra.Command{
		Use:   "validate",
		Short: "Validate the CI selection rules",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if _, err := loadMapping(rulesPath); err != nil {
				return err
			}
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "Validation passed")
			return errors.Wrap(err, "print validation")
		},
	}
	command.Flags().StringVar(&rulesPath, "rules", "ci/test-selection-rules.yaml", "YAML rules")
	return command
}

func newResolveCommand() *cobra.Command {
	var (
		rulesPath string
		filesPath string
	)
	command := &cobra.Command{
		Use:   "resolve",
		Short: "Resolve test decisions from changed paths",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			m, err := loadMapping(rulesPath)
			if err != nil {
				return err
			}
			if filesPath == "" {
				return errors.New("resolve needs --files-file")
			}
			data, err := os.ReadFile(filesPath)
			if err != nil {
				return errors.Wrap(err, "read changed paths")
			}
			var files []string
			for line := range strings.SplitSeq(string(data), "\n") {
				if line != "" {
					files = append(files, line)
				}
			}
			d := resolve(files, m)
			fmt.Fprint(cmd.ErrOrStderr(), d.log)
			_, err = fmt.Fprint(cmd.OutOrStdout(), d.String())
			return errors.Wrap(err, "print decision")
		},
	}
	command.Flags().StringVar(&rulesPath, "rules", "ci/test-selection-rules.yaml", "YAML rules")
	command.Flags().StringVar(&filesPath, "files-file", "", "Changed paths, one per line")
	return command
}

func newGHACommand() *cobra.Command {
	return &cobra.Command{
		Use:   "gha [JOB]",
		Short: "Report or apply CI selection for GitHub Actions",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var job string
			if len(args) > 0 {
				job = args[0]
			}
			return gha(cmd.Context(), job, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
}
