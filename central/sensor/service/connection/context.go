package connection

import "context"

type contextKey struct{}

// FromContext retrieves the active sensor connection from the given context (if any).
func FromContext(ctx context.Context) SensorConnection {
	conn, _ := ctx.Value(contextKey{}).(SensorConnection)
	return conn
}

func withConnection(ctx context.Context, conn SensorConnection) context.Context {
	return context.WithValue(ctx, contextKey{}, conn)
}
