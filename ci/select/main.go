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
			return errors.New("a subcommand is required (validate or resolve)")
		},
	}
	root.AddCommand(newValidateCommand(), newResolveCommand())
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
			_, warnings, err := validation.Load(rulesPath)
			if lintErr := writeLint(cmd.ErrOrStderr(), warnings, err); lintErr != nil {
				return lintErr
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), passedMessage(warnings))
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
			m, warnings, err := validation.Load(rulesPath)
			if err != nil {
				return writeLint(cmd.ErrOrStderr(), warnings, err)
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
			_, err = fmt.Fprint(cmd.OutOrStdout(), "\n-- decisions --\n"+d.String())
			return errors.Wrap(err, "print decision")
		},
	}
	command.Flags().StringVar(&rulesPath, "rules", "ci/test-selection-rules.yaml", "YAML rules")
	command.Flags().StringVar(&filesPath, "files-file", "", "Changed paths, one per line")
	return command
}

// passedMessage is the validate success line. Load returns rendered
// lines, so warnings are counted from "Warning:" lines. They do not
// fail the command.
func passedMessage(warnings []string) string {
	n := 0
	for _, line := range warnings {
		if strings.Contains(line, " Warning:") {
			n++
		}
	}
	if n == 0 {
		return "Validation passed"
	}
	word := "warnings"
	if n == 1 {
		word = "warning"
	}
	return fmt.Sprintf("Validation passed, with %d %s", n, word)
}

// writeLint prints Load's report: the numbered errors, then the warnings
// and the footer. The error text is already the finding lines, so a
// failure returns errLintReported instead of wrapping those lines again.
func writeLint(w io.Writer, warnings []string, err error) error {
	if err != nil {
		for line := range strings.SplitSeq(err.Error(), "\n") {
			fmt.Fprintln(w, line)
		}
	}
	for _, line := range warnings {
		fmt.Fprintln(w, line)
	}
	if err != nil {
		return errLintReported
	}
	return nil
}
