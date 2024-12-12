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
	pool, err := postgres.Connect(ctx, connString, "libvuln")
	if err != nil {
		return fmt.Errorf("connecting to postgres: %w", err)
	}
	defer func() {
		pool.Close()
	}()
	store, err := postgres.InitPostgresMatcherStore(ctx, pool, true)
	if err != nil {
		return fmt.Errorf("initializing postgres matcher store: %w", err)
	}
	metadataStore, err := postgres.InitPostgresMatcherMetadataStore(ctx, pool, true)
	if err != nil {
		return fmt.Errorf("initializing postgres matcher metadata store: %w", err)
	}
	locker, err := ctxlock.New(ctx, pool)
	if err != nil {
		return fmt.Errorf("creating matcher postgres locker: %w", err)
	}
	defer func() {
		_ = locker.Close(ctx)
	}()
	updater, err := vuln.New(ctx, vuln.Opts{
		Store:         store,
		Locker:        locker,
		MetadataStore: metadataStore,
		URL:           vulnsURL,
		SkipGC:        true,
	})
	if err != nil {
		return fmt.Errorf("creating updater: %w", err)
	}
	if err = updater.Update(ctx); err != nil {
		return fmt.Errorf("running updater: %w", err)
	}
	return nil
}
