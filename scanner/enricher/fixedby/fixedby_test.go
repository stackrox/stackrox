package fixedby_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/quay/claircore"
	"github.com/quay/claircore/toolkit/types/cpe"
	"github.com/stackrox/rox/scanner/enricher/fixedby"
	"github.com/stretchr/testify/assert"
)

type testcase struct {
	name     string
	report   *claircore.VulnerabilityReport
	expected string
}

func TestEnrich_Alpine(t *testing.T) {
	tcs := []testcase{
		{
			name: "unfixed",
			report: &claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {
						Version: "2.4.55-r0",
					},
				},
				Distributions: map[string]*claircore.Distribution{
					"1": {
						DID: "alpine",
					},
				},
				Repositories: map[string]*claircore.Repository{},
				Environments: map[string][]*claircore.Environment{
					"1": {
						{
							DistributionID: "1",
						},
					},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						FixedInVersion: "",
						Package:        &claircore.Package{},
					},
					"2": {
						FixedInVersion: "2.4.54-r3",
						Package:        &claircore.Package{},
					},
					"3": {
						FixedInVersion: "0",
						Package:        &claircore.Package{},
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1", "2", "3"},
				},
			},
			expected: "",
		},
		{
			name: "fixed",
			report: &claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {
						Version: "2.4.55-r0",
					},
				},
				Distributions: map[string]*claircore.Distribution{
					"1": {
						DID: "alpine",
					},
				},
				Repositories: map[string]*claircore.Repository{},
				Environments: map[string][]*claircore.Environment{
					"1": {
						{
							DistributionID: "1",
						},
					},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						FixedInVersion: "2.4.58-r0",
						Package:        &claircore.Package{},
						Dist: &claircore.Distribution{
							DID: "alpine",
						},
					},
					"2": {
						FixedInVersion: "2.4.56-r0",
						Package:        &claircore.Package{},
						Dist: &claircore.Distribution{
							DID: "alpine",
						},
					},
					"3": {
						FixedInVersion: "0",
						Package:        &claircore.Package{},
						Dist: &claircore.Distribution{
							DID: "alpine",
						},
					},
					"4": {
						FixedInVersion: "",
						Package:        &claircore.Package{},
						Dist: &claircore.Distribution{
							DID: "alpine",
						},
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1", "2", "3", "4"},
				},
			},
			expected: "2.4.58-r0",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			runTest(t, tc)
		})
	}
}

func TestEnrich_AWS(t *testing.T) {
	tcs := []testcase{
		{
			name: "unfixed",
			report: &claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {
						Version: "3.40.0-1.amzn2023.0.4",
					},
				},
				Distributions: map[string]*claircore.Distribution{
					"1": {
						DID: "amzn",
					},
				},
				Repositories: map[string]*claircore.Repository{},
				Environments: map[string][]*claircore.Environment{
					"1": {
						{
							DistributionID: "1",
						},
					},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						FixedInVersion: "3.39.0-1.amzn2023.0.4",
						Package:        &claircore.Package{},
						Dist: &claircore.Distribution{
							DID: "amzn",
						},
					},
					"2": {
						FixedInVersion: "",
						Package:        &claircore.Package{},
						Dist: &claircore.Distribution{
							DID: "amzn",
						},
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1", "2"},
				},
			},
			expected: "",
		},
		{
			name: "fixed",
			report: &claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {
						Version: "3.40.0-1.amzn2023.0.4",
					},
				},
				Distributions: map[string]*claircore.Distribution{
					"1": {
						DID: "amzn",
					},
				},
				Repositories: map[string]*claircore.Repository{},
				Environments: map[string][]*claircore.Environment{
					"1": {
						{
							DistributionID: "1",
						},
					},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						FixedInVersion: "3.40.0-1.amzn2023.0.4",
						Package:        &claircore.Package{},
						Dist: &claircore.Distribution{
							DID: "amzn",
						},
					},
					"2": {
						FixedInVersion: "3.40.1",
						Package:        &claircore.Package{},
						Dist: &claircore.Distribution{
							DID: "amzn",
						},
					},
					"3": {
						FixedInVersion: "",
						Package:        &claircore.Package{},
						Dist: &claircore.Distribution{
							DID: "amzn",
						},
					},
					"4": {
						FixedInVersion: "3.40.0-2.amzn2023.0.4",
						Package:        &claircore.Package{},
						Dist: &claircore.Distribution{
							DID: "amzn",
						},
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1", "2", "3", "4"},
				},
			},
			expected: "3.40.1",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			runTest(t, tc)
		})
	}
}

