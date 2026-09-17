# Central Worker

Central Worker and Central share the PostgreSQL database. PostgreSQL is the
source of truth for every persisted object; Worker uses database-backed stores
without retained store caches. Central may retain coordinated caches, but
their `LISTEN`/`NOTIFY` stream is only a change hint. Cache observers reread
the affected table (or a full snapshot) before updating derived process-local
state. A listener reconnect or an invalidated cache triggers reconciliation,
so a missed notification does not become permanent state.

## Job ownership

When `ROX_CENTRAL_WORKER_ENABLED=true`, Worker owns bulk pruning. Central runs
only cluster decommissioning/lifecycle cleanup and expired dynamic RBAC
cleanup; this keeps Sensor connection teardown and the internal-token-created
RBAC lifecycle in Central. The two pruning modes serialize through
`dblock.PruningGCLockID`; split owners retry a busy lock until it is available
or shutdown is requested. When the flag is false, Central runs the complete
legacy pruning cycle.

Worker must not perform cluster decommissioning: its process deliberately has
no Sensor connection manager. Cluster removal belongs to Central so connection
closure and the related cascading cleanup run in the Central datastore.

## Extension contract

New retained-cache users should expose Task 3's typed `ObserveCache` capability
from the generated store implementation. The callback must be asynchronous,
cancellation-aware, and safe to retry. Treat notification payloads as hints:
reread authoritative PostgreSQL under the datastore's own synchronization,
avoid calling persisted deletion side effects from a derived-state observer,
and ensure local writes and observer repairs share the relevant key lock.

Derived indexes should be rebuilt from current rows, not by replaying an older
event. Observers must be registered after initial datastore wiring; the cache
observer's initial full delivery closes the subscription race. On shutdown,
stop the owning job before closing the shared database.
