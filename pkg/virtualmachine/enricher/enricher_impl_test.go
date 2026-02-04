package enricher

import (
	"fmt"
	"maps"
	"slices"
	"testing"

	"github.com/pkg/errors"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/scanners/scannerv4"
	scannerMocks "github.com/stackrox/rox/pkg/scanners/types/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"golang.org/x/sync/semaphore"
)

// TestEnrichVirtualMachineWithVulnerabilities_Success tests the successful enrichment flow.
// The test verifies that the enricher correctly converts vulnerability reports to VM scans
// by creating index reports with different package configurations and checking that:
//  1. The correct number of components are created (one per package in the index report)
//  2. Vulnerabilities are properly distributed across components using the helper function
//     createVulnerabilityReportFromIndexReport, which cycles through packages when assigning vulns
//  3. VM notes are properly cleared and scan notes are converted from vulnerability report notes
//  4. Scan metadata is set correctly
func TestEnrichVirtualMachineWithVulnerabilities_Success(t *testing.T) {
	tests := []struct {
		name                         string
		vm                           *storage.VirtualMachine
		indexReport                  *v4.IndexReport
		expectedVulnerabilitiesCount int
		expectedScanNotesCount       int
	}{
		{
			name: "successful enrichment with vulnerabilities",
			vm: &storage.VirtualMachine{
				Id:   "vm-id-1",
				Name: "test-vm",
			},
			indexReport: &v4.IndexReport{
				Contents: &v4.Contents{
					Packages: map[string]*v4.Package{
						"pkg-1": {
							Id:      "pkg-1",
							Name:    "libssl",
							Version: "1.1.1",
						},
						"pkg-2": {
							Id:      "pkg-2",
							Name:    "curl",
							Version: "7.68.0",
						},
					},
				},
			},
			expectedVulnerabilitiesCount: 3,
			expectedScanNotesCount:       0,
		},
		{
			name: "successful enrichment with no vulnerabilities",
			vm: &storage.VirtualMachine{
				Id:   "vm-id-2",
				Name: "clean-vm",
			},
			indexReport: &v4.IndexReport{
				Contents: &v4.Contents{
					Packages: map[string]*v4.Package{
						"pkg-safe": {
							Id:      "pkg-safe",
							Name:    "safe-package",
							Version: "1.0.0",
						},
					},
				},
			},
			expectedVulnerabilitiesCount: 0,
			expectedScanNotesCount:       0,
		},
		{
			name: "enrichment with empty package list",
			vm: &storage.VirtualMachine{
				Id:   "vm-id-3",
				Name: "empty-vm",
			},
			indexReport: &v4.IndexReport{
				Contents: &v4.Contents{
					Packages: map[string]*v4.Package{},
				},
			},
			expectedVulnerabilitiesCount: 0,
			expectedScanNotesCount:       0,
		},
		{
			name: "enrichment with scan notes",
			vm: &storage.VirtualMachine{
				Id:   "vm-id-4",
				Name: "noted-vm",
			},
			indexReport: &v4.IndexReport{
				Contents: &v4.Contents{
					Packages: map[string]*v4.Package{
						"pkg-1": {
							Id:      "pkg-1",
							Name:    "test-package",
							Version: "1.0.0",
						},
					},
				},
			},
			expectedVulnerabilitiesCount: 0,
			expectedScanNotesCount:       2,
		},
		{
			name: "enrichment with multiple scan notes",
			vm: &storage.VirtualMachine{
				Id:   "vm-id-5",
				Name: "multi-noted-vm",
			},
			indexReport: &v4.IndexReport{
				Contents: &v4.Contents{
					Packages: map[string]*v4.Package{
						"pkg-1": {
							Id:      "pkg-1",
							Name:    "another-package",
							Version: "2.0.0",
						},
					},
				},
			},
			expectedVulnerabilitiesCount: 1,
			expectedScanNotesCount:       4, // Test with 4 notes to show cycling
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			virtualMachineScan := createVirtualMachineScanFromIndexReport(tt.indexReport, tt.expectedVulnerabilitiesCount)
			virtualMachineScan.Notes = make([]storage.VirtualMachineScan_Note, tt.expectedScanNotesCount)
			for i := range tt.expectedScanNotesCount {
				// Cycle through available note types
				noteTypes := []storage.VirtualMachineScan_Note{
					storage.VirtualMachineScan_OS_UNKNOWN,
					storage.VirtualMachineScan_OS_UNSUPPORTED,
					storage.VirtualMachineScan_UNSET,
				}
				virtualMachineScan.Notes[i] = noteTypes[i%len(noteTypes)]
			}

			mockScanner := scannerMocks.NewMockVirtualMachineScanner(ctrl)
			mockScanner.EXPECT().MaxConcurrentNodeScanSemaphore().Return(semaphore.NewWeighted(1))
			mockScanner.EXPECT().GetVirtualMachineScan(gomock.Any(), gomock.Any()).Return(virtualMachineScan, nil)

			enricher := New(mockScanner)

			err := enricher.EnrichVirtualMachineWithVulnerabilities(tt.vm, tt.indexReport)
			require.NoError(t, err)

			// Verify the scan was created
			require.NotNil(t, tt.vm.GetScan())
			expectedComponentsCount := len(tt.indexReport.GetContents().GetPackages())
			assert.Equal(t, expectedComponentsCount, len(tt.vm.GetScan().GetComponents()))

			// Count total vulnerabilities across all components
			totalVulns := 0
			for _, component := range tt.vm.GetScan().GetComponents() {
				totalVulns += len(component.GetVulnerabilities())
			}
			assert.Equal(t, tt.expectedVulnerabilitiesCount, totalVulns)

			// Verify VM notes are cleared
			assert.Empty(t, tt.vm.GetNotes())

			// Verify scan notes count
			assert.Len(t, tt.vm.GetScan().GetNotes(), tt.expectedScanNotesCount)

			// Verify scan metadata
			assert.NotNil(t, tt.vm.GetScan().GetScanTime())
			assert.Equal(t, "", tt.vm.GetScan().GetOperatingSystem())
		})
	}
}