func TestEnrich_Debian(t *testing.T) {
	tcs := []testcase{
		{
			name: "unfixed",
			report: &claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {
						Version: "2.36.1-8+deb11u1",
					},
				},
				Distributions: map[string]*claircore.Distribution{
					"1": {
						DID: "debian",
					},
				},
				Repositories: map[string]*claircore.Repository{},
				Environments: map[string][]*claircore.Environment{
					"1": {
						{
							DistributionID: "1",
						},
					},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						FixedInVersion: "",
						Package:        &claircore.Package{},
						Dist: &claircore.Distribution{
							DID: "debian",
						},
					},
					"2": {
						FixedInVersion: "2.36.1-8+deb11u1",
						Package:        &claircore.Package{},
						Dist: &claircore.Distribution{
							DID: "debian",
						},
					},
					"3": {
						FixedInVersion: "0",
						Package:        &claircore.Package{},
						Dist: &claircore.Distribution{
							DID: "debian",
						},
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1", "2", "3"},
				},
			},
			expected: "",
		},
		{
			name: "fixed",
			report: &claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {
						Version: "2.36.1-8+deb11u1",
					},
				},
				Distributions: map[string]*claircore.Distribution{
					"1": {
						DID: "debian",
					},
				},
				Repositories: map[string]*claircore.Repository{},
				Environments: map[string][]*claircore.Environment{
					"1": {
						{
							DistributionID: "1",
						},
					},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						FixedInVersion: "2.36.1-8+deb11u2",
						Package:        &claircore.Package{},
						Dist: &claircore.Distribution{
							DID: "debian",
						},
					},
					"2": {
						FixedInVersion: "2.36.1-9+deb11u1",
						Package:        &claircore.Package{},
						Dist: &claircore.Distribution{
							DID: "debian",
						},
					},
					"3": {
						FixedInVersion: "0",
						Package:        &claircore.Package{},
						Dist: &claircore.Distribution{
							DID: "debian",
						},
					},
					"4": {
						FixedInVersion: "",
						Package:        &claircore.Package{},
						Dist: &claircore.Distribution{
							DID: "debian",
						},
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1", "2", "3", "4"},
				},
			},
			expected: "2.36.1-9+deb11u1",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			runTest(t, tc)
		})
	}
}

