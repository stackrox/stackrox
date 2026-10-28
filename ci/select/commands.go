package main

import (
	"context"
	"io"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func run(ctx context.Context, args []string, input io.Reader, output, diagnostic io.Writer) (int, error) {
	code := 2
	command := newCommand(&code)
	command.SetArgs(args)
	command.SetIn(input)
	command.SetOut(output)
	command.SetErr(diagnostic)
	if err := command.ExecuteContext(ctx); err != nil {
		return code, errors.Wrap(err, "execute command")
	}
	if code == 3 {
		return code, nil
	}
	return 0, nil
}

func newCommand(code *int) *cobra.Command {
	root := &cobra.Command{
		Use:   "ci-select",
		Short: "Select and gate StackRox CI jobs for a pull request",
		Long: `Select CI jobs from changed paths, pull-request labels, YAML rules, and job defaults.

Use resolve, validate, gha, and prow with local input files to inspect a plan.
The *-ci commands read the CI environment and integrate with GitHub Actions or Prow.
Without enforcement, dispatchers report the planned decision and let jobs run.`,
		Example: `  ci-select resolve --files-file files.txt --labels-file labels.txt
  ci-select gha gate --job go --files-file files.txt --enforce
  ci-select validate
  ci-select help prow-ci`,
		Args:          cobra.NoArgs,
		RunE:          commandHelp,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.CompletionOptions.DisableDefaultCmd = true
	root.AddCommand(localCommand("resolve", "", code), localCommand("validate", "", code))
	gha := &cobra.Command{
		Use: "gha", Short: "Inspect or gate GitHub Actions jobs using local inputs",
		Args: cobra.NoArgs, RunE: commandHelp,
	}
	gha.AddCommand(localCommand("gha", "print", code), localCommand("gha", "gate", code))
	prow := &cobra.Command{
		Use: "prow", Short: "Gate Prow jobs using local inputs and /test comments",
		Args: cobra.NoArgs, RunE: commandHelp,
	}
	prow.AddCommand(localCommand("prow", "gate", code))
	root.AddCommand(gha, prow)
	root.AddCommand(&cobra.Command{
		Use:   "gha-ci [job]",
		Short: "Print a plan or gate a job from the GitHub Actions environment",
		Long: `Read CI_BASE, CI_HEAD, PR_LABELS, CI_REPO, CI_PR, and CI_COMMIT.
With a job argument, write action=run or action=skip to GITHUB_OUTPUT.
With no job argument, print the plan and append it to GITHUB_STEP_SUMMARY.
CI_ENFORCE=true applies the decision; otherwise the job runs.
CI_DECISION_OUT optionally saves the decision file.
CI_COMMENT_OUT optionally saves a PR comment with the plan when no job is given.`,
		Example: "  ci-select gha-ci\n  ci-select gha-ci go",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			*code = 1
			return ghaCI(cmd.Context(), args, cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	})
	root.AddCommand(&cobra.Command{
		Use:   "gha-dispatch-ci",
		Short: "Write the end-to-end job map and image-wait outputs for GitHub Actions",
		Long: `Read GITHUB_EVENT_NAME, CI_REPO, CI_PR, CI_BASE, CI_HEAD, CI_COMMIT,
CI_IS_FORK, CI_IS_DRAFT, CI_UI_ONLY, and PR_LABELS from the workflow event.
Write jobs, wait-for-gha-images, and wait-for-konflux-images to GITHUB_OUTPUT.
The ci-dispatcher-enforce label applies selection; fork rejection remains in effect.`,
		Example: "  ci-select gha-dispatch-ci",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			*code = 1
			return ghaDispatchCI(cmd.Context(), cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	})
	root.AddCommand(&cobra.Command{
		Use:   "prow-ci <job>",
		Short: "Gate a Prow job using PR JSON on stdin and the Prow environment",
		Long: `Read PR details as JSON from stdin, PULL_BASE_SHA for the diff, and
PULL_PULL_SHA for the head commit timestamp. Retrieve /test comments from GitHub.
The ci-dispatcher-enforce label enables selection. A current /test request forces run.
Exit status 0 means run, 3 means skip, and other nonzero statuses mean an error.
The bash wrapper lets the job run on errors. GitHub requests use GITHUB_TOKEN or GH_TOKEN.`,
		Example: "  get_pr_details | ci-select prow-ci gke-qa-e2e-tests",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			*code = 1
			action, err := prowCI(cmd.Context(), args[0], cmd.InOrStdin(), cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			*code = 0
			if action == "skip" {
				*code = 3
			}
			return nil
		},
	})
	return root
}

func commandHelp(cmd *cobra.Command, _ []string) error {
	return errors.Wrap(cmd.Help(), "show help")
}

func localCommand(command, mode string, code *int) *cobra.Command {
	var o options
	cmd := &cobra.Command{Use: command, Args: cobra.NoArgs}
	if mode != "" {
		cmd.Use = mode
	}
	switch command + " " + mode {
	case "resolve ":
		cmd.Short = "Resolve changed paths and labels into a list of jobs to run"
		cmd.Long = `Read changed paths and labels, one per line, and evaluate the rules.
Write the decision to stdout unless --decision-out is set.
Use --log-out to save the explanation. An empty files list means no files changed.`
		cmd.Example = "  ci-select resolve --files-file files.txt --labels-file labels.txt --decision-out decision.txt --log-out resolver.log"
	case "validate ":
		cmd.Short = "Validate the rules, job defaults, and dependency graph"
		cmd.Example = "  ci-select validate\n  ci-select validate --rules ci/select/testdata/rules.yaml --defaults ci/select/testdata/defaults"
	case "gha print":
		cmd.Short = "Print the decision and its explanation using local input files"
		cmd.Example = "  ci-select gha print --files-file files.txt --labels-file labels.txt"
	case "gha gate", "prow gate":
		cmd.Short = "Print run or skip for one job using local input files"
		cmd.Long = `Resolve the local inputs and write run or skip to stdout, with explanations on stderr.
Without --enforce, the job runs and the planned action is logged.
If the diff is unavailable, use --files-unknown to fall back to job defaults.`
		cmd.Example = "  ci-select " + command + " gate --job go --files-file files.txt --labels-file labels.txt --enforce"
	}
	flags := cmd.Flags()
	flags.StringVar(&o.rules, "rules", "ci/test-domains.yaml", "Path to the YAML selection rules")
	flags.StringVar(&o.defaults, "defaults", "ci/decision-defaults", "Path to the per-job run/skip defaults")
	if command != "validate" {
		flags.StringVar(&o.files, "files-file", "", "Changed paths file, one path per line")
		flags.StringVar(&o.labels, "labels-file", "", "Labels file, one label per line; omit for no labels")
		flags.StringVar(&o.repo, "repo", "stackrox/stackrox", "Repository owner/name for the log")
		flags.StringVar(&o.pr, "pr", "0", "Pull request number for the log")
		flags.StringVar(&o.commit, "commit", "unknown", "Head commit SHA for the log")
		flags.StringVar(&o.decisionOut, "decision-out", "", "Save the decision to this file")
		flags.StringVar(&o.logOut, "log-out", "", "Save the resolver explanation to this file")
		if command == "resolve" {
			cmd.MarkFlagsOneRequired("files-file")
		} else {
			flags.BoolVar(&o.unknown, "files-unknown", false, "Diff is unavailable; use job defaults")
			flags.BoolVar(&o.enforce, "enforce", false, "Apply the decision instead of running every job")
			cmd.MarkFlagsOneRequired("files-file", "files-unknown")
			cmd.MarkFlagsMutuallyExclusive("files-file", "files-unknown")
		}
	}
	if mode == "gate" {
		flags.StringVar(&o.job, "job", "", "Name of the CI job to gate (required)")
		cmd.MarkFlagsOneRequired("job")
	}
	if command == "prow" {
		flags.StringVar(&o.comments, "comments-file", "", "JSON comment array with body and created_at fields")
		flags.StringVar(&o.commitTime, "commit-time", "", "Head commit RFC3339 timestamp for /test freshness")
		cmd.Long += "\nA /test JOB or /test all comment at or after --commit-time forces the job to run."
		cmd.Example = `  ci-select prow gate --job gke-qa-e2e-tests --files-file files.txt --enforce \
    --comments-file comments.json --commit-time 2026-09-30T11:00:00Z`
	}
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		*code = 1
		if command == "validate" {
			return validateRules(o, cmd.OutOrStdout())
		}
		return runLocal(command, mode, o, cmd.OutOrStdout(), cmd.ErrOrStderr())
	}
	return cmd
}
