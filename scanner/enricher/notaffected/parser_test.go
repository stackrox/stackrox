package notaffected

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/klauspost/compress/snappy"
	"github.com/package-url/packageurl-go"
	"github.com/quay/claircore/libvuln/driver"
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

// Test_ParseEnrichment_SingleProduct tests chunking behavior for individual products with various chunk sizes.
func Test_ParseEnrichment_SingleProduct(t *testing.T) {
	for _, tc := range []struct {
		name             string
		maxCVEsPerRecord int
		productName      string
		numCVEs          int
	}{
		{
			name:             "single CVE per record",
			maxCVEsPerRecord: 1,
			productName:      "single-product",
			numCVEs:          5,
		},
		{
			name:             "multiple CVEs per record",
			maxCVEsPerRecord: 3,
			productName:      "multi-cve-product",
			numCVEs:          7, // should create 3 records (3+3+1)
		},
		{
			name:             "large chunk size",
			maxCVEsPerRecord: 10,
			productName:      "large-chunk-product",
			numCVEs:          5, // should create 1 record
		},
		{
			name:             "product with special chars",
			maxCVEsPerRecord: 2,
			productName:      "product-with-dashes/and-slashes",
			numCVEs:          4, // should create 2 records (2+2)
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Create enricher.
			enricher := &Enricher{maxCVEsPerRecord: tc.maxCVEsPerRecord}
			// Create CSAF structure.
			testData := csafAdvisory(map[string]int{tc.productName: tc.numCVEs})
			// Marshall into JSON.
			var buf bytes.Buffer
			sw := snappy.NewBufferedWriter(&buf)
			_, err := sw.Write([]byte(testData))
			if err != nil {
				t.Fatalf("failed to write test data: %v", err)
			}
			if err = sw.Close(); err != nil {
				t.Fatalf("failed to close test data writer: %v", err)
			}
			// Run.
			records, err := enricher.ParseEnrichment(context.Background(), io.NopCloser(&buf))
			if err != nil {
				t.Fatalf("failed to parse enrichment: %v", err)
			}

			// Check expected number of chunks.
			expectedChunks := (tc.numCVEs + tc.maxCVEsPerRecord - 1) / tc.maxCVEsPerRecord
			if len(records) != expectedChunks {
				t.Errorf("expected %d chunks, got %d", expectedChunks, len(records))
			}
			// Check product and chunk content.
			uniqueProducts := make(map[string]bool)
			totalCVEs := 0
			for i, record := range records {
				// Unmarshal CVEs from enrichment data.
				var cves []string
				if err := json.Unmarshal(record.Enrichment, &cves); err != nil {
					t.Fatalf("failed to unmarshal enrichment: %v", err)
				}
				// Check tags.
				if len(record.Tags) < 2 {
					t.Errorf("record missing expected tags: %v", record.Tags)
					continue
				}
				// Check product name.
				productName := record.Tags[0]
				if productName != tc.productName {
					t.Errorf("expected product name %s, got %s", tc.productName, productName)
				}
				uniqueProducts[productName] = true
				// Chunk chunk tag.
				chunkTag := record.Tags[1]
				expectedTag := tc.productName + ":" + strconv.Itoa(i)
				if chunkTag != expectedTag {
					t.Errorf("chunk %d: expected tag %s, got %s", i, expectedTag, chunkTag)
				}
				// Check chunk size, ensure last one is handled differently (it should have remaining CVEs).
				if i == len(records)-1 {
					expectedLastChunk := tc.numCVEs % tc.maxCVEsPerRecord
					if expectedLastChunk == 0 {
						expectedLastChunk = tc.maxCVEsPerRecord
					}
					if len(cves) != expectedLastChunk {
						t.Errorf("last chunk: expected %d CVEs, got %d", expectedLastChunk, len(cves))
					}
				} else {
					if len(cves) != tc.maxCVEsPerRecord {
						t.Errorf("chunk %d: expected %d CVEs, got %d", i, tc.maxCVEsPerRecord, len(cves))
					}
				}
				totalCVEs += len(cves)
			}
			// Validate total CVE count
			if totalCVEs != tc.numCVEs {
				t.Errorf("total CVEs mismatch: expected %d, got %d", tc.numCVEs, totalCVEs)
			}
			t.Logf("Processed %d records for product %s with %d total CVEs", len(records), tc.productName, totalCVEs)
		})
	}
}

