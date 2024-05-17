// Package jsonblob is a fork of ClairCore's jsonblob with minimal changes to
// support individual record iteration.
//
// TODO(ROX-24333): Remove this package once ClairCore supports it, or become
//                  obsolete by other formats.

package jsonblob

import (
	"bufio"
	"encoding/json"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/quay/claircore"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/stackrox/rox/pkg/sync"
)

type iter2[X, Y any] func(yield func(X, Y) bool)

// RecordIter iterates over records of an update operation.
type RecordIter iter2[*claircore.Vulnerability, *driver.EnrichmentRecord]

// OperationIter iterates over operations, offering a nested iterator for records.
type OperationIter iter2[*driver.UpdateOperation, RecordIter]

// Iterate iterates over each record serialized in the [io.Reader] grouping by
// update operations. It returns an OperationIter, which is an iterator over each
// update operation with a nested iterator for the associated vulnerability
// entries, and an error function, to check for iteration errors.
func Iterate(r io.Reader) (OperationIter, func() error) {
	var err error
	var de diskEntry

	d := json.NewDecoder(r)
	err = d.Decode(&de)

	it := func(yield func(*driver.UpdateOperation, RecordIter) bool) {
		for err == nil {
			op := &driver.UpdateOperation{
				Ref:         de.Ref,
				Updater:     de.Updater,
				Fingerprint: de.Fingerprint,
				Date:        de.Date,
				Kind:        de.Kind,
			}
			it := func(yield func(*claircore.Vulnerability, *driver.EnrichmentRecord) bool) {
				var vuln *claircore.Vulnerability
				var en *driver.EnrichmentRecord
				for err == nil && op.Ref == de.Ref {
					vuln, en, err = de.Unmarshal()
					if err != nil || !yield(vuln, en) {
						break
					}
					err = d.Decode(&de)
				}
			}
			if !yield(op, it) {
				break
			}
			for err == nil && op.Ref == de.Ref {
				err = d.Decode(&de)
			}
		}
	}

	errF := func() error {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}

	return it, errF
}

// CommonEntry is an embedded type that's shared between the "normal" [Entry] type
// and the on-disk JSON produced by the [Store.Store] method.
type CommonEntry struct {
	Updater     string
	Fingerprint driver.Fingerprint
	Date        time.Time
}

// diskEntry is a single vulnerability or enrichment. It's made from unpacking an
// Entry's slice and adding an uuid for grouping back into an Entry upon read.
//
// "Vuln" and "Enrichment" are populated from the backing disk immediately
// before being serialized.
type diskEntry struct {
	CommonEntry
	Ref        uuid.UUID
	Vuln       *bufShim `json:",omitempty"`
	Enrichment *bufShim `json:",omitempty"`
	Kind       driver.UpdateKind
}

// Unmarshal parses the JSON-encoded vulnerability or enrichment record encoded
// in the disk entry, based on the update kind.
func (de *diskEntry) Unmarshal() (v *claircore.Vulnerability, e *driver.EnrichmentRecord, err error) {
	switch de.Kind {
	case driver.VulnerabilityKind:
		v = &claircore.Vulnerability{}
		if err = json.Unmarshal(de.Vuln.buf, v); err != nil {
			return
		}
	case driver.EnrichmentKind:
		e = &driver.EnrichmentRecord{}
		err = json.Unmarshal(de.Enrichment.buf, e)
		if err != nil {
			return
		}
	}
	return
}

// bufShim treats every call to [bufShim.MarshalJSON] as a [bufio.Scanner.Scan]
// call.
//
// Note this type is very weird, in that it can only be used for _either_ an
// Unmarshal or a Marshal, not both. Doing both on the same structure will
// silently do the wrong thing.
type bufShim struct {
	s   *bufio.Scanner
	buf []byte
}

func (s *bufShim) MarshalJSON() ([]byte, error) {
	if !s.s.Scan() {
		return nil, s.s.Err()
	}
	return s.s.Bytes(), nil
}

func (s *bufShim) UnmarshalJSON(b []byte) error {
	s.buf = append(s.buf[0:0], b...)
	return nil
}

func (s *bufShim) Close() error {
	putBuf(s.buf)
	return nil
}

var bufPool sync.Pool

func putBuf(b []byte) {
	bufPool.Put(&b)
}
