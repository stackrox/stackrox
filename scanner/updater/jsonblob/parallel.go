package jsonblob

import (
	"bufio"
	"bytes"
	stdjson "encoding/json"
	"io"
	"time"

	gojson "github.com/goccy/go-json"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/quay/claircore"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/stackrox/rox/pkg/sync"
)

// UnmarshalFunc unmarshals a single JSON record. It exists so the decoder can be
// swapped (stdlib encoding/json vs a faster drop-in) for benchmarking.
type UnmarshalFunc func(data []byte, v any) error

// StdUnmarshal uses the standard library encoding/json decoder.
func StdUnmarshal(data []byte, v any) error { return stdjson.Unmarshal(data, v) }

// GoccyUnmarshal uses github.com/goccy/go-json, a faster drop-in that still
// honours the custom UnmarshalJSON/UnmarshalText methods claircore relies on.
func GoccyUnmarshal(data []byte, v any) error { return gojson.Unmarshal(data, v) }

// parEntry mirrors the on-disk record shape enough to decode a single line.
//
// The bundle is newline-delimited JSON: one diskEntry object per line, with
// records grouped into contiguous runs by update operation (Ref). We rely on
// that here to split records cheaply (by line) and unmarshal them in parallel.
type parEntry struct {
	Vuln       *claircore.Vulnerability `json:"Vuln,omitempty"`
	Enrichment *driver.EnrichmentRecord `json:"Enrichment,omitempty"`
}

// opMeta is the per-operation envelope, decoded once from the first line of each
// operation. The heavy Vuln/Enrichment payload is an unknown field to this struct
// and is skipped by the decoder.
type opMeta struct {
	Updater     string
	Fingerprint driver.Fingerprint
	Date        time.Time
	Ref         uuid.UUID
	Kind        driver.UpdateKind
}

var refKey = []byte(`"Ref":"`)

// extractRef returns the Ref UUID string from a record line without a full JSON
// parse. The on-disk field order places Updater/Fingerprint/Date before Ref and
// the (much larger) Vuln/Enrichment payload after it, and none of the preceding
// string values contain the literal `"Ref":"`, so the first match is the record
// Ref. This keeps operation-boundary detection off the hot path.
func extractRef(line []byte) string {
	i := bytes.Index(line, refKey)
	if i < 0 {
		return ""
	}
	rest := line[i+len(refKey):]
	j := bytes.IndexByte(rest, '"')
	if j < 0 {
		return ""
	}
	return string(rest[:j])
}

// IterateParallel is a drop-in replacement for [Iterate] that fans the expensive
// per-record JSON unmarshal out across `workers` goroutines. Record ordering
// within an operation is not preserved, which is safe: the datastore deduplicates
// vulnerabilities by content hash and associates them as a set, so insertion
// order does not affect the result.
//
// If workers <= 1 it falls back to [Iterate].
func IterateParallel(r io.Reader, workers int, unmarshal UnmarshalFunc) (OperationIter, func() error) {
	if workers <= 1 {
		return Iterate(r)
	}
	if unmarshal == nil {
		unmarshal = StdUnmarshal
	}

	br := bufio.NewReaderSize(r, 1<<20)

	var (
		mu      sync.Mutex
		fatal   error // first error seen, guarded by mu
		pending []byte
	)
	setErr := func(e error) {
		mu.Lock()
		if fatal == nil {
			fatal = e
		}
		mu.Unlock()
	}
	getErr := func() error {
		mu.Lock()
		defer mu.Unlock()
		return fatal
	}

	// readLine returns the next record line (a freshly allocated slice, safe to
	// hand to a worker), or nil at EOF.
	readLine := func() []byte {
		for {
			b, err := br.ReadBytes('\n')
			if len(b) > 0 {
				return b
			}
			if err != nil {
				if !errors.Is(err, io.EOF) {
					setErr(err)
				}
				return nil
			}
		}
	}

	// Prime the first line.
	pending = readLine()

	it := func(yield func(*driver.UpdateOperation, RecordIter) bool) {
		for getErr() == nil {
			first := pending
			pending = nil
			if first == nil {
				return
			}

			var meta opMeta
			if err := unmarshal(first, &meta); err != nil {
				setErr(errors.Wrap(err, "decoding operation envelope"))
				return
			}
			op := &driver.UpdateOperation{
				Ref:         meta.Ref,
				Updater:     meta.Updater,
				Fingerprint: meta.Fingerprint,
				Date:        meta.Date,
				Kind:        meta.Kind,
			}
			curRef := extractRef(first)

			recIter := func(yield2 func(*claircore.Vulnerability, *driver.EnrichmentRecord) bool) {
				// Buffers are deliberately small and bounded. The matcher runs
				// under a tight memory budget (it has OOMed on this workload), and
				// the downstream datastore write is the throughput bottleneck, so
				// letting workers race far ahead would only pile up decoded records
				// in memory for no speedup. Small buffers give enough pipelining to
				// keep the workers fed without an unbounded in-flight backlog.
				lineBuf := workers * 4
				resBuf := workers * 16
				lineCh := make(chan []byte, lineBuf)
				type result struct {
					v *claircore.Vulnerability
					e *driver.EnrichmentRecord
				}
				resCh := make(chan result, resBuf) //nolint:staticcheck // bounded by design

				var wg sync.WaitGroup
				for i := 0; i < workers; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						for lb := range lineCh {
							var pe parEntry
							if err := unmarshal(lb, &pe); err != nil {
								setErr(errors.Wrap(err, "decoding record"))
								continue
							}
							resCh <- result{pe.Vuln, pe.Enrichment}
						}
					}()
				}
				go func() {
					wg.Wait()
					close(resCh)
				}()

				// Feeder: hand this operation's lines to the workers, then stash
				// the first line of the next operation in `pending`.
				go func() {
					defer close(lineCh)
					lineCh <- first
					for getErr() == nil {
						nl := readLine()
						if nl == nil {
							return
						}
						if extractRef(nl) != curRef {
							pending = nl
							return
						}
						lineCh <- nl
					}
				}()

				for res := range resCh {
					if !yield2(res.v, res.e) {
						// Consumer asked to stop; drain remaining results so the
						// worker goroutines can exit.
						for range resCh {
						}
						return
					}
				}
			}

			if !yield(op, recIter) {
				return
			}
		}
	}

	errF := func() error {
		if err := getErr(); err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		return nil
	}

	return it, errF
}