// Test_ParseEnrichment_MultipleProducts tests chunking behavior when multiple products are processed together.
func Test_ParseEnrichment_MultipleProducts(t *testing.T) {
	enricher := &Enricher{maxCVEsPerRecord: 2}
	testData := csafAdvisory(map[string]int{
		"product-a": 5, // Should create 3 chunks: [2, 2, 1]
		"product-b": 3, // Should create 2 chunks: [2, 1]
		"product-c": 2, // Should create 1 chunk: [2]
	})
	var buf bytes.Buffer
	sw := snappy.NewBufferedWriter(&buf)
	_, err := sw.Write([]byte(testData))
	if err != nil {
		t.Fatalf("failed to write test data: %v", err)
	}
	if err = sw.Close(); err != nil {
		t.Fatalf("failed to close snappy writer: %v", err)
	}
	records, err := enricher.ParseEnrichment(context.Background(), io.NopCloser(&buf))
	if err != nil {
		t.Fatalf("failed to parse enrichment: %v", err)
	}
	productRecords := make(map[string][]driver.EnrichmentRecord)
	for _, record := range records {
		if len(record.Tags) > 0 {
			productName := strings.Split(record.Tags[1], ":")[0]
			productRecords[productName] = append(productRecords[productName], record)
		}
	}
	expectedChunks := map[string]int{
		"product-a": 3, // 5 CVEs with chunk size 2 = 3 chunks
		"product-b": 2, // 3 CVEs with chunk size 2 = 2 chunks
		"product-c": 1, // 2 CVEs with chunk size 2 = 1 chunk
	}
	for product, expectedCount := range expectedChunks {
		if got := len(productRecords[product]); got != expectedCount {
			t.Errorf("product %s: expected %d chunks, got %d", product, expectedCount, got)
		}
		// Validate chunk tags are correct.
		for i, record := range productRecords[product] {
			expectedTag := product + ":" + strconv.Itoa(i)
			if len(record.Tags) < 2 || record.Tags[1] != expectedTag {
				t.Errorf("product %s chunk %d: expected tag %s, got %v", product, i, expectedTag, record.Tags)
			}
		}
	}
}

func Test_record_EnrichmentRecord(t *testing.T) {
	rec := record{
		prod:  "test-product",
		chunk: 0,
		cves:  []string{"CVE-2024-1", "CVE-2024-2"},
	}

	enrichmentRecord, err := rec.EnrichmentRecord()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if enrichmentRecord == nil {
		t.Fatalf("expected non-nil enrichment record")
	}

	if len(enrichmentRecord.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(enrichmentRecord.Tags))
	}

	if enrichmentRecord.Tags[0] != "test-product" {
		t.Errorf("expected first tag to be 'test-product', got %s", enrichmentRecord.Tags[0])
	}

	if enrichmentRecord.Tags[1] != "test-product:0" {
		t.Errorf("expected second tag to be 'test-product:0', got %s", enrichmentRecord.Tags[1])
	}
}

func Test_parseRecords(t *testing.T) {
	t.Run("invalid json", func(t *testing.T) {
		enricher := &Enricher{maxCVEsPerRecord: 10}
		// Create invalid JSON that will definitely cause CSAF parsing to fail.
		invalidCSAF := "this is not json!\n"
		var buf bytes.Buffer
		sw := snappy.NewBufferedWriter(&buf)
		_, err := sw.Write([]byte(invalidCSAF))
		if err != nil {
			t.Fatalf("failed to write test data: %v", err)
		}
		if err = sw.Close(); err != nil {
			t.Fatalf("failed to close snappy writer: %v", err)
		}

		errorFound := false
		for record, err := range enricher.parseRecords(context.Background(), io.NopCloser(&buf)) {
			if err != nil {
				errorFound = true
				if record != nil {
					t.Error("expected nil record when error occurs")
				}
				if !strings.Contains(err.Error(), "error parsing CSAF") {
					t.Errorf("expected CSAF parsing error, got: %v", err)
				}
				break
			}
			if record != nil {
				t.Error("expected no records from invalid CSAF data")
			}
		}

		if !errorFound {
			t.Error("expected CSAF parsing error, but got none")
		}
	})
	t.Run("iterator terminates early", func(t *testing.T) {
		enricher := &Enricher{maxCVEsPerRecord: 1}
		// Create test data with multiple CVEs.
		testData := csafAdvisory(map[string]int{"test-product": 5})
		var buf bytes.Buffer
		sw := snappy.NewBufferedWriter(&buf)
		_, err := sw.Write([]byte(testData))
		if err != nil {
			t.Fatalf("failed to write test data: %v", err)
		}
		if err = sw.Close(); err != nil {
			t.Fatalf("failed to close snappy writer: %v", err)
		}
		// Parse but terminate early after first record.
		recordCount := 0
		for record, err := range enricher.parseRecords(context.Background(), io.NopCloser(&buf)) {
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if record == nil {
				t.Error("unexpected nil record")
			}
			recordCount++
			if recordCount == 1 {
				break // Early termination
			}
		}
		if recordCount != 1 {
			t.Errorf("expected to process exactly 1 record, got %d", recordCount)
		}
	})
}

