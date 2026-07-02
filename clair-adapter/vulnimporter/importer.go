package vulnimporter

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/klauspost/compress/zstd"
	"github.com/quay/claircore"
	"github.com/quay/claircore/datastore"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/stackrox/rox/scanner/updater/jsonblob"
)

// Importer imports StackRox vulnerability bundles directly into Clair's database.
type Importer struct {
	store         datastore.MatcherStore
	metadataStore MetadataStore
}

// MetadataStore tracks vulnerability update timestamps.
type MetadataStore interface {
	SetLastVulnerabilityUpdate(ctx context.Context, bundle string, lastUpdate time.Time) error
}

// NewImporter creates an importer that writes to the given Clair matcher store.
// metadataStore is optional; if provided, it records import timestamps per bundle.
func NewImporter(store datastore.MatcherStore, metadataStore MetadataStore) *Importer {
	return &Importer{store: store, metadataStore: metadataStore}
}

// ImportFromZip streams vulnerability bundles from a ZIP file on disk into Clair's database.
// Each entry in the ZIP is a zstd-compressed JSONL file that is streamed through
// decompression and parsing without buffering the full contents in memory.
func (imp *Importer) ImportFromZip(ctx context.Context, zipPath string) error {
	f, err := os.Open(zipPath)
	if err != nil {
		return fmt.Errorf("opening zip: %w", err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat zip: %w", err)
	}

	zr, err := zip.NewReader(f, stat.Size())
	if err != nil {
		return fmt.Errorf("reading zip: %w", err)
	}

	for _, zf := range zr.File {
		if zf.FileInfo().IsDir() {
			continue
		}
		if err := imp.importZipEntry(ctx, zf); err != nil {
			slog.ErrorContext(ctx, "failed to import bundle", "bundle", zf.Name, "error", err)
			continue
		}
	}
	return nil
}

func (imp *Importer) importZipEntry(ctx context.Context, zf *zip.File) error {
	rc, err := zf.Open()
	if err != nil {
		return fmt.Errorf("opening entry %s: %w", zf.Name, err)
	}
	defer rc.Close()

	dec, err := zstd.NewReader(rc)
	if err != nil {
		return fmt.Errorf("creating zstd reader for %s: %w", zf.Name, err)
	}
	defer dec.Close()

	return imp.importFromReader(ctx, dec, zf.Name)
}

// importFromReader is the core import logic, closely following
// scanner/matcher/updater/vuln/updater.go:Import() (lines 209-271).
func (imp *Importer) importFromReader(ctx context.Context, r io.Reader, bundleName string) (err error) {
	iter, iterErr := jsonblob.Iterate(r)

	iter(func(op *driver.UpdateOperation, it jsonblob.RecordIter) bool {
		// Check fingerprint to skip unchanged data.
		var ops map[string][]driver.UpdateOperation
		ops, err = imp.store.GetUpdateOperations(ctx, op.Kind, op.Updater)
		if err != nil {
			return false
		}
		for _, o := range ops[op.Updater] {
			if o.Fingerprint == op.Fingerprint {
				slog.InfoContext(ctx, "fingerprint match, skipping", "updater", op.Updater)
				return true
			}
		}

		slog.InfoContext(ctx, "importing update", "updater", op.Updater, "kind", string(op.Kind), "bundle", bundleName)
		var ref uuid.UUID
		count := 0

		switch op.Kind {
		case driver.VulnerabilityKind:
			ref, err = imp.store.UpdateVulnerabilitiesIter(ctx, op.Updater, op.Fingerprint, func(yield func(*claircore.Vulnerability, error) bool) {
				it(func(v *claircore.Vulnerability, _ *driver.EnrichmentRecord) bool {
					count++
					return yield(v, nil)
				})
				if err := iterErr(); err != nil {
					yield(nil, err)
				}
			})
		case driver.EnrichmentKind:
			ref, err = imp.store.UpdateEnrichmentsIter(ctx, op.Updater, op.Fingerprint, func(yield func(*driver.EnrichmentRecord, error) bool) {
				it(func(_ *claircore.Vulnerability, e *driver.EnrichmentRecord) bool {
					count++
					return yield(e, nil)
				})
				if err := iterErr(); err != nil {
					yield(nil, err)
				}
			})
		default:
			slog.WarnContext(ctx, "unknown kind, skipping", "kind", string(op.Kind))
		}

		if err != nil {
			err = fmt.Errorf("updating %s: %w", op.Kind, err)
			return false
		}

		slog.InfoContext(ctx, "update imported",
			"updater", op.Updater,
			"kind", string(op.Kind),
			"ref", ref.String(),
			"count", count,
			"bundle", bundleName,
		)

		// Record import timestamp for this bundle.
		if imp.metadataStore != nil {
			if storeErr := imp.metadataStore.SetLastVulnerabilityUpdate(ctx, op.Updater, time.Now()); storeErr != nil {
				slog.WarnContext(ctx, "failed to record import timestamp", "updater", op.Updater, "error", storeErr)
			}
		}

		return true
	})

	if err := iterErr(); err != nil {
		return fmt.Errorf("iterating bundle %s: %w", bundleName, err)
	}
	return err
}
