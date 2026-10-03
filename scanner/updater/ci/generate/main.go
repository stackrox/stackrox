// Command generate builds the offline Scanner E2E vulnerability bundle.
package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/quay/claircore"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/scanner/updater/jsonblob"
)

const defaultOutput = "scanner/image/scanner/bundles/ci-minimal/vulnerabilities.zip"

var namespaceOID = uuid.FromStringOrPanic("6ba7b812-9dad-11d1-80b4-00c04fd430c8")

// Fixtures are explicit native records. Expectations live separately in the
// Scanner and backend tests; this command never reads tests or a source bundle.
type operation struct {
	Member, Updater string
	Vulnerabilities []*claircore.Vulnerability
	Enrichments     []enrichmentFixture
}

type enrichmentFixture struct {
	Tags    []string
	Payload any
}

type entry struct {
	Updater     string
	Fingerprint driver.Fingerprint
	Date        time.Time
	Ref         uuid.UUID
	Kind        driver.UpdateKind
	Vuln        *claircore.Vulnerability `json:",omitempty"`
	Enrichment  *driver.EnrichmentRecord `json:",omitempty"`
	member      string
}

func main() {
	output := flag.String("output", defaultOutput, "destination ZIP (run from repository root)")
	check := flag.Bool("check", false, "compare without replacing the destination")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "unexpected positional arguments")
		os.Exit(2)
	}
	if err := generate(*output, *check, fixtures()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func entries(fixtures []operation) ([]entry, error) {
	groups := make(map[string][]entry)
	for _, op := range fixtures {
		if len(op.Vulnerabilities) == 0 && len(op.Enrichments) == 0 {
			return nil, fmt.Errorf("empty fixture operation %q/%q", op.Member, op.Updater)
		}
		if filepath.Base(op.Member) != op.Member || !strings.HasSuffix(op.Member, ".json.zst") || op.Updater == "" {
			return nil, fmt.Errorf("invalid member/updater %q/%q", op.Member, op.Updater)
		}
		add := func(e entry) {
			e.member, e.Updater = op.Member, op.Updater
			key := op.Member + "\x00" + op.Updater + "\x00" + string(e.Kind)
			groups[key] = append(groups[key], e)
		}
		for _, v := range op.Vulnerabilities {
			if err := validateVulnerability(v); err != nil {
				return nil, fmt.Errorf("%s: %w", op.Member, err)
			}
			add(entry{Kind: driver.VulnerabilityKind, Vuln: v})
		}
		for _, e := range op.Enrichments {
			payload, err := json.Marshal(e.Payload)
			if err != nil {
				return nil, fmt.Errorf("%s: marshal enrichment: %w", op.Member, err)
			}
			record := &driver.EnrichmentRecord{Tags: append([]string(nil), e.Tags...), Enrichment: payload}
			slices.Sort(record.Tags)
			if err := validateEnrichment(record); err != nil {
				return nil, err
			}
			add(entry{Kind: driver.EnrichmentKind, Enrichment: record})
		}
	}
	if len(groups) == 0 {
		return nil, errors.New("no fixture records")
	}
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	var out []entry
	for _, key := range keys {
		group := groups[key]
		// Canonical JSON uses typed fields and sorted map keys. The zero metadata
		// participates in neither importer cache identity nor fixture ordering.
		encoded := make(map[string]entry, len(group))
		var records []string
		for _, e := range group {
			b, err := json.Marshal(e)
			if err != nil {
				return nil, err
			}
			s := string(b)
			if _, exists := encoded[s]; !exists {
				records = append(records, s)
			}
			encoded[s] = e
		}
		slices.Sort(records)
		hash := sha256.New()
		// Version the encoding, and scope to member, updater and operation kind.
		fmt.Fprintf(hash, "scanner-ci-fixture-v1\x00%s\x00", key)
		for _, record := range records {
			fmt.Fprintln(hash, record)
		}
		digest := hash.Sum(nil)
		fingerprint := driver.Fingerprint(fmt.Sprintf("sha256:%x", digest))
		ref := uuid.NewV5(namespaceOID, string(digest))
		for _, record := range records {
			e := encoded[record]
			e.Date = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
			e.Fingerprint, e.Ref = fingerprint, ref
			out = append(out, e)
		}
	}
	return out, nil
}

func validateVulnerability(v *claircore.Vulnerability) error {
	if v == nil || v.Name == "" || v.Description == "" || v.Package == nil || v.Package.Name == "" {
		return errors.New("vulnerability requires name, description and package")
	}
	if v.Dist == nil && (v.Repo == nil || v.Repo.Name == "") {
		return fmt.Errorf("%s: missing distribution/repository", v.Name)
	}
	if v.Dist != nil && (v.Dist.DID == "" || v.Dist.VersionID == "") {
		return fmt.Errorf("%s: incomplete distribution", v.Name)
	}
	if v.Range != nil && v.Range.Lower.Compare(&v.Range.Upper) > 0 {
		return fmt.Errorf("%s: reversed version range", v.Name)
	}
	return nil
}

func validateEnrichment(e *driver.EnrichmentRecord) error {
	if e == nil || len(e.Tags) == 0 || !json.Valid(e.Enrichment) || bytes.Equal(e.Enrichment, []byte("null")) {
		return errors.New("enrichment requires tags and JSON payload")
	}
	if slices.Contains(e.Tags, "") {
		return errors.New("empty enrichment tag")
	}
	var identity struct{ ID, Name string }
	if err := json.Unmarshal(e.Enrichment, &identity); err != nil {
		return fmt.Errorf("invalid enrichment object: %w", err)
	}
	id := identity.ID
	if id == "" {
		id = identity.Name
	}
	if id == "" || !slices.Contains(e.Tags, id) {
		return errors.New("enrichment identity must be present in its tags")
	}
	return nil
}

func writeArchive(w io.Writer, records []entry) error {
	zw := zip.NewWriter(w)
	for start := 0; start < len(records); {
		end := start + 1
		for end < len(records) && records[end].member == records[start].member {
			end++
		}
		header := &zip.FileHeader{Name: records[start].member, Method: zip.Store}
		header.SetModTime(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC))
		header.SetMode(0644)
		member, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		zs, err := zstd.NewWriter(member, zstd.WithEncoderConcurrency(1), zstd.WithEncoderLevel(zstd.SpeedBestCompression))
		if err != nil {
			return err
		}
		enc := json.NewEncoder(zs)
		for _, record := range records[start:end] {
			if err := enc.Encode(record); err != nil {
				_ = zs.Close()
				return err
			}
		}
		if err := zs.Close(); err != nil {
			return err
		}
		start = end
	}
	return zw.Close()
}