func TestEnrich_Go(t *testing.T) {
	tcs := []testcase{
		{
			name: "unfixed",
			report: &claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {
						Version: "v0.0.0-20231002182017-d307bd883b97",
					},
				},
				Distributions: map[string]*claircore.Distribution{
					"1": {
						DID: "rhel",
					},
				},
				Repositories: map[string]*claircore.Repository{
					"1": {
						Name: "go",
						URI:  "https://pkg.go.dev/",
					},
				},
				Environments: map[string][]*claircore.Environment{
					"1": {
						{
							RepositoryIDs: []string{"1"},
						},
					},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						FixedInVersion: "",
					},
					"2": {
						FixedInVersion: "v0.0.0-20221002182017-d307bd883b97",
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1", "2"},
				},
			},
			expected: "",
		},
		{
			name: "fixed",
			report: &claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {
						Version: "v0.0.0-20231002182017-d307bd883b97",
					},
				},
				Distributions: map[string]*claircore.Distribution{
					"1": {
						DID: "rhel",
					},
				},
				Repositories: map[string]*claircore.Repository{
					"1": {
						Name: "go",
						URI:  "https://pkg.go.dev/",
					},
				},
				Environments: map[string][]*claircore.Environment{
					"1": {
						{
							RepositoryIDs: []string{"1"},
						},
					},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						FixedInVersion: "",
					},
					"2": {
						FixedInVersion: "v0.0.0-20241002182017-d307bd883b97",
					},
					"3": {
						FixedInVersion: "v0.0.0-20241002182016-d307bd883b97",
					},
					"4": {
						FixedInVersion: "v0.0.0-20241102182017-d307bd883b97",
					},
					"5": {
						FixedInVersion: "",
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1", "2", "3", "4", "5"},
				},
			},
			expected: "v0.0.0-20241102182017-d307bd883b97",
		},
		{
			name: "unfixed basic",
			report: &claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {
						Version: "v0.21.0",
					},
				},
				Distributions: map[string]*claircore.Distribution{
					"1": {
						DID: "rhel",
					},
				},
				Repositories: map[string]*claircore.Repository{
					"1": {
						Name: "go",
						URI:  "https://pkg.go.dev/",
					},
				},
				Environments: map[string][]*claircore.Environment{
					"1": {
						{
							RepositoryIDs: []string{"1"},
						},
					},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						FixedInVersion: "",
					},
					"2": {
						FixedInVersion: "v0.21.0-alpha",
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1", "2"},
				},
			},
			expected: "",
		},
		{
			name: "fixed basic",
			report: &claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {
						Version: "v0.21.0",
					},
				},
				Distributions: map[string]*claircore.Distribution{
					"1": {
						DID: "rhel",
					},
				},
				Repositories: map[string]*claircore.Repository{
					"1": {
						Name: "go",
						URI:  "https://pkg.go.dev/",
					},
				},
				Environments: map[string][]*claircore.Environment{
					"1": {
						{
							RepositoryIDs: []string{"1"},
						},
					},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						FixedInVersion: "",
					},
					"2": {
						FixedInVersion: "v0.21.1-beta",
					},
					"3": {
						FixedInVersion: "v0.21.1",
					},
					"4": {
						FixedInVersion: "v0.21.1-beta",
					},
					"5": {
						FixedInVersion: "",
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1", "2", "3", "4", "5"},
				},
			},
			expected: "v0.21.1",
		},
		{
			name: "fixed main version",
			report: &claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {
						Name:    "stdlib",
						Version: "go1.20.4",
					},
				},
				Distributions: map[string]*claircore.Distribution{
					"1": {
						DID: "rhel",
					},
				},
				Repositories: map[string]*claircore.Repository{
					"1": {
						Name: "go",
						URI:  "https://pkg.go.dev/",
					},
				},
				Environments: map[string][]*claircore.Environment{
					"1": {
						{
							RepositoryIDs: []string{"1"},
						},
					},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						FixedInVersion: "",
					},
					"2": {
						FixedInVersion: "1.20.12",
					},
					"3": {
						FixedInVersion: "v0.0.0-20241002182016-d307bd883b97",
					},
					"4": {
						FixedInVersion: "1.20.5",
					},
					"5": {
						FixedInVersion: "",
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1", "2", "3", "4", "5"},
				},
			},
			expected: "1.20.12",
		},
		{
			name: "unfixed main version",
			report: &claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {
						Name:    "stdlib",
						Version: "go1.22.5",
					},
				},
				Distributions: map[string]*claircore.Distribution{
					"1": {
						DID: "rhel",
					},
				},
				Repositories: map[string]*claircore.Repository{
					"1": {
						Name: "go",
						URI:  "https://pkg.go.dev/",
					},
				},
				Environments: map[string][]*claircore.Environment{
					"1": {
						{
							RepositoryIDs: []string{"1"},
						},
					},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						FixedInVersion: "",
					},
					"2": {
						FixedInVersion: "1.22",
					},
					"3": {
						FixedInVersion: "v0.0.0-20241002182016-d307bd883b97",
					},
					"4": {
						FixedInVersion: "1.22.5-alpha",
					},
					"5": {
						FixedInVersion: "",
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1", "2", "3", "4", "5"},
				},
			},
			expected: "",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			runTest(t, tc)
		})
	}
}

func TestEnrich_Java(t *testing.T) {
	tcs := []testcase{
		{
			name: "unfixed",
			report: &claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {
						Version: "4.1.63.Final-redhat-00001",
					},
				},
				Distributions: map[string]*claircore.Distribution{},
				Repositories: map[string]*claircore.Repository{
					"1": {
						Name: "maven",
						URI:  "https://repo1.maven.apache.org/maven2",
					},
				},
				Environments: map[string][]*claircore.Environment{
					"1": {
						{
							RepositoryIDs: []string{"1"},
						},
					},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						FixedInVersion: "",
					},
					"2": {
						FixedInVersion: "fixed=4.1.63.Final-redhat-000001",
					},
					"3": {
						FixedInVersion: "lastAffected=4.1.63.Final-redhat-00001",
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1", "2", "3"},
				},
			},
			expected: "",
		},
		{
			name: "fixed",
			report: &claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {
						Version: "4.1.63.Final-redhat-00001",
					},
				},
				Distributions: map[string]*claircore.Distribution{},
				Repositories: map[string]*claircore.Repository{
					"1": {
						Name: "maven",
						URI:  "https://repo1.maven.apache.org/maven2",
					},
				},
				Environments: map[string][]*claircore.Environment{
					"1": {
						{
							RepositoryIDs: []string{"1"},
						},
					},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						FixedInVersion: "",
					},
					"2": {
						FixedInVersion: "introduced=4.1.63.Final-redhat-00001",
					},
					"3": {
						FixedInVersion: "introduced=4.1.63.Final-redhat-00001&fixed=4.1.63.Final-redhat-00002",
					},
					"4": {
						FixedInVersion: "lastAffected=4.2.63.Final-redhat-00003",
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1", "2", "3", "4"},
				},
			},
			expected: "4.1.63.Final-redhat-00002",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			runTest(t, tc)
		})
	}
}

