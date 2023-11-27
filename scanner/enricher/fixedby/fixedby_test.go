package fixedby_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/quay/claircore"
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
				Repositories: map[string]*claircore.Repository{
					"1": {},
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
		tc := tc
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
						Version: "7.61.1-25.el8_7.3",
					},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						FixedInVersion: "",
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
					"3": {
						FixedInVersion: "",
						Package:        &claircore.Package{},
						Dist: &claircore.Distribution{
							DID: "amzn",
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
						Version: "7.61.1-25.el8_7.3",
					},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						FixedInVersion: "7.61.2-25.el8_7.3",
						Package:        &claircore.Package{},
						Dist: &claircore.Distribution{
							DID: "amzn",
						},
					},
					"2": {
						FixedInVersion: "7.62.0-10.el8",
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
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1", "2", "3"},
				},
			},
			expected: "7.62.0-10.el8",
		},
	}

	for _, tc := range tcs {
		tc := tc
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
		tc := tc
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
						Version: "7.61.1-25.el8_7.3",
					},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						FixedInVersion: "",
						Package:        &claircore.Package{},
						Repo: &claircore.Repository{
							Key: "rhel-cpe-repository",
						},
					},
					"2": {
						FixedInVersion: "",
						Package:        &claircore.Package{},
						Repo: &claircore.Repository{
							Key: "rhel-cpe-repository",
						},
					},
					"3": {
						FixedInVersion: "",
						Package:        &claircore.Package{},
						Repo: &claircore.Repository{
							Key: "rhel-cpe-repository",
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
						Version: "7.61.1-25.el8_7.3",
					},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						FixedInVersion: "7.61.2-25.el8_7.3",
						Package:        &claircore.Package{},
						Repo: &claircore.Repository{
							Key: "rhel-cpe-repository",
						},
					},
					"2": {
						FixedInVersion: "7.62.0-10.el8",
						Package:        &claircore.Package{},
						Repo: &claircore.Repository{
							Key: "rhel-cpe-repository",
						},
					},
					"3": {
						FixedInVersion: "",
						Package:        &claircore.Package{},
						Repo: &claircore.Repository{
							Key: "rhel-cpe-repository",
						},
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1", "2", "3"},
				},
			},
			expected: "7.62.0-10.el8",
		},
	}

	for _, tc := range tcs {
		tc := tc
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
						Version: "1.2c.3",
						NormalizedVersion: claircore.Version{
							Kind: "pep440",
						},
					},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						FixedInVersion: "",
						Package: &claircore.Package{
							Version: "1.02.3",
						},
						Repo: &claircore.Repository{},
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1"},
				},
			},
			expected: "1.02.3",
		},
		{
			name: "fixed",
			report: &claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {
						Version: "1.2c.3",
						NormalizedVersion: claircore.Version{
							Kind: "pep440",
						},
					},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						FixedInVersion: "",
						Package: &claircore.Package{
							Version: "1.02.3",
						},
						Repo: &claircore.Repository{},
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1"},
				},
			},
			expected: "1.02.3",
		},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			runTest(t, tc)
		})
	}
}

func TestEnrich_Java(t *testing.T) {
	tcs := []testcase{
		{
			name: "fixed",
			report: &claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"1": {
						Version: "2.14.0",
					},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						FixedInVersion: "fixed=2.15.0&introduced=2.13.0",
						Repo: &claircore.Repository{
							Name: "maven",
						},
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1"},
				},
			},
			expected: "2.15.0",
		},
	}

	for _, tc := range tcs {
		tc := tc
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
						Version: "1:1.2.11.dfsg-4.1ubuntu1",
					},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						FixedInVersion: "",
						Package:        &claircore.Package{},
						Dist: &claircore.Distribution{
							DID: "ubuntu",
						},
					},
					"2": {
						FixedInVersion: "1:1.2.11.dfsg-4.1ubuntu1",
						Package:        &claircore.Package{},
						Dist: &claircore.Distribution{
							DID: "ubuntu",
						},
					},
					"3": {
						FixedInVersion: "0",
						Package:        &claircore.Package{},
						Dist: &claircore.Distribution{
							DID: "ubuntu",
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
						Version: "1:1.2.11.dfsg-4.1ubuntu1",
					},
				},
				Vulnerabilities: map[string]*claircore.Vulnerability{
					"1": {
						FixedInVersion: "1:1.3.11.dfsg-4.1ubuntu1",
						Package:        &claircore.Package{},
						Dist: &claircore.Distribution{
							DID: "ubuntu",
						},
					},
					"2": {
						FixedInVersion: "2:1.2.11.dfsg-4.1ubuntu1",
						Package:        &claircore.Package{},
						Dist: &claircore.Distribution{
							DID: "ubuntu",
						},
					},
					"3": {
						FixedInVersion: "0",
						Package:        &claircore.Package{},
						Dist: &claircore.Distribution{
							DID: "ubuntu",
						},
					},
					"4": {
						FixedInVersion: "",
						Package:        &claircore.Package{},
						Dist: &claircore.Distribution{
							DID: "ubuntu",
						},
					},
				},
				PackageVulnerabilities: map[string][]string{
					"1": {"1", "2", "3", "4"},
				},
			},
			expected: "2:1.2.11.dfsg-4.1ubuntu1",
		},
	}

	for _, tc := range tcs {
		tc := tc
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