// validateArchive goes through the exact production reader, not merely JSON
// syntax checking. Also verify operation contiguity and payload completeness.
func validateArchive(path string) error {
	r, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer func() { _ = r.Close() }()
	seen := make(map[string]bool)
	for _, member := range r.File {
		if err := validateMember(member, seen); err != nil {
			return fmt.Errorf("%s: %w", member.Name, err)
		}
	}
	return nil
}

func validateMember(member *zip.File, seen map[string]bool) error {
	rc, err := member.Open()
	if err != nil {
		return err
	}
	defer func() { _ = rc.Close() }()
	zr, err := zstd.NewReader(rc)
	if err != nil {
		return err
	}
	defer zr.Close()
	ops, iterErr := jsonblob.Iterate(zr)
	var validationErr error
	ops(func(op *driver.UpdateOperation, records jsonblob.RecordIter) bool {
		ref := op.Ref.String()
		if seen[ref] || ref == uuid.Nil.String() || op.Updater == "" || op.Fingerprint == "" {
			validationErr = fmt.Errorf("invalid or noncontiguous operation %s", ref)
			return false
		}
		seen[ref] = true
		records(func(v *claircore.Vulnerability, e *driver.EnrichmentRecord) bool {
			switch op.Kind {
			case driver.VulnerabilityKind:
				validationErr = validateVulnerability(v)
			case driver.EnrichmentKind:
				validationErr = validateEnrichment(e)
			default:
				validationErr = fmt.Errorf("unknown operation kind %q", op.Kind)
			}
			return validationErr == nil
		})
		return validationErr == nil
	})
	if validationErr != nil {
		return validationErr
	}
	return iterErr()
}

func generate(output string, check bool, fixtures []operation) error {
	records, err := entries(fixtures)
	if err != nil {
		return err
	}
	// A sibling temporary file makes replacement atomic, including across mounts.
	f, err := os.CreateTemp(filepath.Dir(output), ".ci-bundle-*.zip")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(f.Name()) }()
	writeErr := writeArchive(f, records)
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	if err := validateArchive(f.Name()); err != nil {
		return fmt.Errorf("validate candidate: %w", err)
	}
	if check {
		want, err := os.ReadFile(output)
		if err != nil {
			return err
		}
		got, err := os.ReadFile(f.Name())
		if err != nil {
			return err
		}
		if !bytes.Equal(want, got) {
			return fmt.Errorf("%s differs from fixtures", output)
		}
		return nil
	}
	if err := os.Chmod(f.Name(), 0644); err != nil {
		return err
	}
	return os.Rename(f.Name(), output)
}