// csafAdvisory creates test CSAF document with specified number of CVEs for multiple products.
func csafAdvisory(productCVECounts map[string]int) string {
	relationships, branches, vulnerabilities := csafComponents(productCVECounts)

	csafData := map[string]any{
		"document":        csafDocument(),
		"product_tree":    csafProductTree(relationships, branches),
		"vulnerabilities": vulnerabilities,
	}

	jsonData, _ := json.Marshal(csafData)
	return string(jsonData) + "\n"
}

// csafDocument creates the standard document section for test CSAF data
func csafDocument() map[string]any {
	doc := map[string]any{
		"category": "csaf_vex",
		"title":    "Test VEX",
		"tracking": map[string]any{
			"status": "final",
		},
		"references": []map[string]any{
			{
				"category": "self",
				"url":      "https://example.com/test.json",
			},
		},
	}
	validateCSAFDocument(doc)
	return doc
}

// validateCSAFDocument weak validation of the CSAF structure.
func validateCSAFDocument(doc map[string]any) {
	// Required top-level fields
	requiredFields := []string{"category", "title", "tracking", "references"}
	for _, field := range requiredFields {
		if _, exists := doc[field]; !exists {
			panic(fmt.Sprintf("csafDocument() missing required field: %s", field))
		}
	}

	// Validate category is correct
	if doc["category"] != "csaf_vex" {
		panic(fmt.Sprintf("csafDocument() invalid category: expected 'csaf_vex', got %v", doc["category"]))
	}

	// Validate tracking has status
	tracking, ok := doc["tracking"].(map[string]any)
	if !ok {
		panic("csafDocument() tracking field is not a map")
	}
	if _, exists := tracking["status"]; !exists {
		panic("csafDocument() tracking missing status field")
	}

	// Validate references is an array
	references, ok := doc["references"].([]map[string]any)
	if !ok {
		panic("csafDocument() references field is not an array of maps")
	}

	// Validate self reference exists
	hasSelfRef := false
	for _, ref := range references {
		if category, exists := ref["category"]; exists && category == "self" {
			if _, hasURL := ref["url"]; hasURL {
				hasSelfRef = true
				break
			}
		}
	}
	if !hasSelfRef {
		panic("csafDocument() missing self reference with URL")
	}
}

// csafProductTree creates a product tree.
func csafProductTree(relationships, branches []map[string]any) map[string]any {
	return map[string]any{
		"relationships": relationships,
		"branches":      branches,
	}
}

// csafComponents creates relationships, branches, and vulnerabilities for the given products.
func csafComponents(productCVECounts map[string]int) ([]map[string]any, []map[string]any, []map[string]any) {
	var relationships []map[string]any
	var branches []map[string]any
	var vulnerabilities []map[string]any

	cveCounter := 1000
	for product, cveCount := range productCVECounts {
		rel := map[string]any{
			"category": "default_component_of",
			"full_product_name": map[string]any{
				"name":       product + " as a component of Test Repository",
				"product_id": "test-repo:" + product,
			},
			"product_reference":            product,
			"relates_to_product_reference": "test-repo",
		}
		relationships = append(relationships, rel)
		branch := map[string]any{
			"name": product,
			"product": map[string]any{
				"name":       product,
				"product_id": product,
				"product_identification_helper": map[string]any{
					"purl": "pkg:oci/" + product,
				},
			},
		}
		branches = append(branches, branch)
		for i := 0; i < cveCount; i++ {
			vuln := map[string]any{
				"cve": cveName(cveCounter),
				"product_status": map[string]any{
					"known_not_affected": []string{"test-repo:" + product},
				},
			}
			vulnerabilities = append(vulnerabilities, vuln)
			cveCounter++
		}
	}
	return relationships, branches, vulnerabilities
}

// cveName creates a CVE name from a number.
func cveName(counter int) string {
	return "CVE-2024-" +
		string(rune('0'+(counter/1000))) +
		string(rune('0'+((counter/100)%10))) +
		string(rune('0'+((counter/10)%10))) +
		string(rune('0'+(counter%10)))
}
