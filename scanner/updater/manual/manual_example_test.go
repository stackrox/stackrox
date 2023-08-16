package manual

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/quay/claircore"
	"github.com/quay/claircore/datastore/postgres"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/claircore/libvuln/updates"
	"github.com/quay/claircore/test/integration"
	pgtest "github.com/quay/claircore/test/postgres"
	"github.com/quay/zlog"
)

var manuallyTestVulns = []*claircore.Vulnerability{
	{
		Updater:            "manual",
		Name:               "GHSA-cj7v-27pg-wf7q",
		Description:        "URI use within Jetty's HttpURI class can parse invalid URIs such as http://localhost;/path as having an authority with a host of localhost;A URIs of the type http://localhost;/path should be interpreted to be either invalid or as localhost; to be the userinfo and no host. However, HttpURI.host returns localhost; which is definitely wrong.",
		Links:              "https://github.com/github/advisory-database/blob/main/advisories/github-reviewed/2022/07/GHSA-cj7v-27pg-wf7q/GHSA-cj7v-27pg-wf7q.json",
		Severity:           "CVSS:3.1/AV:N/AC:L/PR:H/UI:N/S:U/C:N/I:L/A:N",
		NormalizedSeverity: claircore.Low,
		Package: &claircore.Package{
			Name: "org.eclipse.jetty:jetty-http",
			Kind: claircore.BINARY,
		},
		FixedInVersion: "fixed=9.4.47&introduced=0",
		Repo: &claircore.Repository{
			Name: "maven",
		},
	}}

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
	appendUpdaterSet(UpdaterSet(ctx, manuallyTestVulns))
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
