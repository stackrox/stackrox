package matcher

import (
	"context"
	"fmt"
	"net/http"

	"github.com/quay/claircore/datastore/postgres"
	"github.com/quay/claircore/enricher/cvss"
	"github.com/quay/claircore/libvuln"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/claircore/pkg/ctxlock"
)

// Matcher represents a vulnerability matcher.
type Matcher interface {
	Close(ctx context.Context) error
}

type matcherImpl struct {
	matcher *libvuln.Libvuln
}

// NewMatcher creates a new matcher.
func NewMatcher(ctx context.Context) (Matcher, error) {
	pool, err := postgres.Connect(ctx, "postgresql:///postgres?host=/var/run/postgresql", "libvuln")
	if err != nil {
		return nil, fmt.Errorf("connecting to postgres for matcher: %w", err)
	}
	store, err := postgres.InitPostgresMatcherStore(ctx, pool, true)
	if err != nil {
		return nil, fmt.Errorf("initializing postgres matcher store: %w", err)
	}
	locker, err := ctxlock.New(ctx, pool)
	if err != nil {
		return nil, fmt.Errorf("creating matcher postgres locker: %w", err)
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
		return nil, fmt.Errorf("creating libvuln: %w", err)
	}
	return &matcherImpl{
		matcher: matcher,
	}, nil
}

// Close closes the matcher.
func (i *matcherImpl) Close(ctx context.Context) error {
	return i.matcher.Close(ctx)
}
