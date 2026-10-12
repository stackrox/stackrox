# SAC scope lookups

The built-in scope checker evaluates cluster and namespace scope rules with
parameterized PostgreSQL queries. It selects matching IDs and names and builds
the existing effective access scope tree from those results. Labels and serialized
objects stay in the database; no complete inventory or shared inventory cache is
required. Ordinary cluster and namespace store reads also go to PostgreSQL.

Scope metadata is read on first use of an access scope in a request. The resulting
tree or query error is cached by scope ID for the rest of that request and shared
across resources and access modes. Concurrent checks cannot duplicate a lookup.
Permissions remain paired with their role's scope. Built-in unrestricted/deny-all
scopes and global resources do not query cluster or namespace metadata. Queries
retain request cancellation and deadlines; failures never grant access.

The predicates preserve selector AND/OR composition, empty selectors matching
nothing, missing labels matching `NOT_IN`/`NOT_EXISTS`, ID precedence in namespace
rules, and the virtual name labels overriding stored labels. Parent clusters are
resolved by the namespace's recorded cluster name, as in the original evaluator.
The GIN indexes support JSONB containment and key existence; negative selectors
can require database scans and broad scopes still return many matches. Existing
name and ID indexes cover direct scopes.

Authorization tracing additionally reads cluster IDs/names and a namespace count
to preserve the existing inventory totals and human-readable decisions. It does
not load namespace objects. Subsequent requests observe inventory changes without
the former five-second snapshot delay; an already-computed request scope remains
stable for that request.

Matching namespaces are aggregated per recorded parent into one text array. Each
entry contains a fixed-width UUID followed by the namespace name, avoiding two
ordered aggregates and keeping IDs/names paired for arbitrary names. The tree is
built directly from compact groups with namespace maps sized for the result.
Duplicate recorded names retain the smallest matching ID without sorting SQL
results. Debug scope strings are formatted only when tracing is enabled.
