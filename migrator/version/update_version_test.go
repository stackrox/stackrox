package version

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stretchr/testify/require"
)

// Embed DB so unexpected calls fail rather than providing unrelated test behavior.
type versionUpdateDB struct {
	postgres.DB
	exec func(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
}

func (db *versionUpdateDB) Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	return db.exec(ctx, query, args...)
}

func TestUpdateVersionPostgres(t *testing.T) {
	dbErr := errors.New("database unavailable")
	for name, tc := range map[string]struct {
		tag       string
		err       error
		wantError string
	}{
		"updated":          {tag: "UPDATE 1"},
		"missing record":   {tag: "UPDATE 0", wantError: "expected one version record, updated 0"},
		"multiple records": {tag: "UPDATE 2", wantError: "expected one version record, updated 2"},
		"database error":   {err: dbErr, wantError: "updating migration version: database unavailable"},
	} {
		t.Run(name, func(t *testing.T) {
			calls := 0
			db := &versionUpdateDB{exec: func(_ context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
				calls++
				require.Equal(t, "UPDATE versions SET seqnum = $1, version = $2, minseqnum = $3, lastpersisted = $4", query)
				require.Len(t, args, 4)
				require.Equal(t, []interface{}{int32(227), "5.1.0", int32(220)}, args[:3])
				require.Nil(t, args[3])
				return pgconn.NewCommandTag(tc.tag), tc.err
			}}
			err := UpdateVersionPostgres(context.Background(), db, &storage.Version{SeqNum: 227, Version: "5.1.0", MinSeqNum: 220})
			require.Equal(t, 1, calls, "failed writes must not be retried or repaired")
			if tc.wantError == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.wantError)
				if tc.err != nil {
					require.ErrorIs(t, err, tc.err)
				}
			}
		})
	}
}
