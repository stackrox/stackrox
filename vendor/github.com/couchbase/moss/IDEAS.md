todo / future ideas
-------------------

* plugin / extension API's for adding "side data structures"
  like perfectly balance b-trees to speed lookups
* or, postings lists and columnar re-layouts of data
* reductions for LSM storage
  (https://docs.google.com/document/d/1X9JtIud9an23d4VTxWLpIe4HxgD4NwAFprKQOQX6aDU/edit#heading=h.jj90qi7qbon1)
* more stats
* performance optimizations for handling time-series data,
  where top-level should be able to binary-search through
  non-overlapping, ordered segments, if each segments knows their start/end keys.
* hard crash testing (a'la sqlite or foundationdb)
* related, use fake file interface implementation to test file corruptions
* checksums
* moss as general purpose k/v store...
  * API for synchronous storage
  * non-batch API
* benchmarks against other KV stores
* incremental compaction, as opposed to the existing full compaction
  * hole punching / punch-line algorithm?
  * plasma inspired multiple-files approach for logical "hole punching"
  * block reuse algorithm?
* more concurrent writer goroutines to utilize more I/O bandwidth
* faster Get()'s by explicitly caching top-level binary-search positions
* optimization where each segment tracks min/max key (skip binary search if outside of range)
* optimization where segmentStack knows if min/max keys of segments are non-overlapping and linear
  * support binary searching of segmentStack
  * as an optimization for time-series patterns
* compression (key-prefix?)
* callback API so apps can hook into compaction (e.g., for TTL expirations)
* C-based version of moss?
  * might be named "mossc" (pronounced like "mossy" or "mosque")?
    or, perhaps cmoss ("sea moss" / "CMOS")?
* Optimizations using posix_fadvise()
* Optimizations using sync_file_range()
  * http://stackoverflow.com/questions/3755765/what-posix-fadvise-args-for-sequential-file-write
  * http://yoshinorimatsunobu.blogspot.com/2014/03/how-syncfilerange-really-works.html

incremental compaction
======================

Some handwave thoughts about incremental compaction...

The mossStore append-only file approach is simple, robust, allows for
fast recovery, allows for partial rollbacks, and is relatively
performant for writes.

Its main downsides are...

* A: the mossStore file (data-*.moss) continually grows.
* B: write amplification.
* C: doesn't support concurrent mutations, but currently has just a
  single persister that performs either file mutation appends or
  full file compaction.

On Issue C, an application can shard data across multiple mossStore
instances to try to achieve higher I/O concurrency.

On Issue A, to avoid a forever growing file size, a full compaction
can be performed, which copies any live data to a brand new file.
However, during a full compaction, incoming mutations are not
persisted and are instead queued, and worst case up to 2x the disk
space might be used to perform a full compaction.

Incremental compaction might help solve the issue (A) of file growth
and stalled mutation persistence while also perhaps helping with write
amplication (B) and with concurrency (C).

mossStore tracks a stack of immutable segments, where each segment is
a sorted array of key-val entries.  So, it's O(1) to retrieve the
smallest key and largest key of each segment.  We'll represent the
smallest and largest keys of a segment like "[smallest, largest]".

A stack of 5 segments might look like...

  Diagram: 1

    Level | Key Range
        4 | [B, C] <-- most recent segment.
        3 | [D, I]
        2 | [F, H]
        1 | [E, G]
        0 | [A, J] <-- oldest segment.

Switching to a representation where the key ranges take horizontal
space, the range overlaps amongst those 5 segments becomes more
apparent...

  Diagram: 2

    Level | Key Range
        4 |   [B-C]               <-- most recent segment.
        3 |       [D---------I]
        2 |           [F---H]
        1 |         [E---G]
        0 | [A-----------------J] <-- oldest segment.

Next, we can flatten the diagrammatic representation into a single row
(e.g., sorted by key), and also incorporate the level information next
to each key...

  Diagram: 3

    [A0   [B4   C4]   [D3   [E1   [F2   G1]   H2]   I3]   J0]

For lisp folks, that might look like a bunch of nested parens.

Next, we can calculate the depth (or nesting level) of sub-ranges
between keys by increasing a running depth counter when we see a '['
and decreasing the depth counter when we see a ']'.

  Diagram: 4

             [A0   [B4   C4]   [D3   [E1   [F2   G1]   H2]   I3]   J0]
    depth: 0     1     2     1     2     3     4     3     2     1     0

We can easily find the sub-range with the largest depth, in this case,
F to G which has depth 4.  That sub-range makes a promising candidate
to incrementally compact, based on the theory that higher depth not
only slows down reads more, but also has the most opportunity for
compaction win (higher depth likely means more potential for
de-duplications of older mutations and removals of deletion
tombstones).

The compaction of range F to G means we'd have to split any
intersecting ranges (like range A0 to J0) into potentially 2 smaller
ranges (sub-range A to E and sub-range H to J), such as...

  Diagram: 5

    Level | Key Range
        7 |           [F-G]       <-- most recent segment.
        6 |   [B-C]
        5 |               [H-I]
        4 |       [D-E]
        3 |               [H]
        2 |         [E]
        1 |               [H---J]
        0 | [A-------E]           <-- oldest segment.

As you can see, [F-G] indeed got shorter depth, but the splitting
introduced even more levels to the left and right of [F-G].

So, as a next step, we need to consider heuristics in the algorithm
that might greedily expand the incremental compaction range, so that
instead of incrementally compacting just the range of F to G, perhaps
the incremental compaction should also take care of (for example) E
and H at the same time, so we end up instead with something like...

  Diagram: 6

    Level | Key Range
        5 |         [E-----H]
        4 |   [B-C]
        3 |                 [I]
        2 |       [D]
        1 |                 [I-J]
        0 | [A-----D]

In addition, other sub-ranges that don't overlap with E to H might be
also concurrently, incrementally compacted.  For example, B to C...

  Diagram: 7

    Level | Key Range
        6 |   [B-C]
        5 |         [E-----H]
        4 |                 [I]
        3 |       [D]
        2 |                 [I-J]
        1 |       [D]
        0 | [A]

Next, imagine that sub-ranges D and I-to-J are concurrently,
incrementally compacted, leaving us with...

  Diagram: 8

    Level | Key Range
        4 |                 [I-J]
        3 |       [D]
        2 |   [B-C]
        1 |         [E-----H]
        0 | [A]

In this case, some "adjacent range merger" should notice that [B-C]
and [D] are adjacent and can be trivially merged, leaving us with....

  Diagram: 9

    Level | Key Range
        3 |   [B---D]
        2 |                 [I-J]
        1 |         [E-----H]
        0 | [A]
