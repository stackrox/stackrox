package indexer

import (
	"context"
	"net/http"

	"github.com/quay/claircore/indexer"
)

var (
	_ indexer.FetchArena = (*NodeArena)(nil)
)

type NodeArena struct {
	wc *http.Client
}

// NewNodeArena returns an initialized NodeArena.
func NewNodeArena(wc *http.Client, mountPath string) *NodeArena {
	return &NodeArena{
		wc: wc,
	}
}

func (n NodeArena) Realizer(ctx context.Context) indexer.Realizer {
	//TODO implement me
	panic("implement me")
}

func (n NodeArena) Close(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}
