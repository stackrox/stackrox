package enricher

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/pkg/errors"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/scannerv4/client/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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
					Packages: []*v4.Package{
						{
							Id:      "pkg-1",
							Name:    "libssl",
							Version: "1.1.1",
						},
						{
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
					Packages: []*v4.Package{
						{
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
					Packages: []*v4.Package{},
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
					Packages: []*v4.Package{
						{
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
					Packages: []*v4.Package{
						{
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

			// Create vulnerability report with expected vulnerabilities and notes
			vulnReport := createVulnerabilityReportFromIndexReport(tt.indexReport, tt.expectedVulnerabilitiesCount)

			// Add notes based on expected count
			vulnReport.Notes = make([]v4.VulnerabilityReport_Note, tt.expectedScanNotesCount)
			for i := range tt.expectedScanNotesCount {
				// Cycle through available note types
				noteTypes := []v4.VulnerabilityReport_Note{
					v4.VulnerabilityReport_NOTE_OS_UNKNOWN,
					v4.VulnerabilityReport_NOTE_OS_UNSUPPORTED,
					v4.VulnerabilityReport_NOTE_UNSPECIFIED,
				}
				vulnReport.Notes[i] = noteTypes[i%len(noteTypes)]
			}

			mockClient := mocks.NewMockScanner(ctrl)
			mockClient.EXPECT().GetVulnerabilities(gomock.Any(), gomock.Any(), gomock.Any()).Return(vulnReport, nil)

			enricher := New(mockClient).(*enricherImpl)

			err := enricher.EnrichVirtualMachineWithVulnerabilities(tt.vm, tt.indexReport)
			require.NoError(t, err)

			// Verify the scan was created
			require.NotNil(t, tt.vm.Scan)
			expectedComponentsCount := len(tt.indexReport.GetContents().GetPackages())
			assert.Equal(t, expectedComponentsCount, len(tt.vm.Scan.Components))

			// Count total vulnerabilities across all components
			totalVulns := 0
			for _, component := range tt.vm.Scan.Components {
				totalVulns += len(component.Vulnerabilities)
			}
			assert.Equal(t, tt.expectedVulnerabilitiesCount, totalVulns)

			// Verify VM notes are cleared
			assert.Empty(t, tt.vm.Notes)

			// Verify scan notes count
			assert.Len(t, tt.vm.Scan.Notes, tt.expectedScanNotesCount)

			// Verify scan metadata
			assert.NotNil(t, tt.vm.Scan.ScanTime)
			assert.Equal(t, "", tt.vm.Scan.OperatingSystem)
		})
	}
}

func TestEnrichVirtualMachineWithVulnerabilities_Errors(t *testing.T) {
	tests := []struct {
		name          string
		vm            *storage.VirtualMachine
		indexReport   *v4.IndexReport
		setupMock     func(*mocks.MockScanner)
		useNilClient  bool
		expectedError string
		expectedNotes []storage.VirtualMachine_Note
	}{
		{
			name: "nil scanner client",
			vm: &storage.VirtualMachine{
				Id:   "vm-id",
				Name: "test-vm",
			},
			indexReport: &v4.IndexReport{
				Contents: &v4.Contents{},
			},
			useNilClient:  true,
			expectedError: "Scanner V4 client not available for VM enrichment",
			expectedNotes: []storage.VirtualMachine_Note{storage.VirtualMachine_MISSING_SCAN_DATA},
		},
		{
			name: "nil index report",
			vm: &storage.VirtualMachine{
				Id:   "vm-id",
				Name: "test-vm",
			},
			indexReport: nil,
			setupMock: func(mockClient *mocks.MockScanner) {
				// No expectations since we should fail before calling scanner
			},
			expectedError: "index report is required for VM scanning",
			expectedNotes: []storage.VirtualMachine_Note{storage.VirtualMachine_MISSING_SCAN_DATA},
		},
		{
			name: "scanner client returns error",
			vm: &storage.VirtualMachine{
				Id:   "vm-id",
				Name: "test-vm",
			},
			indexReport: &v4.IndexReport{
				Contents: &v4.Contents{},
			},
			setupMock: func(mockClient *mocks.MockScanner) {
				mockClient.EXPECT().GetVulnerabilities(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("scanner service unavailable"))
			},
			expectedError: "failed to get vulnerability report for VM",
			expectedNotes: []storage.VirtualMachine_Note{storage.VirtualMachine_MISSING_SCAN_DATA},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			var enricher *enricherImpl
			if tt.useNilClient {
				enricher = New(nil).(*enricherImpl)
			} else {
				mockClient := mocks.NewMockScanner(ctrl)
				if tt.setupMock != nil {
					tt.setupMock(mockClient)
				}
				enricher = New(mockClient).(*enricherImpl)
			}

			err := enricher.EnrichVirtualMachineWithVulnerabilities(tt.vm, tt.indexReport)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
			assert.Equal(t, tt.expectedNotes, tt.vm.Notes)
		})
	}
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
			Packages: []*v4.Package{
				{
					Id:      "pkg-1",
					Name:    "test-package",
					Version: "1.0.0",
				},
			},
		},
	}

	vulnReport := createVulnerabilityReportFromIndexReport(indexReport, 0)
	mockClient := mocks.NewMockScanner(ctrl)
	mockClient.EXPECT().GetVulnerabilities(gomock.Any(), gomock.Any(), gomock.Any()).Return(vulnReport, nil)

	enricher := New(mockClient).(*enricherImpl)

	err := enricher.EnrichVirtualMachineWithVulnerabilities(vm, indexReport)
	require.NoError(t, err)

	// Verify that pre-existing notes are cleared
	assert.Empty(t, vm.Notes)
	assert.NotNil(t, vm.Scan)
}

func TestVMDigestParsing(t *testing.T) {
	// Test that the VM mock digest can be parsed correctly
	digest, err := name.NewDigest(vmMockDigest)
	require.NoError(t, err)
	assert.Contains(t, digest.String(), "vm-registry/repository@sha256:")
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
			Packages: []*v4.Package{
				{
					Id:      "pkg-1",
					Name:    "openssl",
					Version: "1.1.1f",
				},
				{
					Id:      "pkg-2",
					Name:    "libssl",
					Version: "1.1.1f",
				},
			},
		},
	}

	vulnReport := &v4.VulnerabilityReport{
		Contents: indexReport.Contents,
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

	mockClient := mocks.NewMockScanner(ctrl)
	mockClient.EXPECT().GetVulnerabilities(gomock.Any(), gomock.Any(), gomock.Any()).Return(vulnReport, nil)

	enricher := New(mockClient).(*enricherImpl)

	err := enricher.EnrichVirtualMachineWithVulnerabilities(vm, indexReport)
	require.NoError(t, err)

	// Verify component details
	expectedPackages := indexReport.Contents.Packages
	require.Len(t, vm.Scan.Components, len(expectedPackages))

	// Check first component
	firstComponent := vm.Scan.Components[0]
	assert.Equal(t, expectedPackages[0].Name, firstComponent.Name)
	assert.Equal(t, expectedPackages[0].Version, firstComponent.Version)
	assert.Len(t, firstComponent.Vulnerabilities, 2) // pkg-1 has vuln-1 and vuln-2

	// Check second component
	secondComponent := vm.Scan.Components[1]
	assert.Equal(t, expectedPackages[1].Name, secondComponent.Name)
	assert.Equal(t, expectedPackages[1].Version, secondComponent.Version)
	assert.Len(t, secondComponent.Vulnerabilities, 1) // pkg-2 has only vuln-1

	// Verify vulnerability details from the vulnerability report
	criticalVuln := firstComponent.Vulnerabilities[0]
	expectedVuln := vulnReport.Vulnerabilities["vuln-1"]
	assert.Equal(t, expectedVuln.Name, criticalVuln.GetCveBaseInfo().GetCve())
	assert.Equal(t, expectedVuln.Description, criticalVuln.GetCveBaseInfo().GetSummary())
	assert.Equal(t, storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY, criticalVuln.Severity)
	assert.NotNil(t, criticalVuln.SetFixedBy)
	if fixedBy, ok := criticalVuln.SetFixedBy.(*storage.VirtualMachineVulnerability_FixedBy); ok {
		assert.Equal(t, expectedVuln.FixedInVersion, fixedBy.FixedBy)
	}
}

func TestEnrichVirtualMachineWithVulnerabilities_Timeout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	vm := &storage.VirtualMachine{
		Id:   "vm-id",
		Name: "test-vm",
	}

	indexReport := &v4.IndexReport{
		Contents: &v4.Contents{},
	}

	// Create a scanner client that simulates timeout
	mockClient := mocks.NewMockScanner(ctrl)
	mockClient.EXPECT().GetVulnerabilities(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, context.DeadlineExceeded)

	enricher := New(mockClient).(*enricherImpl)

	err := enricher.EnrichVirtualMachineWithVulnerabilities(vm, indexReport)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get vulnerability report for VM")
	assert.Equal(t, []storage.VirtualMachine_Note{storage.VirtualMachine_MISSING_SCAN_DATA}, vm.Notes)
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
		packages = []*v4.Package{} // Empty packages for testing
	}

	// Create vulnerabilities and distribute them across the actual packages
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
			pkgID := packages[i%len(packages)].GetId()
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
