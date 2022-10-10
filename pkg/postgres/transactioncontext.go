package postgres

import (
	"context"

	"github.com/jackc/pgx/v4"
)

type transactionKey struct{}

// WithTransactionContext appends a transaction to the returned context
func WithTransactionContext(ctx context.Context, transaction *pgx.Tx) context.Context {
	return context.WithValue(ctx, transactionKey{}, transaction)
}

// GetTransactionFromContext returns the transaction appended to the context or nil if no transaction exists
func GetTransactionFromContext(ctx context.Context) *pgx.Tx {
	val, ok := ctx.Value(transactionKey{}).(*pgx.Tx)
	if ok {
		return val
	}
	return nil
}
