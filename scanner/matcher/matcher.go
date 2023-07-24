package matcher

import (
	"context"
	"net/http"

	"github.com/quay/claircore/datastore/postgres"
	"github.com/quay/claircore/enricher/cvss"
	"github.com/quay/claircore/libvuln"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/claircore/pkg/ctxlock"
)

type Matcher struct {
	matcher *libvuln.Libvuln
}

func NewMatcher(ctx context.Context) (*Matcher, error) {
	pool, err := postgres.Connect(ctx, "postgresql:///postgres?host=/var/run/postgresql", "libvuln")
	if err != nil {
		return nil, err
	}
	store, err := postgres.InitPostgresMatcherStore(ctx, pool, true)
	if err != nil {
		return nil, err
	}
	locker, err := ctxlock.New(ctx, pool)
	if err != nil {
		return nil, err
	}

	// TODO: Update HTTP client.
	c := http.DefaultClient
	matcher, err := libvuln.New(ctx, &libvuln.Options{
		Store:  store,
		Locker: locker,
		// Run in "air-gapped" mode.
		DisableBackgroundUpdates: true,
		UpdateRetention:          libvuln.DefaultUpdateRetention,
		Client:                   c,
		Enrichers: []driver.Enricher{
			&cvss.Enricher{},
		},
	})
	if err != nil {
		return nil, err
	}
	return &Matcher{
		matcher: matcher,
	}, nil
}

func (i *Matcher) Close(ctx context.Context) error {
	return i.matcher.Close(ctx)
}
