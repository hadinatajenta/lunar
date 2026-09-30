package cache

import "context"

type forceRefreshContextKey struct{}

func WithForceRefresh(ctx context.Context) context.Context {
	return context.WithValue(ctx, forceRefreshContextKey{}, true)
}

func IsForceRefresh(ctx context.Context) bool {
	force, ok := ctx.Value(forceRefreshContextKey{}).(bool)
	return ok && force
}
