package client

import "context"

type contextKey struct{}

// SetInContext adds the sac client to the context for use in checks.
func SetInContext(ctx context.Context, client Client) context.Context {
	return context.WithValue(ctx, contextKey{}, client)
}

// GetFromContext retrieves the client from the context (if present) to use for sac checks.
func GetFromContext(ctx context.Context) Client {
	client, _ := ctx.Value(contextKey{}).(Client)
	return client
}
