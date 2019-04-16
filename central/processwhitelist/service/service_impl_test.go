package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/central/globalindex"
	"github.com/stackrox/rox/central/processwhitelist/datastore"
	"github.com/stackrox/rox/central/processwhitelist/index"
	whitelistSearch "github.com/stackrox/rox/central/processwhitelist/search"
	"github.com/stackrox/rox/central/processwhitelist/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
)

func setupTest(t *testing.T) (*bbolt.DB, datastore.DataStore, Service) {
	db, err := bolthelper.NewTemp("process_whitelist_service_test.db")
	assert.NoError(t, err)
	wlStore := store.New(db)

	tmpIndex, err := globalindex.TempInitializeIndices("")
	assert.NoError(t, err)
	indexer := index.New(tmpIndex)

	searcher, err := whitelistSearch.New(wlStore, indexer)
	assert.NoError(t, err)

	ds := datastore.New(wlStore, indexer, searcher)
	service := New(ds)
	return db, ds, service
}

func makeIDSet(whitelists []*storage.ProcessWhitelist) set.StringSet {
	idSet := set.NewStringSet()
	for _, whitelist := range whitelists {
		idSet.Add(whitelist.GetId())
	}
	return idSet
}

func fillDB(t *testing.T, ds datastore.DataStore, whitelists []*storage.ProcessWhitelist) {
	initialContents, err := ds.GetProcessWhitelists()
	assert.NoError(t, err)
	assert.Empty(t, initialContents, "initial db not empty, you have a test bug")
	for _, whitelist := range whitelists {
		_, err := ds.AddProcessWhitelist(whitelist)
		assert.NoError(t, err)
	}
}

func emptyDB(t *testing.T, ds datastore.DataStore) {
	whitelists, err := ds.GetProcessWhitelists()
	assert.NoError(t, err)
	for _, whitelist := range whitelists {
		err := ds.RemoveProcessWhitelist(whitelist.GetId())
		assert.NoError(t, err)
	}
}

func makeWL(depID string, conName string, procs []*storage.Process) *storage.ProcessWhitelist {
	return &storage.ProcessWhitelist{
		Id:            fmt.Sprintf("%s/%s", depID, conName),
		ContainerName: conName,
		DeploymentId:  depID,
		Processes:     procs,
	}
}

func TestGetProcessWhitelists(t *testing.T) {
	db, ds, service := setupTest(t)
	defer testutils.TearDownDB(db)

	cases := []struct {
		name       string
		whitelists []*storage.ProcessWhitelist
	}{
		{
			name:       "Empty DB",
			whitelists: []*storage.ProcessWhitelist{},
		},
		{
			name:       "One whitelist",
			whitelists: []*storage.ProcessWhitelist{fixtures.GetProcessWhitelistWithID()},
		},
		{
			name: "Many whitelists",
			whitelists: []*storage.ProcessWhitelist{
				fixtures.GetProcessWhitelistWithID(),
				fixtures.GetProcessWhitelistWithID(),
				fixtures.GetProcessWhitelistWithID(),
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fillDB(t, ds, c.whitelists)
			gotWhitelists, err := service.GetProcessWhitelists((context.Context)(nil), nil)
			assert.NoError(t, err)
			gotWhitelistIDs := makeIDSet(gotWhitelists.GetWhitelists())
			expectedWhitelistIDs := makeIDSet(c.whitelists)
			assert.Equal(t, expectedWhitelistIDs, gotWhitelistIDs)
			emptyDB(t, ds)
		})
	}
}

func TestGetProcessWhitelist(t *testing.T) {
	db, ds, service := setupTest(t)
	defer testutils.TearDownDB(db)
	knownWhitelist := fixtures.GetProcessWhitelistWithID()
	cases := []struct {
		name           string
		whitelists     []*storage.ProcessWhitelist
		expectedResult *storage.ProcessWhitelist
		shouldFail     bool
	}{
		{
			name:       "Empty DB",
			whitelists: []*storage.ProcessWhitelist{},
			shouldFail: true,
		},
		{
			name:           "One whitelist",
			whitelists:     []*storage.ProcessWhitelist{knownWhitelist},
			expectedResult: knownWhitelist,
			shouldFail:     false,
		},
		{
			name: "Many Whitelists",
			whitelists: []*storage.ProcessWhitelist{
				knownWhitelist,
				fixtures.GetProcessWhitelist(),
				fixtures.GetProcessWhitelist(),
				fixtures.GetProcessWhitelist(),
			},
			expectedResult: knownWhitelist,
			shouldFail:     false,
		},
		{
			name: "Search for non-existant",
			whitelists: []*storage.ProcessWhitelist{
				fixtures.GetProcessWhitelist(),
				fixtures.GetProcessWhitelist(),
				fixtures.GetProcessWhitelist(),
			},
			shouldFail: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fillDB(t, ds, c.whitelists)
			requestByID := &v1.GetProcessWhitelistByIdRequest{WhitelistId: knownWhitelist.GetId()}
			whitelistByID, errByID := service.GetProcessWhitelist((context.Context)(nil), requestByID)
			if c.shouldFail {
				assert.Error(t, errByID)
			} else {
				assert.NoError(t, errByID)
				assert.Equal(t, c.expectedResult, whitelistByID)
			}
			emptyDB(t, ds)
		})
	}
}

