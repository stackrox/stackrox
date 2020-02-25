package counter

import "context"

// DerivedFieldCounter provides functionality to obtain derived field counts for given ids
type DerivedFieldCounter interface {
	Count(ctx context.Context, ids ...string) (map[string]int32, error)
}
