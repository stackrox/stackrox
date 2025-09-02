package notaffected

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/klauspost/compress/snappy"
	"github.com/package-url/packageurl-go"
	"github.com/quay/claircore/toolkit/types/csaf"
)

func TestParse_Single(t *testing.T) {
	testcases := []struct {
		name     string
		path     string
		expected int
	}{
		{
			name:     "basic 1",
			path:     "testdata/cve-2024-7246.jsonl",
			expected: 473,
		},
		{
			name:     "basic 2",
			path:     "testdata/cve-2025-7783.jsonl",
			expected: 27,
		},
		{
			name:     "no known_not_affected",
			path:     "testdata/cve-2024-57083.jsonl",
			expected: 0,
		},
		{
			name:     "red_hat_products",
			path:     "testdata/cve-2024-21613.jsonl",
			expected: 1,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			f, err := os.Open(tc.path)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				_ = f.Close()
			})

			b, err := io.ReadAll(f)
			if err != nil {
				t.Fatalf("failed to read file bytes: %v", err)
			}
			var buf bytes.Buffer
			sw := snappy.NewBufferedWriter(&buf)
			bLen, err := sw.Write(b)
			if err != nil {
				t.Fatalf("error writing snappy data to buffer: %v", err)
			}
			if bLen != len(b) {
				t.Errorf("didn't write the correct # of bytes")
			}
			if err = sw.Close(); err != nil {
				t.Errorf("failed to close snappy Writer: %v", err)
			}

			e := &Enricher{}
			rs, err := e.ParseEnrichment(t.Context(), io.NopCloser(&buf))
			if err != nil {
				t.Error(err)
			}

			want, got := tc.expected, len(rs)
			if want != got {
				t.Errorf("want %d records, got %d", want, got)
			}
		})
	}
}

func TestParse_Multiple(t *testing.T) {
	f0, err := os.Open("testdata/cve-2024-7246.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = f0.Close()
	})

	f1, err := os.Open("testdata/cve-2025-7783.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = f1.Close()
	})

	b0, err := io.ReadAll(f0)
	if err != nil {
		t.Fatalf("failed to read file bytes: %v", err)
	}
	b1, err := io.ReadAll(f1)
	if err != nil {
		t.Fatalf("failed to read file bytes: %v", err)
	}

	var buf bytes.Buffer
	sw := snappy.NewBufferedWriter(&buf)
	bLen, err := sw.Write(b0)
	if err != nil {
		t.Fatalf("error writing snappy data to buffer: %v", err)
	}
	if bLen != len(b0) {
		t.Errorf("didn't write the correct # of bytes")
	}
	bLen, err = sw.Write(b1)
	if err != nil {
		t.Fatalf("error writing snappy data to buffer: %v", err)
	}
	if bLen != len(b1) {
		t.Errorf("didn't write the correct # of bytes")
	}
	if err = sw.Close(); err != nil {
		t.Errorf("failed to close snappy Writer: %v", err)
	}

	e := &Enricher{}
	rs, err := e.ParseEnrichment(t.Context(), io.NopCloser(&buf))
	if err != nil {
		t.Error(err)
	}

	// Note: this is less than the total sum of the two separate known_not_affected list lengths.
	// That is because there are some products each vulnerability has in common.
	want, got := 494, len(rs)
	if want != got {
		t.Errorf("want %d records, got %d", want, got)
	}

	var oneCVE, twoCVE int
	for _, r := range rs {
		//if strings.HasPrefix(r.Tags[0], "advanced-cluster-security/rhacs-scanner-v4-rhel8") {
		//	t.Log(r.Tags[0])
		//}
		var cves []string
		err := json.Unmarshal(r.Enrichment, &cves)
		if err != nil {
			t.Fatalf("failed to unmarshal enrichment: %v", err)
		}
		switch len(cves) {
		case 1:
			oneCVE++
		case 2:
			t.Log(r.Tags[0])
			twoCVE++
		default:
			t.Errorf("invalid number of CVEs: %d", len(cves))
		}
	}
	t.Log(oneCVE, twoCVE)
}