func TestUpdateProcessWhitelist(t *testing.T) {
	db, ds, service := setupTest(t)
	defer testutils.TearDownDB(db)
	cases := []struct {
		name               string
		whitelists         []*storage.ProcessWhitelist
		toUpdate           []string
		toAdd              []string
		toRemove           []string
		expectedSuccessIDs set.StringSet
		expectedErrorIDs   set.StringSet
	}{
		{
			name:               "Update non-existent",
			toUpdate:           []string{"non", "existent", "ids"},
			toAdd:              []string{"Doesn't matter"},
			toRemove:           []string{"whatever"},
			expectedSuccessIDs: set.NewStringSet(),
			expectedErrorIDs:   set.NewStringSet("non", "existent", "ids"),
		},
		{
			name: "Update one",
			whitelists: []*storage.ProcessWhitelist{
				makeWL("Joseph", "Rules", []*storage.Process{fixtures.GetWhitelistProcess("Another process")}),
			},
			toUpdate:           []string{"Joseph/Rules"},
			toAdd:              []string{"Some process"},
			toRemove:           []string{"Another process"},
			expectedSuccessIDs: set.NewStringSet("Joseph/Rules"),
			expectedErrorIDs:   set.NewStringSet(),
		},
		{
			name: "Update many",
			whitelists: []*storage.ProcessWhitelist{
				makeWL("Joseph", "Rules", []*storage.Process{fixtures.GetWhitelistProcess("Another process")}),
				makeWL("Joseph", "Isthebest", []*storage.Process{fixtures.GetWhitelistProcess("Another process")}),
				makeWL("Some", "Kinda", []*storage.Process{fixtures.GetWhitelistProcess("Another process")}),
				makeWL("Other", "Whitelists", []*storage.Process{fixtures.GetWhitelistProcess("Another process")}),
				makeWL("To", "Test", []*storage.Process{fixtures.GetWhitelistProcess("Another process")}),
			},
			toUpdate:           []string{"Joseph/Rules", "Joseph/Isthebest", "Some/Kinda", "Other/Whitelists", "To/Test"},
			toAdd:              []string{"Some process"},
			toRemove:           []string{"Another process"},
			expectedSuccessIDs: set.NewStringSet("Joseph/Rules", "Joseph/Isthebest", "Some/Kinda", "Other/Whitelists", "To/Test"),
			expectedErrorIDs:   set.NewStringSet(),
		},
		{
			name: "Mixed failures",
			whitelists: []*storage.ProcessWhitelist{
				makeWL("Joseph", "Rules", []*storage.Process{fixtures.GetWhitelistProcess("Another process"), fixtures.GetWhitelistProcess("bla")}),
			},
			toUpdate:           []string{"Joseph/Rules", "Joseph/Isthebest"},
			toAdd:              []string{"Some process"},
			toRemove:           []string{"Another process"},
			expectedSuccessIDs: set.NewStringSet("Joseph/Rules"),
			expectedErrorIDs:   set.NewStringSet("Joseph/Isthebest"),
		},
		{
			name: "Unrelated list",
			whitelists: []*storage.ProcessWhitelist{
				makeWL("Joseph", "Rules", []*storage.Process{fixtures.GetWhitelistProcess("Another process")}),
				makeWL("Joseph", "Isthebest", []*storage.Process{fixtures.GetWhitelistProcess("Another process")}),
			},
			toUpdate:           []string{"Joseph/Rules"},
			toAdd:              []string{"Some process"},
			toRemove:           []string{"Another process"},
			expectedSuccessIDs: set.NewStringSet("Joseph/Rules"),
			expectedErrorIDs:   set.NewStringSet(),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fillDB(t, ds, c.whitelists)
			request := &v1.UpdateProcessWhitelistsRequest{
				WhitelistIds:       c.toUpdate,
				AddProcessNames:    c.toAdd,
				RemoveProcessNames: c.toRemove,
			}
			response, err := service.UpdateProcessWhitelists((context.Context)(nil), request)
			assert.NoError(t, err)
			succeeded := set.NewStringSet()
			for _, wl := range response.Whitelists {
				succeeded.Add(wl.Id)
				processes := set.NewStringSet()
				for _, process := range wl.Processes {
					processes.Add(process.Name)
				}
				for _, add := range c.toAdd {
					assert.True(t, processes.Contains(add))
				}
				for _, remove := range c.toRemove {
					assert.False(t, processes.Contains(remove))
				}
			}
			assert.Equal(t, c.expectedSuccessIDs, succeeded)
			failed := set.NewStringSet()
			for _, err := range response.Errors {
				failed.Add(err.Id)
			}
			assert.Equal(t, c.expectedErrorIDs, failed)
			emptyDB(t, ds)
		})
	}
}
