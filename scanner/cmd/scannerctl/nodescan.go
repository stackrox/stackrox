package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/pkg/errors"
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
		if len(args) != 2 {
			return errors.New("expected two arguments: pkgType <binary|source> version")
		}

		var ir = createIndexReport(args[0], args[1])
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
		return nil
	}
	return &cmd
}

func normalizeVersion(version string) []int32 {
	fields := strings.Split(version, ".")
	switch len(fields) {
	case 0:
		return []int32{0, 0, 0}
	case 1:
		i, err := strconv.Atoi(fields[0])
		if err == nil {
			return []int32{int32(i), 0, 0}
		}
	default:
		i1, err1 := strconv.Atoi(fields[0])
		i2, err2 := strconv.Atoi(fields[1])
		// Only two first fields matter for the matcher
		if err1 == nil && err2 == nil {
			return []int32{int32(i1), int32(i2), 0}
		}
	}
	return []int32{0, 0, 0}
}

func createIndexReport(kind, version string) *v4.IndexReport {
	const goldenName = "Red Hat Container Catalog"
	const goldenURI = `https://catalog.redhat.com/software/containers/explore`

	return &v4.IndexReport{
		HashId:  "sha256:11cf2360bc7d8d4fef440b3fa97ce49cd648318632328f42ecbfb071b823ae14",
		State:   "7", // IndexFinished
		Success: true,
		Err:     "",
		Contents: &v4.Contents{
			Packages: []*v4.Package{
				{
					Id:      "6",
					Name:    "rhcos",
					Version: version, // this will be read!
					NormalizedVersion: &v4.NormalizedVersion{ // required due to Kind
						Kind: "rhctag",
						V:    normalizeVersion(version),
					},
					FixedInVersion: "",
					Kind:           kind,
					Source: &v4.Package{
						Id:  "6",
						Cpe: "cpe:2.3:*", // required to pass validation
					},
					Arch: "",
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
							IntroducedIn:  "sha256:11cf2360bc7d8d4fef440b3fa97ce49cd648318632328f42ecbfb071b823ae14",
							RepositoryIds: []string{"6"},
						},
					},
				},
			},
		},
	}
}
