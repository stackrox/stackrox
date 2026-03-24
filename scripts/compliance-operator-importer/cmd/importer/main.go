// Binary co-acs-scan-importer reads Compliance Operator ScanSettingBinding
// resources from a Kubernetes cluster and creates equivalent ACS compliance
// scan configurations through the ACS v2 API.
//
// Usage:
//
//	co-acs-scan-importer \
//	  --acs-endpoint https://central.example.com \
//	  --co-namespace openshift-compliance \
//	  --acs-cluster-id <cluster-id> \
//	  [--dry-run] [--report-json /tmp/report.json]
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
)

func main() {
	os.Exit(mainWithCode())
}

func mainWithCode() int {
	cfg, err := config.ParseAndValidate(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		return run.ExitFatalError
	}

	ctx := context.Background()

	// IMP-CLI-015, IMP-CLI-016: preflight check before any resource processing.
	if err := preflight.Run(ctx, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: preflight failed: %v\n", err)
		return run.ExitFatalError
	}

	acsClient, err := acs.NewClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: failed to create ACS client: %v\n", err)
		return run.ExitFatalError
	}

	coClient, err := cofetch.NewClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: failed to create CO client: %v\n", err)
		return run.ExitFatalError
	}

	return run.NewRunner(cfg, acsClient, coClient).Run(ctx)
}
