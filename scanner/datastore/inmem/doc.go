// Package inmem implements an in-memory, column-native vulnerability datastore
// for the Scanner V4 matcher.
//
// Each vulnerability updater's stream is compiled on import into an immutable
// columnar fragment (struct-of-arrays columns + flat CSR package-name postings
// on the Go heap, with cold display fields and enrichment payloads
// zstd-compressed into read-only memory-mapped files). Published fragment sets
// are swapped atomically and released deterministically via reference counting
// once in-flight queries finish.
//
// The package also provides an ephemeral in-process MatcherMetadataStore (so
// restarts always re-import the vulnerability bundle into memory) and an
// in-process LockSource (so each matcher replica independently loads into its
// own address space).
package inmem
