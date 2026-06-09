package reconcile

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
)

const defaultConfigScope = "roxctl-default"

// Command provides the reconcile command for policy-config.
func Command(cliEnvironment environment.Environment) *cobra.Command {
	r := &reconcileCommand{env: cliEnvironment}

	cmd := &cobra.Command{
		Use:   "reconcile",
		Short: "Reconcile policy files against Central",
		Long: `Reads SecurityPolicySpec YAML files from a directory and reconciles them
against a Central instance. Policies managed by this command are identified by
source=DECLARATIVE and the given config-scope. Policies in Central that match
the scope but have no corresponding file are deleted.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := r.validate(); err != nil {
				return err
			}
			return r.run()
		},
	}

	cmd.Flags().StringVarP(&r.dir, "dir", "d", "", "Directory containing policy YAML files (required)")
	cmd.Flags().StringVar(&r.configScope, "config-scope", defaultConfigScope, "Scope label identifying this reconciler instance")
	cmd.Flags().BoolVar(&r.dryRun, "dry-run", false, "Show what would be done without making changes")
	cmd.Flags().DurationVar(&r.timeout, "timeout", 5*time.Minute, "Timeout for the reconciliation")
	cmd.Flags().DurationVar(&r.retryTimeout, "retry-timeout", 20*time.Second, "Timeout for retrying API calls")

	if err := cmd.MarkFlagRequired("dir"); err != nil {
		panic(err)
	}

	flags.HideInheritedFlags(cmd)

	return cmd
}

type reconcileCommand struct {
	env          environment.Environment
	dir          string
	configScope  string
	dryRun       bool
	timeout      time.Duration
	retryTimeout time.Duration
}

func (r *reconcileCommand) validate() error {
	if r.dir == "" {
		return errors.New("--dir is required")
	}
	if strings.TrimSpace(r.configScope) == "" {
		return errors.New("--config-scope must not be empty")
	}
	return nil
}

func (r *reconcileCommand) run() error {
	specs, err := loadPoliciesFromDir(r.dir)
	if err != nil {
		return errors.Wrap(err, "loading policy files")
	}

	r.env.Logger().InfofLn("Loaded %d policies from %s", len(specs), r.dir)

	conn, err := r.env.GRPCConnection(common.WithRetryTimeout(r.retryTimeout))
	if err != nil {
		return errors.Wrap(err, "connecting to Central")
	}
	defer func() { _ = conn.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	rec := &reconciler{
		env:         r.env,
		policySvc:   v1.NewPolicyServiceClient(conn),
		notifierSvc: v1.NewNotifierServiceClient(conn),
		clusterSvc:  v1.NewClustersServiceClient(conn),
		configScope: r.configScope,
		dryRun:      r.dryRun,
	}

	result, err := rec.reconcile(ctx, specs)
	if err != nil {
		return err
	}

	r.printResult(result)

	if len(result.errored) > 0 {
		return errors.Errorf("reconciliation completed with %d errors", len(result.errored))
	}
	return nil
}

func (r *reconcileCommand) printResult(result *reconcileResult) {
	logger := r.env.Logger()

	if r.dryRun {
		logger.InfofLn("Dry run results:")
		logger.InfofLn("  Would create: %d policies", len(result.dryCreate))
		for _, name := range result.dryCreate {
			logger.InfofLn("    + %s", name)
		}
		logger.InfofLn("  Would update: %d policies", len(result.dryUpdate))
		for _, name := range result.dryUpdate {
			logger.InfofLn("    ~ %s", name)
		}
		logger.InfofLn("  Would delete: %d policies", len(result.dryDelete))
		for _, name := range result.dryDelete {
			logger.InfofLn("    - %s", name)
		}
		return
	}

	logger.InfofLn("Reconciliation complete:")
	logger.InfofLn("  Created: %d policies", len(result.created))
	logger.InfofLn("  Applied: %d policies", len(result.applied))
	logger.InfofLn("  Deleted: %d policies", len(result.deleted))
	if len(result.errored) > 0 {
		logger.ErrfLn("  Errors: %d policies", len(result.errored))
	}
}
