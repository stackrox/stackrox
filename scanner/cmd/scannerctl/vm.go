package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/spf13/cobra"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
)

const mockDigest = "registry/repository@sha256:deadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33fdeadb33f"

// scanCmd creates the scan command.
func scanVM(ctx context.Context) *cobra.Command {
	cmd := cobra.Command{
		Use:   "vm",
		Short: "Perform vulnerability scans.",
		Args:  cobra.ExactArgs(0),
	}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Create scanner client.
		scanner, err := factory.Create(ctx)
		if err != nil {
			return fmt.Errorf("create client: %w", err)
		}

		digest, err := name.NewDigest(mockDigest)
		if err != nil {
			fmt.Printf("Failed to parse digest: %v\n", err)
		}

		contents := &v4.Contents{
			Packages: []*v4.Package{
				{
					Id:      "1",
					Name:    "kernel",
					Version: "v2",
					Kind:    "binary",
					Source: &v4.Package{
						Name:    "kernel",
						Version: "v2",
						Kind:    "source",
						Source:  nil,
						Cpe:     "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
					},
					PackageDb:      "sqlite:usr/share/rpm",
					RepositoryHint: "hash:sha256:f52ca767328e6919ec11a1da654e92743587bd3c008f0731f8c4de3af19c1830|key:199e2f91fd431d51",
					Arch:           "x86_64",
					Cpe:            "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*",
				},
			},
			Repositories: []*v4.Repository{
				{
					Id:   "1",
					Name: "cpe:/o:redhat:enterprise_linux:9::fastdatapath",
					Key:  "rhel-cpe-repository",
					Cpe:  "cpe:2.3:o:redhat:enterprise_linux:9:*:fastdatapath:*:*:*:*:*",
				},
			},
			Environments: map[string]*v4.Environment_List{"1": {Environments: []*v4.Environment{
				{
					PackageDb:     "sqlite:usr/share/rpm",
					IntroducedIn:  "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					RepositoryIds: []string{"1"},
				},
			},
			}},
		}
		vr, err := scanner.GetVulnerabilities(ctx, digest, contents)

		if err != nil {
			return fmt.Errorf("scanning: %w", err)
		}
		reportJSON, err := json.MarshalIndent(vr, "", "  ")
		if err != nil {
			return fmt.Errorf("decoding report: %w", err)
		}
		fmt.Println(string(reportJSON))
		return nil
	}
	return &cmd
}
