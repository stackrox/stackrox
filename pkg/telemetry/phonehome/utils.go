package phonehome

import (
	"context"

	"github.com/pkg/errors"
)

// AddTotal sets an entry in the props map with key and number of elements
// returned by f as the value.
func AddTotal[T any](ctx context.Context, props Properties, key string, f func(context.Context) ([]*T, error)) error {
	ps, err := f(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to get %s", key)
	}
	props["Total "+key] = len(ps)
	return nil
}
