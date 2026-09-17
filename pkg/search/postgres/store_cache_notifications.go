package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/postgres"
)

const cacheNotificationChannel = "rox_retained_cache"

type cacheNotification struct {
	Relation uint32  `json:"r"`
	Key      *string `json:"k,omitempty"`
	Full     bool    `json:"f,omitempty"`
}

func (c *cacheCoordinator) notification(_, payload string) {
	var event cacheNotification
	if err := json.Unmarshal([]byte(payload), &event); err != nil || event.Relation == 0 || (!event.Full && event.Key == nil) {
		// A malformed wakeup is an unknown gap. A full scan is safe even when
		// an older/newer Central emitted a payload we cannot interpret.
		stores := concurrency.WithLock1(&c.mu, func() []*cacheRegistration {
			var stores []*cacheRegistration
			for _, table := range c.tables {
				stores = append(stores, table.stores...)
			}
			return stores
		})
		for _, store := range stores {
			store.invalidate()
			store.enqueue(nil, true)
		}
		return
	}
	stores := concurrency.WithLock1(&c.mu, func() []*cacheRegistration {
		var stores []*cacheRegistration
		for _, table := range c.tables {
			select {
			case <-table.ready:
				if table.relation == event.Relation {
					stores = append(stores, table.stores...)
				}
			default:
			}
		}
		return stores
	})
	for _, store := range stores {
		if event.Full {
			store.invalidate()
			store.enqueue(nil, true)
		} else {
			store.enqueue([]string{*event.Key}, false)
		}
	}
}

// installCacheNotifications resolves the real relation and single primary key,
// including schemas and non-id/UUID keys. Only keys are extracted from OLD/NEW;
// serialized protobuf columns are never converted to JSON. Notifications and
// table mutations commit (or roll back) together, including COPY and TRUNCATE.
func installCacheNotifications(ctx context.Context, db postgres.DB, table string) (uint32, error) {
	var relation uint32
	var schema, name, key string
	err := db.QueryRow(ctx, `
SELECT c.oid, n.nspname, c.relname, a.attname
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
JOIN pg_index i ON i.indrelid = c.oid AND i.indisprimary AND i.indnkeyatts = 1
JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum = i.indkey[0]
WHERE c.oid = to_regclass($1)`, table).Scan(&relation, &schema, &name, &key)
	if err != nil {
		return 0, errors.Wrapf(err, "resolving cache notification primary key for %s", table)
	}
	function := pgx.Identifier{schema, fmt.Sprintf("rox_cache_change_%d", relation)}.Sanitize()
	target := pgx.Identifier{schema, name}.Sanitize()
	column := pgx.Identifier{key}.Sanitize()
	// Quoting an identifier does not escape a surrounding dollar-quoted
	// function body. Choose a delimiter absent from the interpolated column.
	delimiter := "$rox_cache$"
	for strings.Contains(column, delimiter) {
		delimiter = strings.TrimSuffix(delimiter, "$") + "_$"
	}
	statement := fmt.Sprintf(`
CREATE OR REPLACE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS %s
DECLARE
    keys text[] := ARRAY[]::text[];
    key text;
    payload text;
BEGIN
    IF TG_OP = 'TRUNCATE' THEN
        PERFORM pg_catalog.pg_notify('rox_retained_cache', '{"r":%d,"f":true}');
        RETURN NULL;
    END IF;
    IF TG_OP = 'DELETE' OR TG_OP = 'UPDATE' THEN
        keys := array_append(keys, OLD.%s::text);
    END IF;
    IF TG_OP = 'INSERT' OR TG_OP = 'UPDATE' THEN
        keys := array_append(keys, NEW.%s::text);
    END IF;
    FOREACH key IN ARRAY keys LOOP
        payload := pg_catalog.json_build_object('r', %d, 'k', key)::text;
        IF pg_catalog.octet_length(payload) >= 7600 THEN
            payload := '{"r":%d,"f":true}';
        END IF;
        PERFORM pg_catalog.pg_notify('rox_retained_cache', payload);
    END LOOP;
    RETURN NULL;
END;
%s;
CREATE OR REPLACE TRIGGER rox_cache_row_change
AFTER INSERT OR UPDATE OR DELETE ON %s FOR EACH ROW EXECUTE FUNCTION %s();
CREATE OR REPLACE TRIGGER rox_cache_truncate
AFTER TRUNCATE ON %s FOR EACH STATEMENT EXECUTE FUNCTION %s();
`, function, delimiter, relation, column, column, relation, relation, delimiter, target, function, target, function)
	tx, err := db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// Serialize idempotent DDL across independent Central pools/processes too.
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", int64(0x726f780000000000)+int64(relation)); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, statement); err != nil {
		return 0, errors.Wrapf(err, "installing cache notifications on %s", target)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return relation, nil
}
