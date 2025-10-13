package postgres

import "context"

type txContextKey struct{}

// ContextWithTx adds a database transaction to the context
// The tx will be modified to be used as an outer tx.
func ContextWithTx(ctx context.Context, tx *Tx) context.Context {
	tx.UseInContext()
	return context.WithValue(ctx, txContextKey{}, tx)
}

// TxFromContext gets a database transaction from the context if it exists
func TxFromContext(ctx context.Context) (*Tx, bool) {
	obj := ctx.Value(txContextKey{})
	if obj == nil {
		return nil, false
	}
	return obj.(*Tx), true
}

// HasTxInContext returns true if the tx is in the context
func HasTxInContext(ctx context.Context) bool {
	return ctx.Value(txContextKey{}) != nil
}
