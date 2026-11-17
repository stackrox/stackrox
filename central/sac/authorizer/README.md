# SAC inventory

The built-in scope checker evaluates access scopes using cluster and namespace
IDs, names, and labels. Each request has its own effective access scope trees;
requests share an immutable inventory snapshot for five seconds after a successful
database refresh. Ordinary cluster and namespace store reads go to PostgreSQL.

The inventory is loaded with an unrestricted SAC context because it is the input
to authorization, while retaining the requesting context's cancellation and
deadline. Only one refresh runs at a time. Clusters and namespaces are published
together after both reads succeed; an expired snapshot is never used as a fallback
when refreshing fails. Changes to inventory labels and names can take five seconds
plus the duration of a refresh to affect new requests. Role permissions and access
scope rules are not cached here.

This snapshot keeps the existing in-memory scope evaluator usable while the
whole-store caches are removed. A subsequent change will evaluate scope selectors
using SQL and the label GIN indexes. Name and ID scopes already have the cluster
primary key/name indexes and the namespace `(name, clusterid)` index available.
