package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/compliance/virtualmachines/roxagent/common"
	"github.com/stackrox/rox/compliance/virtualmachines/roxagent/index"
	"github.com/stackrox/rox/compliance/virtualmachines/roxagent/vsock"
)

const (
	minDaemonIndexInterval = 10 * time.Minute
	repoToCPEMappingURL    = "https://security.access.redhat.com/data/metrics/repository-to-cpe.json"
)

func RootCmd(ctx context.Context) *cobra.Command {
	cmd := cobra.Command{
		Use:          "agent",
		Short:        "Collects index reports for vulnerability scanning of virtual machines.",
		SilenceUsage: true,
	}
	cmd.SetContext(ctx)
	cfg := &common.Config{}
	cmd.Flags().BoolVar(&cfg.DaemonMode, "daemon", false,
		"Run in daemon mode. Sends index reports continuously.",
	)

	// Shortening this interval results in more frequent scans and therefore more load,
	// which, assuming the throughput continues to be limited by scanning capacity,
	// reduces the number of VMs that Stackrox can handle.
	// A report every 4 h results in Stackrox being able to handle around 4500 VMs,
	// while a report every hour results in a capacity of around 1100 VMs.
	// See the documentation for more details.
	cmd.Flags().DurationVar(&cfg.IndexInterval, "index-interval", 240*time.Minute,
		fmt.Sprintf(
			"Interval at which index reports are sent in daemon mode (minimum: %v). "+
				"Shorter intervals increase scanning load and reduce the overall number of VMs that can be scanned.",
			minDaemonIndexInterval,
		),
	)
	cmd.Flags().StringVar(&cfg.IndexHostPath, "host-path", "/",
		"Path where the indexer starts searching for the RPM and DNF databases.",
	)
	cmd.Flags().DurationVar(&cfg.MaxInitialReportDelay, "max-initial-report-delay", 20*time.Minute,
		"Max delay before starting to send in daemon mode.",
	)
	cmd.Flags().StringVar(&cfg.RepoToCPEMappingURL, "repo-cpe-url", repoToCPEMappingURL,
		"URL for the repository to CPE mapping.",
	)
	cmd.Flags().DurationVar(&cfg.Timeout, "timeout", 30*time.Second,
		"VSock client timeout when sending index reports.",
	)
	cmd.Flags().BoolVar(&cfg.Verbose, "verbose", false,
		"Prints the index reports to stdout.",
	)
	cmd.Flags().Uint32Var(&cfg.VsockPort, "port", 818,
		"VSock port to connect with the virtual machine host.",
	)
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		if err := validateDaemonConfig(cfg); err != nil {
			return err
		}

		client := &vsock.Client{
			Port:     cfg.VsockPort,
			HostPath: cfg.IndexHostPath,
			Timeout:  cfg.Timeout,
			Verbose:  cfg.Verbose,
		}
		if cfg.DaemonMode {
			if err := index.RunDaemon(ctx, cfg, client); err != nil {
				return fmt.Errorf("running indexer daemon: %w", err)
			}
			return nil
		}
		if err := index.RunSingle(ctx, cfg, client); err != nil {
			return fmt.Errorf("running indexer: %w", err)
		}
		return nil
	}
	return &cmd
}

func validateDaemonConfig(cfg *common.Config) error {
	if !cfg.DaemonMode {
		return nil
	}
	if cfg.IndexInterval < minDaemonIndexInterval {
		return fmt.Errorf("index interval must be at least %s in daemon mode (got %s)", minDaemonIndexInterval, cfg.IndexInterval)
	}
	return nil
}
