moss
----

moss provides a simple, fast, persistable, ordered key-val collection
implementation as a 100% golang library.

moss stands for "memory-oriented sorted segments".

[![Build Status](https://travis-ci.org/couchbase/moss.svg?branch=master)](https://travis-ci.org/couchbase/moss) [![Coverage Status](https://coveralls.io/repos/github/couchbase/moss/badge.svg?branch=master)](https://coveralls.io/github/couchbase/moss?branch=master) [![GoDoc](https://godoc.org/github.com/couchbase/moss?status.svg)](https://godoc.org/github.com/couchbase/moss) [![Go Report Card](https://goreportcard.com/badge/github.com/couchbase/moss)](https://goreportcard.com/report/github.com/couchbase/moss)

Features
========

* ordered key-val collection API
* 100% go implementation
* key range iterators
* snapshots provide for isolated reads
* atomic mutations via a batch API
* merge operations allow for read-compute-write optimizations
  for write-heavy use cases (e.g., updating counters)
* concurrent readers and writers don't block each other
* child collections allow multiple related collections to be atomically grouped
* optional, advanced API's to avoid extra memory copying
* optional lower-level storage implementation, called "mossStore",
  that uses an append-only design for writes and mmap() for reads,
  with configurable compaction policy; see: OpenStoreCollection()
* mossStore supports navigating back through previous commit points in
  read-only fashion, and supports reverting to previous commit points.
* optional persistence hooks to allow write-back caching to a
  lower-level storage implementation that advanced users may wish to
  provide (e.g., you can hook moss up to leveldb, sqlite, etc)
* event callbacks allow the monitoring of asynchronous tasks
* unit tests
* fuzz tests via go-fuzz & smat (github.com/mschoch/smat);
  see README-smat.md
* moss store's diagnostic tool: [mossScope](https://github.com/couchbase/mossScope)

License
=======

Apache 2.0

Example
=======

    import github.com/couchbase/moss

    c, err := moss.NewCollection(moss.CollectionOptions{})
    c.Start()
    defer c.Close()

    batch, err := c.NewBatch(0, 0)
    defer batch.Close()

    batch.Set([]byte("car-0"), []byte("tesla"))
    batch.Set([]byte("car-1"), []byte("honda"))

    err = c.ExecuteBatch(batch, moss.WriteOptions{})

    ss, err := c.Snapshot()
    defer ss.Close()

    ropts := moss.ReadOptions{}

    val0, err := ss.Get([]byte("car-0"), ropts) // val0 == []byte("tesla").
    valX, err := ss.Get([]byte("car-not-there"), ropts) // valX == nil.

    // A Get can also be issued directly against the collection
    val1, err := c.Get([]byte("car-1"), ropts) // val1 == []byte("honda").

For persistence, you can use...

    store, collection, err := moss.OpenStoreCollection(directoryPath,
        moss.StoreOptions{}, moss.StorePersistOptions{})

Design
======

The design is similar to a (much) simplified LSM tree, with a stack of
sorted, immutable key-val arrays or "segments".

To incorporate the next Batch of key-val mutations, the incoming
key-val entries are first sorted into an immutable "segment", which is
then atomically pushed onto the top of the stack of segments.

For readers, a higher segment in the stack will shadow entries of the
same key from lower segments.

Separately, an asynchronous goroutine (the "merger") will continuously
merge N sorted segments to keep stack height low.

In the best case, a remaining, single, large sorted segment will be
efficient in memory usage and efficient for binary search and range
iteration.

Iterations when the stack height is > 1 are implementing using a N-way
heap merge.

In this design, the stack of segments is treated as immutable via a
copy-on-write approach whenever the stack needs to be "modified".  So,
multiple readers and writers won't block each other, and taking a
Snapshot is also a similarly cheap operation by cloning the stack.

See also the DESIGN.md writeup.

Limitations and considerations
==============================

NOTE: Keys in a Batch must be unique.  That is, myBatch.Set("x",
"foo"); myBatch.Set("x", "bar") is not supported.  Applications that
do not naturally meet this requirement might maintain their own
map[key]val data structures to ensure this uniqueness constraint.

Max key length is 2^24 (24 bits used to track key length).

Max val length is 2^28 (28 bits used to track val length).

Metadata overhead for each key-val operation is 16 bytes.

Read performance characterization is roughly O(log N) for key-val
retrieval.

Write performance characterization is roughly O(M log M), where M is
the number of mutations in a batch when invoking ExecuteBatch().

Those performance characterizations, however, don't account for
background, asynchronous processing for the merging of segments and
data structure maintenance.

A background merger task, for example, that is too slow can eventually
stall ingest of new batches.  (See the CollectionOptions settings that
limit segment stack height.)

As another example, one slow reader that holds onto a Snapshot or onto
an Iterator for a long time can hold onto a lot of resources.  Worst
case is the reader's Snapshot or Iterator may delay the reclaimation
of large, old segments, where incoming mutations have obsoleted the
immutable segments that the reader is still holding onto.

Error handling
==============

Please note that the background goroutines of moss may run into
errors, for example during optional persistence operations.  To be
notified of these cases, your application can provide (highly
recommended) an optional CollectionOptions.OnError callback func which
will be invoked by moss.

Logging
=======

Please see the optional CollectionOptions.Log callback func and the
CollectionOptions.Debug flag.

Performance
===========

Please try `go test -bench=.` for some basic performance tests.

Each performance test will emit output that generally looks like...

    ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
    spec: {numItems:1000000 keySize:20 valSize:100 batchSize:100 randomLoad:false noCopyValue:false accesses:[]}
         open || time:     0 (ms) |        0 wop/s |        0 wkb/s |        0 rop/s |        0 rkb/s || cumulative:        0 wop/s |        0 wkb/s |        0 rop/s |        0 rkb/s
         load || time:   840 (ms) |  1190476 wop/s |   139508 wkb/s |        0 rop/s |        0 rkb/s || cumulative:  1190476 wop/s |   139508 wkb/s |        0 rop/s |        0 rkb/s
        drain || time:   609 (ms) |        0 wop/s |        0 wkb/s |        0 rop/s |        0 rkb/s || cumulative:   690131 wop/s |    80874 wkb/s |        0 rop/s |        0 rkb/s
        close || time:     0 (ms) |        0 wop/s |        0 wkb/s |        0 rop/s |        0 rkb/s || cumulative:   690131 wop/s |    80874 wkb/s |        0 rop/s |        0 rkb/s
       reopen || time:     0 (ms) |        0 wop/s |        0 wkb/s |        0 rop/s |        0 rkb/s || cumulative:   690131 wop/s |    80874 wkb/s |        0 rop/s |        0 rkb/s
         iter || time:    81 (ms) |        0 wop/s |        0 wkb/s | 12344456 rop/s |  1446616 rkb/s || cumulative:   690131 wop/s |    80874 wkb/s | 12344456 rop/s |  1446616 rkb/s
        close || time:     2 (ms) |        0 wop/s |        0 wkb/s |        0 rop/s |        0 rkb/s || cumulative:   690131 wop/s |    80874 wkb/s | 12344456 rop/s |  1446616 rkb/s
    total time: 1532 (ms)
    file size: 135 (MB), amplification: 1.133
    BenchmarkStore_numItems1M_keySize20_valSize100_batchSize100-8

There are various phases in each test...

* open - opening a brand new moss storage instance
* load - time to load N sequential keys
* drain - additional time after load for persistence to complete
* close - time to close the moss storage instance
* reopen - time to reopen the moss storage instance (OS/filesystem caches are still warm)
* iter - time to sequentially iterate through key-val items
* access - time to perform various access patterns, like random or sequential reads and writes

The file size measurement is after final compaction, with
amplification as a naive calculation to compare overhead against raw
key-val size.

Contributing changes
====================

Please see the CONTRIBUTING.md document.