func TestEnrich_Nodejs(t *testing.T) {
	tcs := []testcase{
		{
			name: "unfixed",
			report: &claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {
						Version: "10.0.1",
					},
				},
				Distributions: map[string]*claircore.Distribution{
					"1": {
						DID: "rhel",
					},
				},
				Repositories: map[string]*claircore.Repository{
					"1": {
						Name: "npm",
						URI:  "https://www.npmjs.com/",
					},
				},
				Environments: map[string][]*claircore.Environment{
					"1": {
						{
							RepositoryIDs: []string{"1"},
						},
					},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						FixedInVersion: "",
					},
					"2": {
						FixedInVersion: "10.0.1-alpha",
					},
					"3": {
						FixedInVersion: "9",
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1", "2", "3"},
				},
			},
			expected: "",
		},
		{
			name: "fixed",
			report: &claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {
						Version: "10.0.1",
					},
				},
				Distributions: map[string]*claircore.Distribution{
					"1": {
						DID: "rhel",
					},
				},
				Repositories: map[string]*claircore.Repository{
					"1": {
						Name: "npm",
						URI:  "https://www.npmjs.com/",
					},
				},
				Environments: map[string][]*claircore.Environment{
					"1": {
						{
							RepositoryIDs: []string{"1"},
						},
					},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						FixedInVersion: "20",
					},
					"2": {
						FixedInVersion: "20-beta",
					},
					"3": {
						// Invalid semver.
						FixedInVersion: "20.0.0.1",
					},
					"4": {
						FixedInVersion: "10.0.2",
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1", "2", "3", "4"},
				},
			},
			expected: "20",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			runTest(t, tc)
		})
	}
}

func TestEnrich_Oracle(t *testing.T) {
	tcs := []testcase{
		{
			name: "unfixed",
			report: &claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {
						Version: "4.0.21-23.0.1.el8",
						Arch:    "x86_64",
					},
				},
				Distributions: map[string]*claircore.Distribution{
					"1": {
						DID: "ol",
					},
				},
				Repositories: map[string]*claircore.Repository{},
				Environments: map[string][]*claircore.Environment{
					"1": {{}},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						Package: &claircore.Package{
							Version: "",
							Arch:    "x86_64",
						},
						ArchOperation: claircore.OpEquals,
					},
					"2": {
						Package: &claircore.Package{
							Version: "4.0.21-23.0.1.el8",
							Arch:    "x86_64",
						},
						ArchOperation: claircore.OpEquals,
					},
					"3": {
						Package: &claircore.Package{
							Version: "4.1.21-23.0.1.el8",
							Arch:    "x86_64",
						},
						ArchOperation: claircore.OpEquals,
					},
					"4": {
						Package: &claircore.Package{
							Version: "4.0.21-24.0.1.el8",
							Arch:    "x86_64",
						},
						ArchOperation: claircore.OpEquals,
					},
					"5": {
						Package: &claircore.Package{
							Version: "0",
							Arch:    "x86_64",
						},
						ArchOperation: claircore.OpEquals,
					},
					"6": {
						FixedInVersion: "4.0.21-23.0.1.el8",
						Package: &claircore.Package{
							Version: "",
							Arch:    "x86_64",
						},
						ArchOperation: claircore.OpEquals,
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1", "2", "3", "4", "5", "6"},
				},
			},
			expected: "",
		},
		{
			name: "fixed",
			report: &claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {
						Version: "4.0.21-23.0.1.el8",
						Arch:    "x86_64",
					},
				},
				Distributions: map[string]*claircore.Distribution{
					"1": {
						DID: "ol",
					},
				},
				Repositories: map[string]*claircore.Repository{},
				Environments: map[string][]*claircore.Environment{
					"1": {{}},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						Package: &claircore.Package{
							Version: "",
							Arch:    "x86_64",
						},
						ArchOperation: claircore.OpEquals,
					},
					"2": {
						FixedInVersion: "1",
						Package: &claircore.Package{
							Version: "4.0.21-23.0.1.el8",
							Arch:    "x86_64",
						},
						ArchOperation: claircore.OpEquals,
					},
					"3": {
						FixedInVersion: "4.1.21-23.1.1.el8",
						Package: &claircore.Package{
							Version: "4.1.21-23.0.1.el8",
							Arch:    "x86_64",
						},
						ArchOperation: claircore.OpEquals,
					},
					"4": {
						Package: &claircore.Package{
							Version: "4.0.21-24.0.1.el8",
							Arch:    "x86_64",
						},
						ArchOperation: claircore.OpEquals,
					},
					"5": {
						FixedInVersion: "0",
						Package: &claircore.Package{
							Version: "4.1.21-24.1.1.el8",
							Arch:    "x86_64",
						},
						ArchOperation: claircore.OpEquals,
					},
					"6": {
						FixedInVersion: "4.1.21-23.0.2.el8",
						Package: &claircore.Package{
							Version: "",
							Arch:    "x86_64",
						},
						ArchOperation: claircore.OpEquals,
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1", "2", "3", "4", "5", "6"},
				},
			},
			expected: "4.1.21-23.1.1.el8",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			runTest(t, tc)
		})
	}
}