func TestEnrichVirtualMachineWithVulnerabilities_Errors(t *testing.T) {
	// Make sure the enricher propagates the error returned
	// by the virtual machine scanner.
	t.Run("nil virtual machine scanner", func(it *testing.T) {
		enricher := New(nil)

		virtualMachine := &storage.VirtualMachine{
			Id:   "vm-id",
			Name: "test-vm",
		}
		indexReport := &v4.IndexReport{
			Contents: &v4.Contents{},
		}

		err := enricher.EnrichVirtualMachineWithVulnerabilities(virtualMachine, indexReport)
		require.Error(it, err)
		assert.Contains(it, err.Error(), "Scanner V4 client not available for VM enrichment")
		expectedNotes := []storage.VirtualMachine_Note{storage.VirtualMachine_MISSING_SCAN_DATA}
		assert.Equal(it, expectedNotes, virtualMachine.GetNotes())
	})
	t.Run("virtual machine enricher propagates virtual machine scanner errors", func(it *testing.T) {
		ctrl := gomock.NewController(it)
		defer ctrl.Finish()

		testError := errors.New("test error")

		mockScanner := scannerMocks.NewMockVirtualMachineScanner(ctrl)
		mockScanner.EXPECT().MaxConcurrentNodeScanSemaphore().Return(semaphore.NewWeighted(1))
		mockScanner.EXPECT().GetVirtualMachineScan(gomock.Any(), gomock.Any()).Return(nil, testError)

		enricher := New(mockScanner)

		virtualMachine := &storage.VirtualMachine{
			Id:   "vm-id",
			Name: "test-vm",
		}
		indexReport := &v4.IndexReport{
			Contents: &v4.Contents{},
		}

		err := enricher.EnrichVirtualMachineWithVulnerabilities(virtualMachine, indexReport)
		assert.Error(it, err)
		assert.Nil(it, virtualMachine.GetScan())
		assert.ErrorIs(it, err, testError)
	})
}

