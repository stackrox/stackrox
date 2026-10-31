package main

import (
	"fmt"
	"os"
	"time"

	"github.com/stackrox/stackrox/openshift-tests-ext/pkg/adapter"
	"github.com/stackrox/stackrox/openshift-tests-ext/test/acs"
	"github.com/spf13/cobra"

	"github.com/openshift-eng/openshift-tests-extension/pkg/cmd"
	e "github.com/openshift-eng/openshift-tests-extension/pkg/extension"
)

func main() {
	defaultTimeout := 5 * time.Minute

	registry := e.NewRegistry()
	ext := e.NewExtension("acs", "external", "acs-tests")

	// Add acs/all suite
	ext.AddSuite(e.Suite{
		Name:             "acs/all",
		Description:      "All ACS tests",
		Qualifiers:       []string{`tags["component"] == "acs"`},
		ClusterStability: e.ClusterStabilityStable,
		Parallelism:      1,
		TestTimeout:      &defaultTimeout,
	})

	// Add acs/smoke suite
	ext.AddSuite(e.Suite{
		Name:             "acs/smoke",
		Description:      "ACS smoke tests",
		Qualifiers:       []string{`labels.exists(l, l=="SMOKE")`},
		ClusterStability: e.ClusterStabilityStable,
		Parallelism:      1,
		TestTimeout:      &defaultTimeout,
	})

	// Build test specs from ACS scenarios (following Ori's adapter pattern)
	specs, err := adapter.BuildExtensionTestSpecsFromACSScenarios(acs.ACSScenarios, acs.ScriptFS)
	if err != nil {
		panic(fmt.Sprintf("couldn't build extension test specs from ACS scenarios: %+v", err.Error()))
	}

	ext.AddSpecs(specs)
	registry.Register(ext)

	root := &cobra.Command{
		Use:   "acs-tests",
		Short: "ACS Tests Extension for OpenShift",
		Long:  "OpenShift Tests Extension (OTE) binary for Advanced Cluster Security testing",
	}

	root.AddCommand(cmd.DefaultExtensionCommands(registry)...)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