func TestWalkRelationships(t *testing.T) {
	testcases := []struct {
		name            string
		in              string
		c               *csaf.CSAF
		expectedPkgName string
		err             bool
	}{
		{
			c: &csaf.CSAF{
				ProductTree: csaf.ProductBranch{},
			},
			in:              "EAP 7.4 log4j async",
			expectedPkgName: "",
			name:            "no_relationship",
			err:             true,
		},
		{
			c: &csaf.CSAF{
				ProductTree: csaf.ProductBranch{
					Relationships: csaf.Relationships{
						csaf.Relationship{
							Category: "default_component_of",
							FullProductName: csaf.Product{
								Name: "advanced-cluster-security/rhacs-scanner-rhel8 as a component of Red Hat Advanced Cluster Security 4",
								ID:   "red_hat_advanced_cluster_security_4:advanced-cluster-security/rhacs-scanner-rhel8",
							},
							ProductRef:          "advanced-cluster-security/rhacs-scanner-rhel8",
							RelatesToProductRef: "red_hat_advanced_cluster_security_4",
						},
					},
				},
			},
			in:              "red_hat_advanced_cluster_security_4:advanced-cluster-security/rhacs-scanner-rhel8",
			expectedPkgName: "advanced-cluster-security/rhacs-scanner-rhel8",
			name:            "simple_source_oci_relationship",
		},
		{
			c: &csaf.CSAF{
				ProductTree: csaf.ProductBranch{
					Relationships: csaf.Relationships{
						csaf.Relationship{
							Category: "default_component_of",
							FullProductName: csaf.Product{
								Name: "openshift4/network-tools-rhel8@sha256:0400e3a56ac366267783941486eaa58970f2c27fa669c9eb325a290583320c13_arm64 as a component of Red Hat OpenShift Container Platform 4.14",
								ID:   "8Base-RHOSE-4.14:openshift4/network-tools-rhel8@sha256:0400e3a56ac366267783941486eaa58970f2c27fa669c9eb325a290583320c13_arm64",
							},
							ProductRef:          "openshift4/network-tools-rhel8@sha256:0400e3a56ac366267783941486eaa58970f2c27fa669c9eb325a290583320c13_arm64",
							RelatesToProductRef: "8Base-RHOSE-4.14",
						},
					},
				},
			},
			in:              "8Base-RHOSE-4.14:openshift4/network-tools-rhel8@sha256:0400e3a56ac366267783941486eaa58970f2c27fa669c9eb325a290583320c13_arm64",
			expectedPkgName: "openshift4/network-tools-rhel8@sha256:0400e3a56ac366267783941486eaa58970f2c27fa669c9eb325a290583320c13_arm64",
			name:            "simple_binary_oci_relationship",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			pkgName, err := walkRelationships(tc.in, tc.c)
			if err != nil && !tc.err {
				t.Errorf("expected no error but got %q", err)
			}
			if pkgName != tc.expectedPkgName {
				t.Errorf("expected %s but got %s", tc.expectedPkgName, pkgName)
			}
		})
	}
}

func TestExtractPackageName(t *testing.T) {
	testcases := []struct {
		name        string
		purl        packageurl.PackageURL
		expectedErr bool
		want        string
	}{
		{
			name: "oci_with_repository_url",
			purl: packageurl.PackageURL{
				Type:      packageurl.TypeOCI,
				Namespace: "",
				Name:      "keepalived-rhel9",
				Version:   "sha256:36abd2b22ebabea813c5afde35b0b80a200056f811267e89f0270da9155b1a22",
				Qualifiers: packageurl.QualifiersFromMap(map[string]string{
					"arch":           "ppc64le",
					"repository_url": "registry.redhat.io/rhceph/keepalived-rhel9",
					"tag":            "2.2.4-3",
				}),
			},
			want: "rhceph/keepalived-rhel9",
		},
		{
			name: "oci_without_repository_url",
			purl: packageurl.PackageURL{
				Type:      packageurl.TypeOCI,
				Namespace: "",
				Name:      "keepalived-rhel9",
				Version:   "sha256:36abd2b22ebabea813c5afde35b0b80a200056f811267e89f0270da9155b1a22",
				Qualifiers: packageurl.QualifiersFromMap(map[string]string{
					"arch": "ppc64le",
				}),
			},
			want: "keepalived-rhel9",
		},
		{
			name: "oci_invalid_repository_url",
			purl: packageurl.PackageURL{
				Type:      packageurl.TypeOCI,
				Namespace: "",
				Name:      "keepalived-rhel9",
				Version:   "sha256:36abd2b22ebabea813c5afde35b0b80a200056f811267e89f0270da9155b1a22",
				Qualifiers: packageurl.QualifiersFromMap(map[string]string{
					"arch":           "ppc64le",
					"repository_url": "registry.redhat.iorhcephkeepalived-rhel9",
				}),
			},
			expectedErr: true,
		},
		{
			name: "repository_url_with_namespace",
			purl: packageurl.PackageURL{
				Type:      packageurl.TypeOCI,
				Namespace: "something",
				Name:      "keepalived-rhel9",
				Version:   "sha256:36abd2b22ebabea813c5afde35b0b80a200056f811267e89f0270da9155b1a22",
				Qualifiers: packageurl.QualifiersFromMap(map[string]string{
					"arch":           "ppc64le",
					"repository_url": "registry.redhat.io/rhceph/keepalived-rhel9",
				}),
			},
			want: "something/keepalived-rhel9",
		},
		{
			name: "unsupported_type",
			purl: packageurl.PackageURL{
				Type:      packageurl.TypeApk,
				Namespace: "",
				Name:      "nice APK",
				Version:   "v1.1.1",
				Qualifiers: packageurl.QualifiersFromMap(map[string]string{
					"arch": "ppc64le",
				}),
			},
			expectedErr: true,
		},
		{
			name: "repository_url_without_arch",
			purl: packageurl.PackageURL{
				Type:      packageurl.TypeOCI,
				Namespace: "",
				Name:      "rhacs-scanner-rhel8",
				Version:   "",
				Qualifiers: packageurl.QualifiersFromMap(map[string]string{
					"repository_url": "repository_url=registry.redhat.io/advanced-cluster-security/rhacs-scanner-rhel8",
				}),
			},
			want: "advanced-cluster-security/rhacs-scanner-rhel8",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			v, err := extractPackageName(tc.purl)
			if !errors.Is(err, nil) && !tc.expectedErr {
				t.Fatalf("expected no err but got %v", err)
			}
			if errors.Is(err, nil) && tc.expectedErr {
				t.Fatal("expected err but got none")
			}
			if v != tc.want {
				t.Fatalf("expected name %v but got %v", tc.want, v)
			}
		})
	}
}
