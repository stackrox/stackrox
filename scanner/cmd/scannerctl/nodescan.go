package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/spf13/cobra"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
)

func nodeScanCmd(ctx context.Context) *cobra.Command {
	cmd := cobra.Command{
		Use:   "nodescan",
		Short: "Triggers a node scan for the node the target node indexer runs on",
	}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Create scanner client.
		scanner, err := factory.Create(ctx)
		if err != nil {
			return fmt.Errorf("create client: %w", err)
		}

		var ir = createIndexReport()
		nodeDigest, err := name.NewDigest("registry.redhat.io/rhcos@" + ir.GetHashId())
		if err != nil {
			return fmt.Errorf("failed to parse const node digest %w", err)
		}
		_ = nodeDigest
		digest := nodeDigest

		vr, err := scanner.GetVulnerabilities(ctx, digest, ir.GetContents())
		if err != nil {
			return fmt.Errorf("scanning: %w", err)
		}
		reportJSON, err := json.MarshalIndent(vr, "", "  ")
		if err != nil {
			return fmt.Errorf("decoding report: %w", err)
		}
		_ = reportJSON
		// fmt.Println(string(reportJSON))
		fmt.Printf("Vulnerability Report contains %d packages with %d vulns and %d package vulns\n",
			len(vr.GetContents().GetPackages()),
			len(vr.GetVulnerabilities()),
			len(vr.GetPackageVulnerabilities()))

		for s, vulnerability := range vr.GetVulnerabilities() {
			fmt.Printf("VULN ID=%s: NAME=%s\n", s, vulnerability.GetName())
		}
		for s, vulnerability := range vr.GetPackageVulnerabilities() {
			fmt.Printf("PKG-VULN ID=%s: VALUES=%s\n", s, vulnerability.GetValues())
		}
		return nil
	}
	return &cmd
}

func createIndexReport() *v4.IndexReport {
	const goldenName = "Red Hat Container Catalog"
	const goldenURI = `https://catalog.redhat.com/software/containers/explore`

	return &v4.IndexReport{
		HashId:  "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		State:   "7", // IndexFinished
		Success: true,
		Err:     "",
		Contents: &v4.Contents{
			Packages: []*v4.Package{
				{
					Id:      "6",
					Name:    "rhcos",
					Version: "411.94.2024", // this will be read!
					NormalizedVersion: &v4.NormalizedVersion{ // required due to Kind
						Kind: "rhctag",
					},
					FixedInVersion: "",
					Kind:           "binary",
					Source: &v4.Package{
						Id:  "6",
						Cpe: "cpe:2.3:*", // required to pass validation
					},
					Arch: "x86_64",
					Cpe:  "cpe:2.3:*", // required to pass validation
				},
			},
			Repositories: []*v4.Repository{
				{
					Id:   "6",
					Name: goldenName,
					Key:  "",
					Uri:  goldenURI,
					Cpe:  "cpe:2.3:*", // required to pass validation
				},
			},
			// Environments must be present for the matcher to discover records
			Environments: map[string]*v4.Environment_List{
				"6": {
					Environments: []*v4.Environment{
						{
							PackageDb:     "",
							IntroducedIn:  "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
							RepositoryIds: []string{"6"},
						},
					},
				},
			},
		},
	}
}
