package test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/quay/claircore/datastore/postgres"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/claircore/libvuln/updates"
	"github.com/quay/claircore/test/integration"
	pgtest "github.com/quay/claircore/test/postgres"
	"github.com/quay/zlog"
	"github.com/stackrox/stackrox/scanner/v4/updater/manual"
)

func ManualUpdater(t *testing.T) {
	ctx := context.Background()
	var updaters []driver.Updater

	// Append updater sets directly to the updaters
	appendUpdaterSet := func(updaterSet driver.UpdaterSet, err error) {
		if err != nil {
			zlog.Error(ctx).Msg(err.Error())
			return
		}
		updaters = append(updaters, updaterSet.Updaters()...)
	}
	integration.NeedDB(t)
	pool := pgtest.TestMatcherDB(ctx, t)
	store := postgres.NewMatcherStore(pool)
	appendUpdaterSet(manual.UpdaterSet(ctx, manuallyTestVulns))
	updaterSetMgr, err := updates.NewManager(ctx, store, updates.NewLocalLockSource(), http.DefaultClient,
		updates.WithOutOfTree(updaters),
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	if err = updaterSetMgr.Run(ctx); err != nil {
		fmt.Println(err)
		return
	}
}

// TestExampleManualUpdater: go test -run ExampleManualUpdater
func TestExampleManualUpdater(t *testing.T) {
	ManualUpdater(t)
}