func TestEnrichVirtualMachineWithVulnerabilities_ClearPreExistingNotes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	vm := &storage.VirtualMachine{
		Id:    "vm-id",
		Name:  "test-vm",
		Notes: []storage.VirtualMachine_Note{storage.VirtualMachine_MISSING_SCAN_DATA}, // Pre-existing notes
	}

	indexReport := &v4.IndexReport{
		Contents: &v4.Contents{
			Packages: map[string]*v4.Package{
				"pkg-1": {
					Id:      "pkg-1",
					Name:    "test-package",
					Version: "1.0.0",
				},
			},
		},
	}

	virtualMachineScan := createVirtualMachineScanFromIndexReport(indexReport, 0)
	mockScanner := scannerMocks.NewMockVirtualMachineScanner(ctrl)
	mockScanner.EXPECT().MaxConcurrentNodeScanSemaphore().Return(semaphore.NewWeighted(1))
	mockScanner.EXPECT().GetVirtualMachineScan(gomock.Any(), gomock.Any()).Return(virtualMachineScan, nil)

	enricher := New(mockScanner)

	err := enricher.EnrichVirtualMachineWithVulnerabilities(vm, indexReport)
	require.NoError(t, err)

	// Verify that pre-existing notes are cleared
	assert.Empty(t, vm.GetNotes())
	assert.NotNil(t, vm.GetScan())
}

func TestEnrichVirtualMachineWithVulnerabilities_ComponentConversion(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	vm := &storage.VirtualMachine{
		Id:   "vm-id",
		Name: "test-vm",
	}

	indexReport := &v4.IndexReport{
		Contents: &v4.Contents{
			Packages: map[string]*v4.Package{
				"pkg-1": {
					Id:      "pkg-1",
					Name:    "openssl",
					Version: "1.1.1f",
				},
				"pkg-2": {
					Id:      "pkg-2",
					Name:    "libssl",
					Version: "1.1.1f",
				},
			},
		},
	}

	vulnReport := &v4.VulnerabilityReport{
		Contents: indexReport.GetContents(),
		PackageVulnerabilities: map[string]*v4.StringList{
			"pkg-1": {Values: []string{"vuln-1", "vuln-2"}},
			"pkg-2": {Values: []string{"vuln-1"}}, // Shared vulnerability
		},
		Vulnerabilities: map[string]*v4.VulnerabilityReport_Vulnerability{
			"vuln-1": {
				Name:               "CVE-2021-1234",
				Description:        "Critical SSL vulnerability",
				NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_CRITICAL,
				Link:               "https://cve.mitre.org/cgi-bin/cvename.cgi?name=CVE-2021-1234",
				Issued:             protocompat.TimestampNow(),
				FixedInVersion:     "1.1.1g",
			},
			"vuln-2": {
				Name:               "CVE-2021-5678",
				Description:        "Moderate SSL vulnerability",
				NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_MODERATE,
				Issued:             protocompat.TimestampNow(),
			},
		},
		Notes: []v4.VulnerabilityReport_Note{},
	}
	virtualMachineScan := scannerv4.ToVirtualMachineScan(vulnReport)

	mockScanner := scannerMocks.NewMockVirtualMachineScanner(ctrl)
	mockScanner.EXPECT().MaxConcurrentNodeScanSemaphore().Return(semaphore.NewWeighted(1))
	mockScanner.EXPECT().GetVirtualMachineScan(gomock.Any(), gomock.Any()).Return(virtualMachineScan, nil)

	enricher := New(mockScanner)

	err := enricher.EnrichVirtualMachineWithVulnerabilities(vm, indexReport)
	require.NoError(t, err)

	// Verify component details
	expectedPackages := indexReport.GetContents().GetPackages()
	require.Len(t, vm.GetScan().GetComponents(), len(expectedPackages))

	// Check first component
	var pkg *storage.EmbeddedVirtualMachineScanComponent
	for _, component := range vm.GetScan().GetComponents() {
		if component.GetName() == "openssl" {
			pkg = component
			break
		}
	}
	assert.NotNil(t, pkg)
	expectedPkg := expectedPackages["pkg-1"]
	assert.Equal(t, expectedPkg.GetName(), pkg.GetName())
	assert.Equal(t, expectedPkg.GetVersion(), pkg.GetVersion())
	assert.Len(t, pkg.GetVulnerabilities(), 2) // pkg-1 has vuln-1 and vuln-2
	// Verify vulnerability details from the vulnerability report
	criticalVuln := pkg.GetVulnerabilities()[0]
	expectedVuln := vulnReport.GetVulnerabilities()["vuln-1"]
	assert.Equal(t, expectedVuln.GetName(), criticalVuln.GetCveBaseInfo().GetCve())
	assert.Equal(t, expectedVuln.GetDescription(), criticalVuln.GetCveBaseInfo().GetSummary())
	assert.Equal(t, storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY, criticalVuln.GetSeverity())
	assert.NotNil(t, criticalVuln.GetSetFixedBy())
	if fixedBy, ok := criticalVuln.GetSetFixedBy().(*storage.VirtualMachineVulnerability_FixedBy); ok {
		assert.Equal(t, expectedVuln.GetFixedInVersion(), fixedBy.FixedBy)
	}

	// Check second component
	for _, component := range vm.GetScan().GetComponents() {
		if component.GetName() == "libssl" {
			pkg = component
			break
		}
	}
	assert.NotNil(t, pkg)
	expectedPkg = expectedPackages["pkg-2"]
	assert.Equal(t, expectedPkg.GetName(), pkg.GetName())
	assert.Equal(t, expectedPkg.GetVersion(), pkg.GetVersion())
	assert.Len(t, pkg.GetVulnerabilities(), 1) // pkg-2 has only vuln-1
}

