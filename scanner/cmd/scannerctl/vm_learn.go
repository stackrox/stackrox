package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/fixtures/vmindexreport"
)

// PackageVulnData stores vulnerability information for a single package.
type PackageVulnData struct {
	Index   int    `json:"index"`   // Index in the fixture array
	Name    string `json:"name"`    // Package name
	Version string `json:"version"` // Package version
	Repo    string `json:"repo"`    // Repository
	Vulns   int    `json:"vulns"`   // Number of vulnerabilities found
}

// LearnedVulnData is the top-level structure for the learned data file.
type LearnedVulnData struct {
	LearnedAt   time.Time         `json:"learned_at"`
	MatcherAddr string            `json:"matcher_address"`
	TotalPkgs   int               `json:"total_packages"`
	Packages    []PackageVulnData `json:"packages"`
}

// LoadLearnedData loads vulnerability data from a JSON file.
func LoadLearnedData(path string) (*LearnedVulnData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}
	var learned LearnedVulnData
	if err := json.Unmarshal(data, &learned); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}
	return &learned, nil
}

// SaveLearnedData saves vulnerability data to a JSON file.
func SaveLearnedData(path string, data *LearnedVulnData) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}
	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}
	return nil
}

// vmLearnCmd creates the VM learn command to discover vulnerability counts per package.
func vmLearnCmd(ctx context.Context) *cobra.Command {
	cmd := cobra.Command{
		Use:   "vm-learn",
		Short: "Learn vulnerability counts for each package in the fixture",
		Long: `Query Scanner V4 Matcher for each package individually to learn how many
vulnerabilities each package has. This data is saved to a JSON file that can be
used by vm-scale to craft specific test scenarios.

Example:
  # Learn vulnerability data and save to file
  scannerctl vm-learn --output package_vulns.json

  # Then use with vm-scale
  scannerctl vm-scale --vuln-data package_vulns.json --target-vulns 100 --packages 50
`,
	}

	flags := cmd.PersistentFlags()
	outputFile := flags.String("output", "package_vulns.json", "Output file for learned vulnerability data")
	verbose := flags.Bool("verbose", false, "Print progress for each package")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		matcherAddr, _ := cmd.Flags().GetString("matcher-address")

		log.Printf("Learning vulnerability data for %d packages...", len(vmindexreport.Rhel9Packages))
		log.Printf("This may take a few minutes.")

		// Create scanner client
		scanner, err := factory.Create(ctx)
		if err != nil {
			return fmt.Errorf("creating scanner client: %w", err)
		}

		// Parse digest once
		digest, err := name.NewDigest(vmindexreport.MockDigestWithRegistry)
		if err != nil {
			return fmt.Errorf("parsing digest: %w", err)
		}

		learned := &LearnedVulnData{
			LearnedAt:   time.Now(),
			MatcherAddr: matcherAddr,
			TotalPkgs:   len(vmindexreport.Rhel9Packages),
			Packages:    make([]PackageVulnData, 0, len(vmindexreport.Rhel9Packages)),
		}

		// Query each package individually
		for i, pkg := range vmindexreport.Rhel9Packages {
			// Generate index report with just this one package
			gen := vmindexreport.NewGeneratorWithPackageIndices([]int{i}, 2)
			indexReport := gen.GenerateV4IndexReport()

			// Query scanner
			vulnReport, err := scanner.GetVulnerabilities(ctx, digest, indexReport.GetContents())
			if err != nil {
				log.Printf("[%d/%d] %s: ERROR - %v", i+1, len(vmindexreport.Rhel9Packages), pkg.Name, err)
				// Store with -1 vulns to indicate error
				learned.Packages = append(learned.Packages, PackageVulnData{
					Index:   i,
					Name:    pkg.Name,
					Version: pkg.Version,
					Repo:    pkg.Repo,
					Vulns:   -1,
				})
				continue
			}

			vulnCount := 0
			if vulnReport != nil {
				vulnCount = len(vulnReport.GetVulnerabilities())
			}

			learned.Packages = append(learned.Packages, PackageVulnData{
				Index:   i,
				Name:    pkg.Name,
				Version: pkg.Version,
				Repo:    pkg.Repo,
				Vulns:   vulnCount,
			})

			if *verbose {
				log.Printf("[%d/%d] %s (%s): %d vulns", i+1, len(vmindexreport.Rhel9Packages), pkg.Name, pkg.Version, vulnCount)
			} else if (i+1)%50 == 0 {
				log.Printf("Progress: %d/%d packages scanned...", i+1, len(vmindexreport.Rhel9Packages))
			}
		}

		// Save results
		if err := SaveLearnedData(*outputFile, learned); err != nil {
			return fmt.Errorf("saving learned data: %w", err)
		}

		// Print summary
		var withVulns, withoutVulns, totalVulns int
		for _, pkg := range learned.Packages {
			if pkg.Vulns > 0 {
				withVulns++
				totalVulns += pkg.Vulns
			} else if pkg.Vulns == 0 {
				withoutVulns++
			}
		}

		log.Printf("\n=== Learning Complete ===")
		log.Printf("Total packages: %d", len(learned.Packages))
		log.Printf("Packages with vulnerabilities: %d", withVulns)
		log.Printf("Packages without vulnerabilities: %d", withoutVulns)
		log.Printf("Total vulnerabilities: %d", totalVulns)
		log.Printf("Data saved to: %s", *outputFile)

		return nil
	}

	return &cmd
}