func TestEnrich_Photon(t *testing.T) {
	tcs := []testcase{
		{
			name: "unfixed",
			report: &claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {
						Version: "1.0.8-2.ph3",
					},
				},
				Distributions: map[string]*claircore.Distribution{
					"1": {
						DID: "photon",
					},
				},
				Repositories: map[string]*claircore.Repository{},
				Environments: map[string][]*claircore.Environment{
					"1": {{}},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						Package: &claircore.Package{Version: ""},
					},
					"2": {
						Package: &claircore.Package{Version: "1.0.8-2.ph4"},
					},
					"3": {
						Package: &claircore.Package{Version: "1.0.8-2.ph2"},
					},
					"4": {
						Package: &claircore.Package{Version: "1.0.8-2.ph3"},
					},
					"5": {
						Package: &claircore.Package{Version: "0"},
					},
					"6": {
						FixedInVersion: "1.0.8-2.ph2",
						Package:        &claircore.Package{Version: ""},
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1", "2", "3", "4", "5", "6"},
				},
			},
			expected: "",
		},
		{
			name: "fixed",
			report: &claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {
						Version: "1.0.8-2.ph3",
					},
				},
				Distributions: map[string]*claircore.Distribution{
					"1": {
						DID: "photon",
					},
				},
				Repositories: map[string]*claircore.Repository{},
				Environments: map[string][]*claircore.Environment{
					"1": {
						{
							DistributionID: "1",
						},
					},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						Package: &claircore.Package{Version: ""},
					},
					"2": {
						FixedInVersion: "1.0.8-2.ph2",
						Package:        &claircore.Package{Version: ""},
					},
					"3": {
						FixedInVersion: "1.0.8-6.ph5",
						Package:        &claircore.Package{Version: ""},
					},
					"4": {
						Package: &claircore.Package{Version: "0"},
					},
					"5": {
						FixedInVersion: "1.0.8-2.ph5",
						Package:        &claircore.Package{Version: ""},
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1", "2", "3", "4", "5"},
				},
			},
			expected: "1.0.8-6.ph5",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			runTest(t, tc)
		})
	}
}

func TestEnrich_Python(t *testing.T) {
	tcs := []testcase{
		{
			name: "unfixed",
			report: &claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {
						Version: "23.2.1",
						NormalizedVersion: claircore.Version{
							Kind: "pep440",
						},
					},
				},
				Distributions: map[string]*claircore.Distribution{
					"1": {
						DID: "debian",
					},
				},
				Repositories: map[string]*claircore.Repository{
					"1": {
						Name: "pypi",
						URI:  "https://pypi.org/simple",
					},
				},
				Environments: map[string][]*claircore.Environment{
					"1": {
						{
							RepositoryIDs: []string{"1"},
						},
					},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						FixedInVersion: "",
					},
					"2": {
						FixedInVersion: "introduced=23.2.1",
					},
					"3": {
						FixedInVersion: "lastAffected=23.2.1",
					},
					"4": {
						FixedInVersion: "fixed=23.2.1c",
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1", "2", "3", "4"},
				},
			},
			expected: "",
		},
		{
			name: "fixed",
			report: &claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {
						Version: "23.2.1",
						NormalizedVersion: claircore.Version{
							Kind: "pep440",
						},
					},
				},
				Distributions: map[string]*claircore.Distribution{
					"1": {
						DID: "debian",
					},
				},
				Repositories: map[string]*claircore.Repository{
					"1": {
						Name: "pypi",
						URI:  "https://pypi.org/simple",
					},
				},
				Environments: map[string][]*claircore.Environment{
					"1": {
						{
							RepositoryIDs: []string{"1"},
						},
					},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						FixedInVersion: "",
					},
					"2": {
						FixedInVersion: "introduced=23.2.1",
					},
					"3": {
						FixedInVersion: "fixed=24.0.0a",
					},
					"4": {
						FixedInVersion: "lastAffected=23.2.1",
					},
					"5": {
						FixedInVersion: "introduced=23.2.1&lastAffected=25.0.0",
					},
					"6": {
						FixedInVersion: "fixed=23.2.1c",
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1", "2", "3", "4", "5", "6"},
				},
			},
			expected: "24.0.0a",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			runTest(t, tc)
		})
	}
}

