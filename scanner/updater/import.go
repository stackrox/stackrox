package updater

import (
	"context"
	"fmt"

	"github.com/quay/claircore/pkg/ctxlock"
	"github.com/stackrox/rox/scanner/datastore/postgres"
	"github.com/stackrox/rox/scanner/matcher/updater/vuln"
)

// Load loads vulnerabilities into the Matcher DB.
func Load(ctx context.Context, connString, vulnsURL string) error {
	var success bool

	pool, err := postgres.Connect(ctx, connString, "libvuln")
	if err != nil {
		return fmt.Errorf("connecting to postgres: %w", err)
	}
	defer func() {
		if !success {
			pool.Close()
		}
	}()
	store, err := postgres.InitPostgresMatcherStore(ctx, pool, true)
	if err != nil {
		return fmt.Errorf("initializing postgres matcher store: %w", err)
	}
	locker, err := ctxlock.New(ctx, pool)
	if err != nil {
		return fmt.Errorf("creating matcher postgres locker: %w", err)
	}
	defer func() {
		if !success {
			_ = locker.Close(ctx)
		}
	}()

	metadataStore, err := postgres.InitPostgresMatcherMetadataStore(ctx, pool, true)
	if err != nil {
		return fmt.Errorf("initializing postgres matcher metadata store: %w", err)
	}
	updater, err := vuln.New(ctx, vuln.Opts{
		Store:         store,
		Locker:        locker,
		Pool:          pool,
		MetadataStore: metadataStore,
		URL:           vulnsURL,
	})
	if err != nil {
		return fmt.Errorf("creating updater: %w", err)
	}
	if err = updater.Update(ctx); err != nil {
		return fmt.Errorf("running updater: %w", err)
	}
	success = true
	return nil
}
