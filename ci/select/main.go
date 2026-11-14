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
	"github.com/stackrox/rox/ci/select/internal/ruleengine"
	"github.com/stackrox/rox/ci/select/internal/ruleengine/alwaysrun"
)

//nolint:forbidigo // The entry point supplies standard streams to the injectable command runner.
func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := run(ctx, os.Args[1:], os.Stdout, os.Stderr, alwaysrun.New()); err != nil {
		fmt.Fprintln(os.Stderr, "ci-select:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, output, diagnostic io.Writer, eng ruleengine.Engine) error {
	root := &cobra.Command{
		Use:               "ci-select",
		Short:             "Select CI tests from a rules file",
		Args:              cobra.NoArgs,
		SilenceUsage:      true,
		SilenceErrors:     true,
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
		RunE: func(_ *cobra.Command, _ []string) error {
			return errors.New("a subcommand is required (plan)")
		},
	}
	root.AddCommand(newPlanCommand(eng))
	// A nil argument list would make Cobra read the process arguments.
	if args == nil {
		args = []string{}
	}
	root.SetArgs(args)
	root.SetOut(output)
	root.SetErr(diagnostic)
	return errors.Wrap(root.ExecuteContext(ctx), "execute command")
}

func newPlanCommand(eng ruleengine.Engine) *cobra.Command {
	var (
		rulesPath string
		filesPath string
	)
	command := &cobra.Command{
		Use:   "plan",
		Short: "Print the run/skip plan for the rules file and changed paths",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			data, err := os.ReadFile(rulesPath)
			if err != nil {
				return errors.Wrap(err, "read rules")
			}
			changed, err := readChangedPaths(filesPath)
			if err != nil {
				return err
			}
			decision, err := eng.Plan(data, changed)
			if err != nil {
				return errors.Wrap(err, "plan selection")
			}
			fmt.Fprint(cmd.ErrOrStderr(), decision.Log)
			plan := "-- plan --\n" + decision.String()
			if _, err := fmt.Fprint(cmd.OutOrStdout(), plan); err != nil {
				return errors.Wrap(err, "print plan")
			}
			// Surface the plan in the GitHub Actions run summary when present.
			return appendStepSummary(plan)
		},
	}
	command.Flags().StringVar(&rulesPath, "rules", "ci/test-selection-rules.yaml", "YAML rules file")
	command.Flags().StringVar(&filesPath, "files-file", "", "Changed paths, one per line (e.g. from detect-changes)")
	return command
}

// readChangedPaths loads the changed-paths list. An empty path means no
// list was provided (for example a push event upstream), which is a valid
// empty change set.
func readChangedPaths(path string) ([]string, error) {
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "read changed paths")
	}
	var changed []string
	for line := range strings.SplitSeq(string(data), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			changed = append(changed, line)
		}
	}
	return changed, nil
}

func appendStepSummary(plan string) error {
	path := os.Getenv("GITHUB_STEP_SUMMARY")
	if path == "" {
		return nil
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return errors.Wrap(err, "open step summary")
	}
	_, writeErr := io.WriteString(file, "## CI selection plan\n\n```\n"+plan+"```\n")
	closeErr := file.Close()
	if writeErr != nil {
		return errors.Wrap(writeErr, "write step summary")
	}
	return errors.Wrap(closeErr, "close step summary")
}