func TestEnrich_RHCC(t *testing.T) {
	tcs := []testcase{
		{
			name: "unfixed",
			report: &claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {
						Version: "8.8-1072.1697626218",
					},
				},
				Distributions: map[string]*claircore.Distribution{
					"1": {
						DID: "rhel",
					},
				},
				Repositories: map[string]*claircore.Repository{
					"1": {
						Name: "Red Hat Container Catalog",
						URI:  "https://catalog.redhat.com/software/containers/explore",
					},
				},
				Environments: map[string][]*claircore.Environment{
					"1": {{
						RepositoryIDs: []string{"1"},
					}},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						FixedInVersion: "",
					},
					"2": {
						FixedInVersion: "8.8-1072.1697626208",
					},
					"3": {
						FixedInVersion: "0",
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1", "2", "3"},
				},
			},
			expected: "",
		},
		{
			name: "fixed",
			report: &claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {
						Version: "8.8-1072.1697626218",
					},
				},
				Distributions: map[string]*claircore.Distribution{
					"1": {
						DID: "rhel",
					},
				},
				Repositories: map[string]*claircore.Repository{
					"1": {
						Name: "Red Hat Container Catalog",
						URI:  "https://catalog.redhat.com/software/containers/explore",
					},
				},
				Environments: map[string][]*claircore.Environment{
					"1": {{
						RepositoryIDs: []string{"1"},
					}},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						FixedInVersion: "",
					},
					"2": {
						FixedInVersion: "8.8-1072.1697626248",
					},
					"3": {
						FixedInVersion: "0",
					},
					"4": {
						FixedInVersion: "8.8-1172.1697626248",
					},
					"5": {
						FixedInVersion: "8.8-1072.1697626218",
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1", "2", "3", "4", "5"},
				},
			},
			expected: "8.8-1172.1697626248",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			runTest(t, tc)
		})
	}
}

