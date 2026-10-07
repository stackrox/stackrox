// Command generate builds the offline Scanner E2E vulnerability bundle.
package main

// If the generated bundle changes, update its immutable consumer pins as described
// in ../README.md under "Publication and rollback".
//go:generate go run . -output ../../../image/scanner/bundles/ci-minimal/vulnerabilities.zip

import (
	"archive/zip"
	"bytes"
	"context"
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
	storeblob "github.com/quay/claircore/libvuln/jsonblob"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/scanner/updater/bundle"
	"github.com/stackrox/rox/scanner/updater/jsonblob"
)

const defaultOutput = "scanner/image/scanner/bundles/ci-minimal/vulnerabilities.zip"
const defaultBundleRevision = "2026-10-07T00:00:00Z"

var namespaceOID = uuid.FromStringOrPanic("6ba7b812-9dad-11d1-80b4-00c04fd430c8")

func bundleRevision() time.Time {
	revision, err := time.Parse(time.RFC3339, defaultBundleRevision)
	if err != nil {
		panic(err)
	}
	return revision
}

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

// fixtureOperation is an internal grouping, not the serialized bundle format.
type fixtureOperation struct {
	Member, Updater string
	Fingerprint     driver.Fingerprint
	Ref             uuid.UUID
	Kind            driver.UpdateKind
	Vulnerabilities []*claircore.Vulnerability
	Enrichments     []driver.EnrichmentRecord
}

type fixtureRecord struct {
	vuln       *claircore.Vulnerability
	enrichment *driver.EnrichmentRecord
}

