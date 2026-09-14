// Command operatorbundle is a test harness CLI for the pkg/operatorbundle upgrade advisor.
//
// Given one or more vulnerable image digests, it resolves the installed operator bundle
// from the Red Hat Ecosystem Catalog, selects the latest patch update within the same
// major.minor, scans the candidate bundle's images via StackRox Central, and prints a
// per-image CVE diff (fixed / still active / newly introduced).
//
// It reuses roxctl connection/auth flags to reach Central. Example:
//
//	operatorbundle diff --endpoint central.example.com:443 \
//	  --digest sha256:fcb63b... [--digest sha256:...] [--format table|json]
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/operatorbundle"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/tools/operatorbundle/catalog"
	"github.com/stackrox/rox/tools/operatorbundle/central"
)

func main() {
	if err := rootCommand().Execute(); err != nil {
		os.Exit(1)
	}
}

func rootCommand() *cobra.Command {
	c := &cobra.Command{
		Use:          "operatorbundle <command>",
		Short:        "Operator bundle CVE upgrade advisor",
		SilenceUsage: true,
	}

	// Reuse roxctl-standard Central connection/auth flags.
	flags.AddConnectionFlags(c)
	flags.AddPassword(c)
	flags.AddAPITokenFile(c)
	c.MarkFlagsMutuallyExclusive("password", "token-file")

	cliEnvironment := environment.CLIEnvironment()
	c.AddCommand(diffCommand(cliEnvironment))
	return c
}

type diffOptions struct {
	env          environment.Environment
	digests      []string
	catalogURL   string
	catalogToken string
	format       string
	timeout      time.Duration
	force        bool
	quiet        bool
}

func diffCommand(env environment.Environment) *cobra.Command {
	opts := &diffOptions{env: env}
	c := &cobra.Command{
		Use:   "diff",
		Short: "Diff CVEs between the installed operator bundle and its latest patch update",
		RunE: func(_ *cobra.Command, _ []string) error {
			return opts.run()
		},
	}
	c.Flags().StringArrayVar(&opts.digests, "digest", nil, "Vulnerable image digest to analyze (repeatable)")
	c.Flags().StringVar(&opts.catalogURL, "catalog-url", catalog.DefaultURL, "Operator catalog GraphQL endpoint")
	c.Flags().StringVar(&opts.catalogToken, "catalog-token", "", "Bearer token for the catalog GraphQL endpoint (optional)")
	c.Flags().StringVar(&opts.format, "format", "table", "Output format: table or json")
	c.Flags().DurationVar(&opts.timeout, "timeout", 5*time.Minute, "Overall timeout for the analysis")
	c.Flags().BoolVar(&opts.force, "force", false, "Force re-scan of candidate images instead of using cached scans")
	c.Flags().BoolVar(&opts.quiet, "quiet", false, "Suppress progress output on stderr")
	utils.Must(c.MarkFlagRequired("digest"))
	return c
}

// progress writes a progress line to stderr (via the roxctl logger) unless --quiet is set.
func (o *diffOptions) progress(format string, args ...any) {
	if o.quiet {
		return
	}
	o.env.Logger().InfofLn(format, args...)
}

func (o *diffOptions) run() error {
	if o.format != "table" && o.format != "json" {
		return errors.Errorf("invalid --format %q: must be table or json", o.format)
	}

	o.progress("Connecting to Central…")
	conn, err := o.env.GRPCConnection()
	if err != nil {
		return errors.Wrap(err, "could not establish gRPC connection to Central")
	}
	defer utils.IgnoreError(conn.Close)

	centralClient := central.NewClient(conn, o.force)
	catalogClient := catalog.NewClient(o.catalogURL, o.catalogToken)

	var advisorOpts []operatorbundle.AdvisorOption
	if !o.quiet {
		advisorOpts = append(advisorOpts, operatorbundle.WithProgress(o.progress))
	}
	advisor := operatorbundle.NewAdvisor(catalogClient, centralClient, centralClient, advisorOpts...)

	ctx, cancel := context.WithTimeout(context.Background(), o.timeout)
	defer cancel()

	reports, unresolved, err := advisor.Advise(ctx, o.digests)
	if err != nil {
		return errors.Wrap(err, "computing bundle CVE diff")
	}

	out := o.env.InputOutput().Out()
	if o.format == "json" {
		return renderJSON(out, reports)
	}
	renderTable(out, reports, unresolved)
	if len(reports) == 0 && len(unresolved) > 0 {
		fmt.Fprintln(out, "\nNo operator bundles matched the provided digests.")
	}
	return nil
}