func TestEnrich_RHEL(t *testing.T) {
	tcs := []testcase{
		{
			name: "unfixed",
			report: &claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {
						Version: "1.12.8-26.el8",
						Arch:    "x86_64",
					},
				},
				Distributions: map[string]*claircore.Distribution{
					"1": {
						DID: "rhel",
					},
				},
				Repositories: map[string]*claircore.Repository{
					"1": {
						Key: "rhel-cpe-repository",
					},
					"2": {
						Key: "rhel-cpe-repository",
					},
					"3": {
						Key: "rhel-cpe-repository",
					},
					"4": {
						Key: "rhel-cpe-repository",
					},
				},
				Environments: map[string][]*claircore.Environment{
					"1": {{RepositoryIDs: []string{"1", "2", "3", "4"}}},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						FixedInVersion: "",
						Package:        &claircore.Package{Arch: "x86_64"},
						ArchOperation:  claircore.OpEquals,
					},
					"2": {
						FixedInVersion: "1.12.7-26.el8",
						Package:        &claircore.Package{Arch: "x86_64"},
						ArchOperation:  claircore.OpEquals,
					},
					"3": {
						FixedInVersion: "0",
						Package:        &claircore.Package{Arch: "x86_64"},
						ArchOperation:  claircore.OpEquals,
					},
					"4": {
						FixedInVersion: "1.11.8-26.el8",
						Package:        &claircore.Package{Arch: "x86_64"},
						ArchOperation:  claircore.OpEquals,
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1", "2", "3", "4"},
				},
			},
			expected: "",
		},
		{
			name: "fixed",
			report: &claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {
						Version: "1.12.8-26.el8",
						Arch:    "x86_64",
					},
				},
				Distributions: map[string]*claircore.Distribution{
					"1": {
						DID: "rhel",
					},
				},
				Repositories: map[string]*claircore.Repository{
					"1": {
						Key: "rhel-cpe-repository",
						CPE: cpe.MustUnbind("cpe:/o:redhat:enterprise_linux:9::baseos"),
					},
					"2": {
						Key: "rhel-cpe-repository",
						CPE: cpe.MustUnbind("cpe:/o:redhat:enterprise_linux:9::baseos"),
					},
					"3": {
						Key: "rhel-cpe-repository",
						CPE: cpe.MustUnbind("cpe:/o:redhat:enterprise_linux:9::baseos"),
					},
					"4": {
						Key: "rhel-cpe-repository",
						CPE: cpe.MustUnbind("cpe:/o:redhat:enterprise_linux:9::baseos"),
					},
				},
				Environments: map[string][]*claircore.Environment{
					"1": {{RepositoryIDs: []string{"1", "2", "3", "4"}}},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						FixedInVersion: "",
						Package:        &claircore.Package{Arch: "x86_64"},
						ArchOperation:  claircore.OpEquals,
						Repo: &claircore.Repository{
							Key:  "rhel-cpe-repository",
							Name: cpe.MustUnbind("cpe:/o:redhat:enterprise_linux:9::baseos").String(),
						},
					},
					"2": {
						FixedInVersion: "1.13.7-26.el8",
						Package:        &claircore.Package{Arch: "x86_64"},
						ArchOperation:  claircore.OpEquals,
						Repo: &claircore.Repository{
							Key:  "rhel-cpe-repository",
							Name: cpe.MustUnbind("cpe:/o:redhat:enterprise_linux:9::baseos").String(),
						},
					},
					"3": {
						FixedInVersion: "0",
						Package:        &claircore.Package{Arch: "x86_64"},
						ArchOperation:  claircore.OpEquals,
						Repo: &claircore.Repository{
							Key:  "rhel-cpe-repository",
							Name: cpe.MustUnbind("cpe:/o:redhat:enterprise_linux:9::baseos").String(),
						},
					},
					"4": {
						FixedInVersion: "1.14.8-26.el8",
						Package:        &claircore.Package{Arch: "x86_64"},
						ArchOperation:  claircore.OpEquals,
						Repo: &claircore.Repository{
							Key:  "rhel-cpe-repository",
							Name: cpe.MustUnbind("cpe:/o:redhat:enterprise_linux:9::baseos").String(),
						},
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1", "2", "3", "4"},
				},
			},
			expected: "1.14.8-26.el8",
		},
		{
			name: "fixed openshift",
			report: &claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {
						Version: "4.10.1650890594-1.el8",
						Arch:    "noarch",
					},
				},
				Distributions: map[string]*claircore.Distribution{
					"1": {
						DID: "rhel",
					},
				},
				Repositories: map[string]*claircore.Repository{
					"1": {
						Key: "rhel-cpe-repository",
						CPE: cpe.MustUnbind("cpe:/a:redhat:openshift:4.10::el8"),
					},
				},
				Environments: map[string][]*claircore.Environment{
					"1": {{RepositoryIDs: []string{"1"}}},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						FixedInVersion: "",
						Package:        &claircore.Package{Arch: "noarch"},
						ArchOperation:  claircore.OpEquals,
						Repo: &claircore.Repository{
							Key:  "rhel-cpe-repository",
							Name: cpe.MustUnbind("cpe:/a:redhat:openshift:4").String(),
						},
					},
					"2": {
						FixedInVersion: "4.10.1685679861-1.el8",
						Package:        &claircore.Package{Arch: "noarch"},
						ArchOperation:  claircore.OpEquals,
						Repo: &claircore.Repository{
							Key:  "rhel-cpe-repository",
							Name: cpe.MustUnbind("cpe:/a:redhat:openshift:4.10::el8").String(),
						},
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1", "2"},
				},
			},
			expected: "4.10.1685679861-1.el8",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			runTest(t, tc)
		})
	}
}

func TestEnrich_Ruby(t *testing.T) {
	tcs := []testcase{
		{
			name: "unfixed",
			report: &claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {
						Version: "1",
					},
				},
				Distributions: map[string]*claircore.Distribution{
					"1": {
						DID: "debian",
					},
				},
				Repositories: map[string]*claircore.Repository{
					"1": {
						Name: "rubygems",
						URI:  "https://rubygems.org/gems/",
					},
				},
				Environments: map[string][]*claircore.Environment{
					"1": {
						{
							RepositoryIDs: []string{"1"},
						},
					},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						FixedInVersion: "",
					},
					"2": {
						FixedInVersion: "introduced=0&lastAffected=1",
					},
					"3": {
						FixedInVersion: "introduced=0&fixed=1.alpha",
					},
					"4": {
						FixedInVersion: "fixed=0-0",
					},
					"5": {
						FixedInVersion: "fixed=1-234567890987654321234567890987654321234567890987654321234567890987654.3.21",
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1", "2", "3", "4", "5"},
				},
			},
			expected: "",
		},
		{
			name: "fixed",
			report: &claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {
						Version: "1",
					},
				},
				Distributions: map[string]*claircore.Distribution{
					"1": {
						DID: "debian",
					},
				},
				Repositories: map[string]*claircore.Repository{
					"1": {
						Name: "rubygems",
						URI:  "https://rubygems.org/gems/",
					},
				},
				Environments: map[string][]*claircore.Environment{
					"1": {
						{
							RepositoryIDs: []string{"1"},
						},
					},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						FixedInVersion: "fixed=2",
					},
					"2": {
						FixedInVersion: "introduced=0&lastAffected=1",
					},
					"3": {
						FixedInVersion: "introduced=0&fixed=1.alpha",
					},
					"4": {
						FixedInVersion: "fixed=0-0",
					},
					"5": {
						FixedInVersion: "fixed=1-234567890987654321234567890987654321234567890987654321234567890987654.3.21",
					},
					"6": {
						FixedInVersion: "introduced=1&lastAffected=5",
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1", "2", "3", "4", "5", "6"},
				},
			},
			expected: "2",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			runTest(t, tc)
		})
	}
}