func (r fixtureRecord) canonical() ([]byte, error) {
	if r.vuln != nil {
		return json.Marshal(r.vuln)
	}
	return json.Marshal(r.enrichment)
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

func entries(fixtures []operation) ([]fixtureOperation, error) {
	groups := make(map[string][]fixtureRecord)
	for _, op := range fixtures {
		if len(op.Vulnerabilities) == 0 && len(op.Enrichments) == 0 {
			return nil, fmt.Errorf("empty fixture operation %q/%q", op.Member, op.Updater)
		}
		if filepath.Base(op.Member) != op.Member || !strings.HasSuffix(op.Member, ".json.zst") || op.Updater == "" || strings.ContainsRune(op.Updater, 0) {
			return nil, fmt.Errorf("invalid member/updater %q/%q", op.Member, op.Updater)
		}
		add := func(kind driver.UpdateKind, e fixtureRecord) {
			key := op.Member + "\x00" + op.Updater + "\x00" + string(kind)
			groups[key] = append(groups[key], e)
		}
		for _, v := range op.Vulnerabilities {
			if err := validateVulnerability(v); err != nil {
				return nil, fmt.Errorf("%s: %w", op.Member, err)
			}
			add(driver.VulnerabilityKind, fixtureRecord{vuln: v})
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
			add(driver.EnrichmentKind, fixtureRecord{enrichment: record})
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
	var out []fixtureOperation
	for _, key := range keys {
		group := groups[key]
		// Canonical payloads use typed fields and sorted map keys.
		encoded := make(map[string]fixtureRecord, len(group))
		var records []string
		for _, e := range group {
			b, err := e.canonical()
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
		fmt.Fprintf(hash, "scanner-ci-fixture-v2\x00%s\x00", key)
		for _, record := range records {
			fmt.Fprintln(hash, record)
		}
		digest := hash.Sum(nil)
		fingerprint := driver.Fingerprint(fmt.Sprintf("sha256:%x", digest))
		ref := uuid.NewV5(namespaceOID, string(digest))
		parts := strings.Split(key, "\x00")
		op := fixtureOperation{Member: parts[0], Updater: parts[1], Kind: driver.UpdateKind(parts[2]), Fingerprint: fingerprint, Ref: ref}
		for _, record := range records {
			e := encoded[record]
			if e.vuln != nil {
				op.Vulnerabilities = append(op.Vulnerabilities, e.vuln)
			}
			if e.enrichment != nil {
				op.Enrichments = append(op.Enrichments, *e.enrichment)
			}
		}
		out = append(out, op)
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

func writeArchive(w io.Writer, records []fixtureOperation) error {
	return writeArchiveAt(w, records, bundleRevision())
}

func writeArchiveAt(w io.Writer, records []fixtureOperation, revision time.Time) error {
	zw := zip.NewWriter(w)
	for start := 0; start < len(records); {
		end := start + 1
		for end < len(records) && records[end].Member == records[start].Member {
			end++
		}
		header := &zip.FileHeader{Name: records[start].Member, Method: zip.Store}
		header.SetModTime(revision)
		header.SetMode(0644)
		member, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		err = bundle.WriteCompressed(member, func(w io.Writer) error {
			for _, op := range records[start:end] {
				if err := writeOperation(w, op, revision); err != nil {
					return err
				}
			}
			return nil
		}, zstd.WithEncoderConcurrency(1), zstd.WithEncoderLevel(zstd.SpeedBestCompression))
		if err != nil {
			return err
		}
		start = end
	}
	return zw.Close()
}

// A fresh store for each operation avoids the store's map iteration order.
func writeOperation(w io.Writer, op fixtureOperation, revision time.Time) error {
	store, err := storeblob.New()
	if err != nil {
		return err
	}
	switch op.Kind {
	case driver.VulnerabilityKind:
		_, err = store.UpdateVulnerabilities(context.Background(), op.Updater, op.Fingerprint, op.Vulnerabilities)
	case driver.EnrichmentKind:
		_, err = store.UpdateEnrichments(context.Background(), op.Updater, op.Fingerprint, op.Enrichments)
	default:
		return fmt.Errorf("unknown operation kind %q", op.Kind)
	}
	if err != nil {
		return err
	}
	var serialized bytes.Buffer
	if err := store.Store(&serialized); err != nil {
		return err
	}
	return normalizeRecords(w, &serialized, op.Ref, revision)
}

// Preserve the production envelope and payload, normalizing only volatile metadata.
func normalizeRecords(w io.Writer, r io.Reader, ref uuid.UUID, date time.Time) error {
	refJSON, err := json.Marshal(ref)
	if err != nil {
		return err
	}
	dateJSON, err := json.Marshal(date)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(r)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	for {
		var record map[string]json.RawMessage
		if err := dec.Decode(&record); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		for _, field := range []string{"Ref", "Date"} {
			if _, ok := record[field]; !ok {
				return fmt.Errorf("production bundle record missing %s", field)
			}
		}
		record["Ref"], record["Date"] = refJSON, dateJSON
		if err := enc.Encode(record); err != nil {
			return err
		}
	}
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
	return generateAt(output, check, fixtures, bundleRevision())
}

func generateAt(output string, check bool, fixtures []operation, revision time.Time) error {
	revision = revision.UTC().Truncate(time.Second)
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
	writeErr := writeArchiveAt(f, records, revision)
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
	got, err := os.ReadFile(f.Name())
	if err != nil {
		return err
	}
	identical, err := checkExistingBundle(output, got, revision)
	if err != nil {
		return err
	}
	if identical {
		return nil
	}
	if err := os.Chmod(f.Name(), 0644); err != nil {
		return err
	}
	return os.Rename(f.Name(), output)
}

func checkExistingBundle(output string, candidate []byte, revision time.Time) (bool, error) {
	existing, err := os.ReadFile(output)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read existing bundle %s: %w", output, err)
	}
	if bytes.Equal(existing, candidate) {
		return true, nil
	}
	if err := validateArchive(output); err != nil {
		return false, fmt.Errorf("existing bundle %s is invalid; refusing to replace it: %w", output, err)
	}
	archive, err := zip.OpenReader(output)
	if err != nil {
		return false, fmt.Errorf("open existing bundle %s: %w", output, err)
	}
	defer func() { _ = archive.Close() }()
	for _, member := range archive.File {
		if !revision.After(member.Modified) {
			return false, fmt.Errorf("bundle revision %s must be later than existing member %q timestamp %s; bump defaultBundleRevision before replacing the archive", revision.Format(time.RFC3339), member.Name, member.Modified.Format(time.RFC3339))
		}
	}
	return false, nil
}
