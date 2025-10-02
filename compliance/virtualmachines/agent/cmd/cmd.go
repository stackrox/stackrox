package cmd

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/compliance/virtualmachines/agent/config"
	"github.com/stackrox/rox/compliance/virtualmachines/agent/index"
	"github.com/stackrox/rox/compliance/virtualmachines/agent/vsock"
	"github.com/stackrox/rox/pkg/logging"
)

const repoToCPEMappingURL = "https://security.access.redhat.com/data/metrics/repository-to-cpe.json"

var log = logging.LoggerForModule()

func RootCmd(ctx context.Context) *cobra.Command {
	cmd := cobra.Command{
		Use:          "agent",
		Short:        "Collects index reports for vulnerability scanning of virtual machines.",
		SilenceUsage: true,
	}
	cmd.SetContext(ctx)
	cfg := &config.AgentConfig{}
	cmd.Flags().BoolVar(&cfg.DaemonMode, "daemon", false,
		"Run in daemon mode. Sends index reports continuously.",
	)
	cmd.Flags().DurationVar(&cfg.IndexInterval, "index-interval", 5*time.Minute,
		"Interval duration in which index reports are sent in daemon mode.",
	)
	cmd.Flags().StringVar(&cfg.IndexHostPath, "host-path", "/",
		"Path where the indexer starts searching for the RPM and DNF databases.",
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
	cmd.Flags().Uint32Var(&cfg.VsockPort, "port", 1024,
		"VSock port to connect with the virtual machine host.",
	)
	cmd.Run = func(cmd *cobra.Command, _ []string) {
		client := &vsock.Client{Port: cfg.VsockPort, Timeout: cfg.Timeout}
		if cfg.DaemonMode {
			if err := index.RunDaemon(ctx, cfg, client); err != nil {
				log.Errorf("Running indexer daemon: %v", err)
			}
			return
		}
		if err := index.RunSingle(ctx, cfg, client); err != nil {
			log.Errorf("Running indexer: %v", err)
		}
	}
	return &cmd
}
