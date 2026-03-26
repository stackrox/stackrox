// Binary co-acs-scan-importer reads Compliance Operator ScanSettingBinding
// resources from Kubernetes clusters and creates equivalent ACS compliance
// scan configurations through the ACS v2 API.
//
// Run with --help for full usage information and examples.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/stackrox/co-acs-importer/internal/acs"
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

	// Build cluster sources from kubeconfig contexts.
	if len(cfg.Contexts) > 0 {
		s.Stagef("Discovery", "resolving %d specified contexts", len(cfg.Contexts))
	} else {
		s.Stage("Discovery", "resolving all kubeconfig contexts")
	}
	sources, err := run.BuildClusterSources(ctx, cfg, acsClient)
	if err != nil {
		s.Failf("%v", err)
		return run.ExitFatalError
	}
	for _, src := range sources {
		s.OKf("%s → %s", src.Label, src.ACSClusterID)
	}

	if len(sources) == 1 {
		cfg.ACSClusterID = sources[0].ACSClusterID
		return run.NewRunner(cfg, acsClient, sources[0].COClient).Run(ctx)
	}
	return run.NewRunner(cfg, acsClient, nil).RunMultiCluster(ctx, sources)
}
