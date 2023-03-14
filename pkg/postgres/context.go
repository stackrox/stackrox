package postgres

import "context"

type txContextKey struct{}

// ContextWithTx adds a database transaction to the context
func ContextWithTx(ctx context.Context, tx *Tx) context.Context {
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
