package pgutils

import (
	"context"
	"reflect"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
)

var (
	log = logging.LoggerForModule()
)

// ErrNilIfNoRows returns nil if the error is pgx.ErrNoRows
func ErrNilIfNoRows(err error) error {
	if err == pgx.ErrNoRows {
		return nil
	}
	return err
}

// ConvertEnumSliceToIntArray converts an enum slice into a Postgres intarray
func ConvertEnumSliceToIntArray(i interface{}) []int32 {
	enumSlice := reflect.ValueOf(i)
	enumSliceLen := enumSlice.Len()
	resultSlice := make([]int32, 0, enumSliceLen)
	for i := 0; i < enumSlice.Len(); i++ {
		resultSlice = append(resultSlice, int32(enumSlice.Index(i).Int()))
	}
	return resultSlice
}

// NilOrTime allows for a proto timestamp to be stored a timestamp type in Postgres
func NilOrTime(t *types.Timestamp) *time.Time {
	if t == nil {
		return nil
	}
	ts, err := types.TimestampFromProto(t)
	if err != nil {
		return nil
	}
	return &ts
}

// CreateTable executes input create statement using the input connection.
func CreateTable(ctx context.Context, db *pgxpool.Pool, createStmt *postgres.CreateStmts) {
	_, err := db.Exec(ctx, createStmt.Table)
	if err != nil {
		log.Panicf("Error creating table %s: %v", createStmt.Table, err)
	}

	for _, index := range createStmt.Indexes {
		if _, err := db.Exec(ctx, index); err != nil {
			log.Panicf("Error creating index %s: %v", index, err)
		}
	}

	for _, child := range createStmt.Children {
		CreateTable(ctx, db, child)
	}
}