func createVirtualMachineScanFromIndexReport(indexReport *v4.IndexReport, vulnCount int) *storage.VirtualMachineScan {
	vulnerabilityReport := createVulnerabilityReportFromIndexReport(indexReport, vulnCount)
	return scannerv4.ToVirtualMachineScan(vulnerabilityReport)
}

// createVulnerabilityReportFromIndexReport creates a vulnerability report for testing.
// It distributes the specified number of vulnerabilities across the packages in the index report
// using round-robin assignment (vuln i goes to package i % len(packages)).
// This ensures vulnerabilities are spread across components for realistic test scenarios.
func createVulnerabilityReportFromIndexReport(indexReport *v4.IndexReport, vulnCount int) *v4.VulnerabilityReport {
	vulns := make(map[string]*v4.VulnerabilityReport_Vulnerability)
	packageVulns := make(map[string]*v4.StringList)

	packages := indexReport.GetContents().GetPackages()
	if len(packages) == 0 {
		packages = map[string]*v4.Package{} // Empty packages for testing
	}

	// Create vulnerabilities and distribute them across the actual packages
	keys := slices.Collect(maps.Keys(packages))
	for i := range vulnCount {
		vulnID := fmt.Sprintf("vuln-%d", i)

		vulns[vulnID] = &v4.VulnerabilityReport_Vulnerability{
			Name:               fmt.Sprintf("CVE-2021-%04d", 1000+i),
			Description:        fmt.Sprintf("Test vulnerability %d", i),
			NormalizedSeverity: v4.VulnerabilityReport_Vulnerability_SEVERITY_MODERATE,
			Issued:             protocompat.TimestampNow(),
		}

		// Distribute vulnerabilities across available packages, or create test packages if none exist
		if len(packages) > 0 {
			pkgID := packages[keys[i%len(packages)]].GetId()
			if packageVulns[pkgID] == nil {
				packageVulns[pkgID] = &v4.StringList{Values: []string{}}
			}
			packageVulns[pkgID].Values = append(packageVulns[pkgID].Values, vulnID)
		}
	}

	return &v4.VulnerabilityReport{
		Contents:               indexReport.GetContents(),
		PackageVulnerabilities: packageVulns,
		Vulnerabilities:        vulns,
		Notes:                  []v4.VulnerabilityReport_Note{},
	}
}
