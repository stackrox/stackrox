package persistentlog

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/persistentlog/store"
)

type PostgresWriter struct {
	logsStore store.Store
}

func (pw PostgresWriter) Write(p []byte) (int, error) {
	//fmt.Println("SHREWS -- In my Postgres Writer")
	// implement the functionality
	logLine := storage.PersistentLog{
		Log:       string(p[:]),
		Timestamp: types.TimestampNow(),
	}

	err := pw.logsStore.Upsert(context.Background(), &logLine)
	if err != nil {
		return 0, err
	}

	return len(p), err
}

func New() *PostgresWriter {
	return &PostgresWriter{
		logsStore: Singleton(),
	}
}
