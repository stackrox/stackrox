// Binary co-acs-scan-importer reads Compliance Operator ScanSettingBinding
// resources from one or more Kubernetes clusters and creates equivalent ACS
// compliance scan configurations through the ACS v2 API.
//
// Run with --help for full usage information and examples.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/stackrox/co-acs-importer/internal/acs"
	"github.com/stackrox/co-acs-importer/internal/cofetch"
	"github.com/stackrox/co-acs-importer/internal/config"
	"github.com/stackrox/co-acs-importer/internal/preflight"
	"github.com/stackrox/co-acs-importer/internal/run"
	"github.com/stackrox/co-acs-importer/internal/status"
)

func main() {
	os.Exit(mainWithCode())
}

func mainWithCode() int {
	cfg, err := config.ParseAndValidate(os.Args[1:])
	if err != nil {
		if err == config.ErrHelpRequested {
			return 0
		}
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return run.ExitFatalError
	}

	s := status.New()
	ctx := context.Background()

	// Preflight check before any resource processing.
	s.Stage("Preflight", "checking ACS connectivity and credentials")
	if err := preflight.Run(ctx, cfg); err != nil {
		s.Failf("%v", err)
		return run.ExitFatalError
	}
	s.OKf("ACS endpoint is reachable at %s", cfg.ACSEndpoint)

	acsClient, err := acs.NewClient(cfg)
	if err != nil {
		s.Failf("failed to create ACS client: %v", err)
		return run.ExitFatalError
	}

	// Resolve --cluster name lookup if needed (IMP-MAP-022).
	if cfg.ClusterNameLookup != "" {
		s.Stagef("Resolve", "looking up cluster %q in ACS", cfg.ClusterNameLookup)
		clusters, err := acsClient.ListClusters(ctx)
		if err != nil {
			s.Failf("failed to list ACS clusters: %v", err)
			return run.ExitFatalError
		}
		var found bool
		for _, c := range clusters {
			if c.Name == cfg.ClusterNameLookup {
				cfg.ACSClusterID = c.ID
				found = true
				break
			}
		}
		if !found {
			s.Failf("cluster %q not found in ACS", cfg.ClusterNameLookup)
			return run.ExitFatalError
		}
		s.OKf("resolved %q → %s", cfg.ClusterNameLookup, cfg.ACSClusterID)
	}

	// Multi-cluster mode or single-cluster with auto-discovery both use
	// BuildClusterSources to resolve cluster IDs and create CO clients.
	isMultiClusterMode := len(cfg.Kubeconfigs) > 0 || len(cfg.Kubecontexts) > 0

	if isMultiClusterMode || cfg.AutoDiscoverClusterID {
		if isMultiClusterMode {
			s.Stagef("Discovery", "resolving %d cluster sources", len(cfg.Kubeconfigs)+len(cfg.Kubecontexts))
		} else {
			s.Stage("Discovery", "auto-discovering ACS cluster ID from current context")
		}
		sources, err := run.BuildClusterSources(ctx, cfg, acsClient)
		if err != nil {
			s.Failf("%v", err)
			return run.ExitFatalError
		}
		for _, src := range sources {
			s.OKf("%s → %s", src.Label, src.ACSClusterID)
		}

		if isMultiClusterMode {
			return run.NewRunner(cfg, acsClient, nil).RunMultiCluster(ctx, sources)
		}
		// Single-cluster with auto-discovered ID: use the resolved source.
		cfg.ACSClusterID = sources[0].ACSClusterID
		return run.NewRunner(cfg, acsClient, sources[0].COClient).Run(ctx)
	}

	// Single-cluster mode with explicit --cluster UUID.
	s.Stagef("Setup", "using cluster %s", cfg.ACSClusterID)
	coClient, err := cofetch.NewClient(cfg)
	if err != nil {
		s.Failf("failed to create CO client: %v", err)
		return run.ExitFatalError
	}
	s.OK("CO client ready")

	return run.NewRunner(cfg, acsClient, coClient).Run(ctx)
}
