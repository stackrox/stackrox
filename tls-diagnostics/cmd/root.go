package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/tls-diagnostics/internal/certs"
	"github.com/stackrox/rox/tls-diagnostics/internal/detect"
	"github.com/stackrox/rox/tls-diagnostics/internal/diagnostics"
	"github.com/stackrox/rox/tls-diagnostics/internal/k8s"
	"github.com/stackrox/rox/tls-diagnostics/internal/liveprobe"
	"github.com/stackrox/rox/tls-diagnostics/internal/output"
	"github.com/stackrox/rox/tls-diagnostics/internal/rotation"
)

var outputFormat string

func Root() *cobra.Command {
	c := &cobra.Command{
		Use:          "tls-diagnostics",
		Short:        "Inspect TLS certificates in a StackRox deployment",
		SilenceUsage: true,
		RunE:         run,
	}
	c.Flags().StringVarP(&outputFormat, "output", "o", "human", "output format: human or json")
	return c
}

func run(_ *cobra.Command, _ []string) error {
	ctx := context.Background()

	clients, err := k8s.NewClients()
	if err != nil {
		return err
	}

	topo, err := detect.Detect(ctx, clients.Dynamic)
	if err != nil {
		return err
	}

	namespaces := topo.Namespaces()
	reports, err := certs.Collect(ctx, clients.Typed, namespaces)
	if err != nil {
		return err
	}

	now := time.Now()

	var rotationReport *rotation.Report
	var clusterState *rotation.ClusterState
	if centralNS := findCentralNamespace(topo); centralNS != "" {
		state, fetchErr := rotation.FetchClusterState(ctx, clients.Typed, centralNS, namespaces)
		if fetchErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: CA rotation analysis failed: %v\n", fetchErr)
		} else {
			clusterState = state
			rotationReport = rotation.AnalyzeState(state, now)
		}
	}

	fmt.Fprintf(os.Stderr, "Probing live TLS endpoints...\n")
	probeResults := liveprobe.ProbeAll(ctx, clients.RestConfig, clients.Typed, topo, reports, os.Stderr)

	diagResult := diagnostics.RunAll(rotationReport, clusterState, probeResults, now)

	switch outputFormat {
	case "json":
		return output.WriteJSON(os.Stdout, topo, rotationReport, reports, probeResults, diagResult)
	case "human":
		output.WriteTable(os.Stdout, topo, rotationReport, reports, probeResults, diagResult)
		return nil
	default:
		return fmt.Errorf("unknown output format %q (supported: human, json)", outputFormat)
	}
}

func findCentralNamespace(topo *detect.Topology) string {
	for _, inst := range topo.Installations {
		if inst.Kind == "Central" {
			return inst.Namespace
		}
	}
	return ""
}
