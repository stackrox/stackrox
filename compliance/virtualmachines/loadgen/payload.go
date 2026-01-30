package main

import (
	"fmt"
	"sort"
	"time"

	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/fixtures/vmindexreport"
	"google.golang.org/protobuf/proto"
)

// payloadProvider provides pre-generated and pre-marshaled payloads for each CID.
type payloadProvider struct {
	payloads map[uint32][]byte
}

// newPayloadProvider pre-generates payloads for all VMs based on their configurations.
// It groups by package count to reuse generators efficiently.
func newPayloadProvider(vmConfigs []vmConfig, specificPackage string) (*payloadProvider, error) {
	if len(vmConfigs) == 0 {
		return &payloadProvider{payloads: make(map[uint32][]byte)}, nil
	}

	log.Infof("pre-generating %d unique reports (specificPackage=%q)...", len(vmConfigs), specificPackage)
	start := time.Now()

	// Group by package count to reuse generators
	generatorsByPkgCount := make(map[int]*vmindexreport.Generator)
	payloads := make(map[uint32][]byte)

	for _, vmCfg := range vmConfigs {
		// Get or create generator for this package count
		gen, ok := generatorsByPkgCount[vmCfg.numPackages]
		if !ok {
			if specificPackage != "" {
				// All packages will be the specified package (for controlled testing)
				gen = vmindexreport.NewGeneratorWithSpecificPackage(specificPackage, vmCfg.numPackages)
			} else {
				// Use package count as seed for deterministic generation
				gen = vmindexreport.NewGeneratorWithSeed(vmCfg.numPackages, int64(vmCfg.numPackages))
			}
			generatorsByPkgCount[vmCfg.numPackages] = gen
		}

		// Generate report for this CID
		report := gen.GenerateV1IndexReport(vmCfg.cid)
		data, err := proto.Marshal(report)
		if err != nil {
			return nil, fmt.Errorf("marshal report for CID %d: %w", vmCfg.cid, err)
		}
		payloads[vmCfg.cid] = data
	}

	log.Infof("pre-generated %d unique reports (using %d generators) in %s",
		len(payloads), len(generatorsByPkgCount), time.Since(start))

	// Extract and print all package names from all reports (sorted, with duplicates)
	printPackageList(payloads)

	return &payloadProvider{payloads: payloads}, nil
}

func (p *payloadProvider) get(cid uint32) ([]byte, error) {
	payload, ok := p.payloads[cid]
	if !ok {
		return nil, fmt.Errorf("CID %d not in pre-generated range", cid)
	}
	return payload, nil
}

// printPackageList extracts and prints package names grouped by CID.
// For each CID, packages are sorted (with duplicates kept).
func printPackageList(payloads map[uint32][]byte) {
	// Get sorted list of CIDs for consistent output order
	cids := make([]uint32, 0, len(payloads))
	for cid := range payloads {
		cids = append(cids, cid)
	}
	sort.Slice(cids, func(i, j int) bool { return cids[i] < cids[j] })

	fmt.Println("=== Package List by CID ===")
	totalPackages := 0

	for _, cid := range cids {
		data := payloads[cid]
		var report v1.IndexReport
		if err := proto.Unmarshal(data, &report); err != nil {
			log.Errorf("unmarshal report for CID %d: %v", cid, err)
			continue
		}

		var packageNames []string
		if report.IndexV4 != nil && report.IndexV4.Contents != nil {
			for _, pkg := range report.IndexV4.Contents.Packages {
				if pkg != nil {
					packageNames = append(packageNames, pkg.Name)
				}
			}
		}

		// Sort packages for this CID
		sort.Strings(packageNames)

		// Print CID and its packages
		fmt.Printf("\nCID %d:\n", cid)
		for _, name := range packageNames {
			fmt.Printf("- %s\n", name)
		}
		totalPackages += len(packageNames)
	}

	fmt.Printf("\n=== Total: %d CIDs, %d packages ===\n", len(cids), totalPackages)
}
