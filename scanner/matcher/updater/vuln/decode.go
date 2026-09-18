package vuln

import (
	"io"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/scanner/updater/jsonblob"
)

var (
	// importWorkers controls how many goroutines decode (JSON-unmarshal) bundle
	// records in parallel during a vulnerability import. 1 (default) keeps the
	// original single-threaded, deterministic decode. Higher values speed up the
	// CPU-bound decode of large feeds at the cost of more transient heap, so tune
	// against the matcher's memory budget.
	importWorkers = env.RegisterIntegerSetting("ROX_SCANNER_V4_IMPORT_WORKERS", 1)

	// importJSONDecoder selects the JSON decoder used for bundle records.
	// "std" (default) uses encoding/json; "goccy" uses github.com/goccy/go-json,
	// a faster drop-in that still honours the custom (Un)marshalers claircore
	// relies on.
	importJSONDecoder = env.RegisterSetting("ROX_SCANNER_V4_JSON_DECODER", env.WithDefault("std"))
)

// iterFunc mirrors the signature of jsonblob.Iterate / IterateParallel.
type iterFunc func(r io.Reader) (jsonblob.OperationIter, func() error)

// newIterateFunc returns the bundle iterator to use, honouring the import-worker
// and decoder settings. With the defaults it is exactly jsonblob.Iterate.
func newIterateFunc() iterFunc {
	decode := jsonblob.StdUnmarshal
	if importJSONDecoder.Setting() == "goccy" {
		decode = jsonblob.GoccyUnmarshal
	}
	workers := importWorkers.IntegerSetting()
	return func(r io.Reader) (jsonblob.OperationIter, func() error) {
		// IterateParallel falls back to the serial Iterate when workers <= 1.
		return jsonblob.IterateParallel(r, workers, decode)
	}
}
