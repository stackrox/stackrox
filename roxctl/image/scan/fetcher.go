package scan

import (
	"context"

	"github.com/quay/claircore"
	"github.com/quay/claircore/indexer"
)

var (
	_ indexer.FetchArena = (*LocalFetchArena)(nil)
	_ indexer.Realizer   = (*realizer)(nil)
)

type LocalFetchArena struct{}

type LocalFetcher struct{}

// Arena does coordination and global refcounting.
func (*LocalFetchArena) Realizer(context.Context) indexer.Realizer {
	return &realizer{}
}

func (*LocalFetchArena) Close(context.Context) error {
	return nil
}

type realizer struct{}

func (*realizer) Realize(ctx context.Context, ls []*claircore.Layer) error {
	for _, l := range ls {
		l.SetLocal(l.URI)
	}
	return nil
}

func (*realizer) Close() error {
	return nil
}
