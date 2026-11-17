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
	"github.com/stackrox/rox/ci/select/internal/resolver"
	"github.com/stackrox/rox/ci/select/internal/validation"
)

// errLintReported marks a failure the validate command already printed
// as a lint report; main exits nonzero without printing it a second time.
var errLintReported = errors.New("lint findings were reported")

//nolint:forbidigo // The entry point supplies standard streams to the injectable command runner.
func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := run(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		if !errors.Is(err, errLintReported) {
			fmt.Fprintln(os.Stderr, "ci-select:", err)
		}
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
			if _, err := validation.Load(rulesPath); err != nil {
				return writeLint(cmd.ErrOrStderr(), err)
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
			m, err := validation.Load(rulesPath)
			if err != nil {
				return writeLint(cmd.ErrOrStderr(), err)
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
			d := resolver.Resolve(files, m)
			fmt.Fprint(cmd.ErrOrStderr(), d.Log)
			// The summary is stdout while the trace is stderr; a header and
			// leading blank line keep the two sections apart when a terminal
			// merges the streams.
			_, err = fmt.Fprint(cmd.OutOrStdout(), "\n-- decisions --\n"+d.String())
			return errors.Wrap(err, "print decision")
		},
	}
	command.Flags().StringVar(&rulesPath, "rules", "ci/test-selection-rules.yaml", "YAML rules")
	command.Flags().StringVar(&filesPath, "files-file", "", "Changed paths, one per line")
	return command
}

func newGHACommand() *cobra.Command {
	return &cobra.Command{
		Use:   "gha",
		Short: "Report CI selection for GitHub Actions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return gha(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
}

// writeLint prints the finding lines here and returns the sentinel so main
// does not print them again (see errLintReported).
func writeLint(w io.Writer, err error) error {
	for line := range strings.SplitSeq(err.Error(), "\n") {
		fmt.Fprintln(w, line)
	}
	return errLintReported
}

// lintText renders the report as a string for the gha path, which records it
// in the fallback log rather than printing it.
func lintText(err error) string {
	return "validate rules: " + err.Error()
}