func TestEnrich_SUSE(t *testing.T) {
	tcs := []testcase{
		{
			name: "unfixed",
			report: &claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {
						Version: "1.1.0i-lp151.8.12.2",
					},
				},
				Distributions: map[string]*claircore.Distribution{
					"1": {
						DID: "opensuse-leap",
					},
				},
				Repositories: map[string]*claircore.Repository{},
				Environments: map[string][]*claircore.Environment{
					"1": {{}},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						Package: &claircore.Package{Version: ""},
					},
					"2": {
						Package: &claircore.Package{Version: "1.1.0i-lp151.8.12.1"},
					},
					"3": {
						Package: &claircore.Package{Version: "0"},
					},
					"4": {
						FixedInVersion: "1.1.0i-lp151.8.12.0",
						Package:        &claircore.Package{Version: "1.1.0i-lp151.8.12.1"},
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1", "2", "3", "4"},
				},
			},
			expected: "",
		},
		{
			name: "fixed",
			report: &claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {
						Version: "1.1.0i-lp151.8.12.2",
					},
				},
				Distributions: map[string]*claircore.Distribution{
					"1": {
						DID: "opensuse-leap",
					},
				},
				Repositories: map[string]*claircore.Repository{},
				Environments: map[string][]*claircore.Environment{
					"1": {
						{
							DistributionID: "1",
						},
					},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						Package: &claircore.Package{Version: ""},
					},
					"2": {
						FixedInVersion: "1.1.0i-lp151.8.12.1",
						Package:        &claircore.Package{Version: ""},
					},
					"3": {
						FixedInVersion: "1.1.0j-lp151.8.12.2",
						Package:        &claircore.Package{Version: ""},
					},
					"4": {
						Package: &claircore.Package{Version: "0"},
					},
					"5": {
						FixedInVersion: "1.1.0j-lp151.8.11.0",
						Package:        &claircore.Package{Version: ""},
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1", "2", "3", "4", "5"},
				},
			},
			expected: "1.1.0j-lp151.8.12.2",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			runTest(t, tc)
		})
	}
}

func TestEnrich_Ubuntu(t *testing.T) {
	tcs := []testcase{
		{
			name: "unfixed",
			report: &claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {
						Version: "1.0.8-5build1",
					},
				},
				Distributions: map[string]*claircore.Distribution{
					"1": {
						DID: "ubuntu",
					},
				},
				Repositories: map[string]*claircore.Repository{},
				Environments: map[string][]*claircore.Environment{
					"1": {
						{
							DistributionID: "1",
						},
					},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						FixedInVersion: "",
					},
					"2": {
						FixedInVersion: "1.0.8-4",
					},
					"3": {
						FixedInVersion: "0",
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1", "2", "3"},
				},
			},
			expected: "",
		},
		{
			name: "fixed",
			report: &claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {
						Version: "1.0.8-5build1",
					},
				},
				Distributions: map[string]*claircore.Distribution{
					"1": {
						DID: "ubuntu",
					},
				},
				Repositories: map[string]*claircore.Repository{},
				Environments: map[string][]*claircore.Environment{
					"1": {
						{
							DistributionID: "1",
						},
					},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						FixedInVersion: "1.0.9-2ubuntu1",
					},
					"2": {
						FixedInVersion: "1.0.8-2ubuntu1",
					},
					"3": {
						FixedInVersion: "0",
					},
					"4": {
						FixedInVersion: "",
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1", "2", "3", "4"},
				},
			},
			expected: "1.0.9-2ubuntu1",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			runTest(t, tc)
		})
	}
}

func runTest(t *testing.T, tc testcase) {
	var e fixedby.Enricher
	_, m, err := e.Enrich(context.Background(), nil, tc.report)
	assert.NoError(t, err)

	got := make(map[string]string)
	err = json.Unmarshal(m[0], &got)
	assert.NoError(t, err)

	assert.Len(t, got, 1)
	assert.Equal(t, tc.expected, got["1"])
}
