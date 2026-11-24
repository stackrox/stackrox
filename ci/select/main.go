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
	"github.com/stackrox/rox/ci/select/internal/ruleengine/jobcentric"
	"github.com/stackrox/rox/ci/select/internal/rules"
	"github.com/stackrox/rox/ci/select/internal/validation"
)

// errLintReported marks a failure the validate command already printed as a
// lint report; main exits nonzero without printing it a second time.
var errLintReported = errors.New("lint findings were reported")

// errGateShouldSkip signals that the gate command decided the job should be
// skipped; main exits nonzero without printing an error message.
var errGateShouldSkip = errors.New("gate: job should be skipped")

//nolint:forbidigo // The entry point supplies standard streams to the injectable command runner.
func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := run(ctx, os.Args[1:], os.Stdout, os.Stderr, jobcentric.New()); err != nil {
		if !errors.Is(err, errLintReported) && !errors.Is(err, errGateShouldSkip) {
			fmt.Fprintln(os.Stderr, "ci-select:", err)
		}
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
			return errors.New("a subcommand is required (plan or validate)")
		},
	}
	root.AddCommand(newPlanCommand(eng), newValidateCommand(), newGateCommand(eng))
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
			if err := appendStepSummary(plan); err != nil {
				return err
			}
			return appendGHAOutputs(decision)
		},
	}
	command.Flags().StringVar(&rulesPath, "rules", "ci/test-selection-rules.yaml", "YAML rules file")
	command.Flags().StringVar(&filesPath, "files-file", "", "Changed paths, one per line (e.g. from detect-changes)")
	return command
}

// newGateCommand returns the gate subcommand. It resolves <job> as either an
// inventory name or a prow_alias and exits 0 (run) or 1 (skip). It fails open
// on any error so a misconfigured or missing rules file never blocks a Prow job.
func newGateCommand(eng ruleengine.Engine) *cobra.Command {
	var rulesPath, filesPath string
	command := &cobra.Command{
		Use:   "gate <job>",
		Short: "Exit 0 (run) or 1 (skip) for a CI job",
		Long: `gate resolves <job> as either an inventory name or a prow_alias and
exits 0 if the job should run, 1 if it should be skipped.
Fails open: exits 0 on error or if <job> is not in the rules.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			jobArg := args[0]

			data, err := os.ReadFile(rulesPath)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "ci-select gate: %v; failing open\n", err)
				return nil
			}
			changed, err := readChangedPaths(filesPath)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "ci-select gate: %v; failing open\n", err)
				return nil
			}

			// Parse to resolve prow_alias → inventory name. Fail open on error.
			m, err := rules.Parse(data)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "ci-select gate: parse rules: %v; failing open\n", err)
				return nil
			}

			// Resolve: inventory name first, then prow alias. Unknown → fail open.
			inventoryName := jobArg
			if !m.HasJob(jobArg) {
				inventoryName = m.ResolveByProwAlias(jobArg)
				if inventoryName == "" {
					return nil
				}
			}

			// Run the plan. Fail open on engine error.
			decision, err := eng.Plan(data, changed)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "ci-select gate: plan: %v; failing open\n", err)
				return nil
			}

			if decision.Action(inventoryName) == "skip" {
				return errGateShouldSkip
			}
			return nil
		},
	}
	command.Flags().StringVar(&rulesPath, "rules", "ci/test-selection-rules.yaml", "YAML rules file")
	command.Flags().StringVar(&filesPath, "files-file", "", "Changed paths file")
	return command
}

func newValidateCommand() *cobra.Command {
	var rulesPath string
	command := &cobra.Command{
		Use:   "validate",
		Short: "Lint the rules file for structural problems",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if _, err := validation.ValidateFile(rulesPath); err != nil {
				return writeLint(cmd.ErrOrStderr(), err)
			}
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "Validation passed")
			return errors.Wrap(err, "print validation")
		},
	}
	command.Flags().StringVar(&rulesPath, "rules", "ci/test-selection-rules.yaml", "YAML rules file")
	return command
}

// writeLint prints the finding lines and returns errLintReported.
func writeLint(w io.Writer, err error) error {
	for line := range strings.SplitSeq(err.Error(), "\n") {
		fmt.Fprintln(w, line)
	}
	return errLintReported
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

func appendGHAOutputs(decision ruleengine.Decision) error {
	path := os.Getenv("GITHUB_OUTPUT")
	if path == "" {
		return nil
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return errors.Wrap(err, "open github output")
	}
	var writeErr error
	for job, action := range decision.Actions {
		if _, err := fmt.Fprintf(file, "%s=%s\n", job, action); err != nil {
			writeErr = err
			break
		}
	}
	closeErr := file.Close()
	if writeErr != nil {
		return errors.Wrap(writeErr, "write github output")
	}
	return errors.Wrap(closeErr, "close github output")
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
