// Package inmem implements an in-memory, column-native vulnerability datastore
// for the Scanner V4 matcher.
//
// Each vulnerability updater's stream is compiled on import into an immutable
// columnar fragment: per-row primitive columns, sparse version ranges, and flat
// CSR package-name postings are written to an 8-byte-aligned read-only mmap
// file (inmem-cols-*.col), while cold display strings and enrichment payloads
// are zstd-compressed into read-only mmap blobs (inmem-display-*.blob,
// inmem-enrich-*.blob). String dictionaries and package-name span maps stay on
// the Go heap so returned claircore.Vulnerability values hold no pointers into
// mmap regions. Published fragment sets are swapped atomically and released
// deterministically via reference counting once in-flight queries drain.
//
// The package also provides an ephemeral in-process MatcherMetadataStore (so
// restarts always re-import the vulnerability bundle into memory) and an
// in-process LockSource (so each matcher replica independently loads into its
// own address space).
package inmem
