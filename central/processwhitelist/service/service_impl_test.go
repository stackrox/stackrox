package service

import (
	"context"
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
	"github.com/stackrox/rox/pkg/sliceutils"
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
		assert.NoError(t, ds.RemoveProcessWhitelist(whitelist.GetKey()))
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
			name: "Empty DB",
		},
		{
			name:       "One whitelist",
			whitelists: []*storage.ProcessWhitelist{fixtures.GetProcessWhitelistWithKey()},
		},
		{
			name: "Many whitelists",
			whitelists: []*storage.ProcessWhitelist{
				fixtures.GetProcessWhitelistWithKey(),
				fixtures.GetProcessWhitelistWithKey(),
				fixtures.GetProcessWhitelistWithKey(),
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fillDB(t, ds, c.whitelists)
			defer emptyDB(t, ds)
			gotWhitelists, err := service.GetProcessWhitelists((context.Context)(nil), nil)
			assert.NoError(t, err)
			assert.ElementsMatch(t, gotWhitelists.Whitelists, c.whitelists)
		})
	}
}

func TestGetProcessWhitelist(t *testing.T) {
	db, ds, service := setupTest(t)
	defer testutils.TearDownDB(db)
	knownWhitelist := fixtures.GetProcessWhitelistWithKey()
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
				fixtures.GetProcessWhitelistWithKey(),
				fixtures.GetProcessWhitelistWithKey(),
				fixtures.GetProcessWhitelistWithKey(),
			},
			expectedResult: knownWhitelist,
			shouldFail:     false,
		},
		{
			name: "Search for non-existant",
			whitelists: []*storage.ProcessWhitelist{
				fixtures.GetProcessWhitelistWithKey(),
				fixtures.GetProcessWhitelistWithKey(),
				fixtures.GetProcessWhitelistWithKey(),
			},
			shouldFail: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fillDB(t, ds, c.whitelists)
			defer emptyDB(t, ds)
			requestByKey := &v1.GetProcessWhitelistRequest{Key: knownWhitelist.GetKey()}
			whitelist, err := service.GetProcessWhitelist((context.Context)(nil), requestByKey)
			if c.shouldFail {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, c.expectedResult, whitelist)
			}
			emptyDB(t, ds)
		})
	}
}

func TestUpdateProcessWhitelist(t *testing.T) {
	db, ds, service := setupTest(t)
	defer testutils.TearDownDB(db)

	stockProcesses := []string{"stock_process_1", "stock_process_2"}

	whitelistCollection := make(map[int]*storage.ProcessWhitelist)
	getWhitelist := func(index int) *storage.ProcessWhitelist {
		if whitelist, ok := whitelistCollection[index]; ok {
			return whitelist
		}
		whitelist := fixtures.GetProcessWhitelistWithKey()
		whitelist.Elements = make([]*storage.WhitelistElement, 0, len(stockProcesses))
		for _, stockProcess := range stockProcesses {
			whitelist.Elements = append(whitelist.Elements, &storage.WhitelistElement{
				Element: &storage.WhitelistItem{Item: &storage.WhitelistItem_ProcessName{ProcessName: stockProcess}},
			})
		}
		whitelistCollection[index] = whitelist
		return whitelist
	}

	getWhitelists := func(indexes ...int) []*storage.ProcessWhitelist {
		whitelists := make([]*storage.ProcessWhitelist, 0, len(indexes))
		for _, i := range indexes {
			whitelists = append(whitelists, getWhitelist(i))
		}
		return whitelists
	}

	getWhitelistKey := func(index int) *storage.ProcessWhitelistKey {
		return getWhitelist(index).GetKey()
	}

	getWhitelistKeys := func(indexes ...int) []*storage.ProcessWhitelistKey {
		keys := make([]*storage.ProcessWhitelistKey, 0, len(indexes))
		for _, i := range indexes {
			keys = append(keys, getWhitelistKey(i))
		}
		return keys
	}

	cases := []struct {
		name                string
		whitelists          []*storage.ProcessWhitelist
		toUpdate            []*storage.ProcessWhitelistKey
		toAdd               []string
		toRemove            []string
		expectedSuccessKeys []*storage.ProcessWhitelistKey
		expectedErrorKeys   []*storage.ProcessWhitelistKey
	}{
		{
			name:              "Update non-existent",
			toUpdate:          getWhitelistKeys(0, 1),
			toAdd:             []string{"Doesn't matter"},
			toRemove:          []string{"whatever"},
			expectedErrorKeys: getWhitelistKeys(0, 1),
		},
		{
			name:                "Update one",
			whitelists:          getWhitelists(0),
			toUpdate:            getWhitelistKeys(0),
			toAdd:               []string{"Some process"},
			toRemove:            []string{stockProcesses[0]},
			expectedSuccessKeys: getWhitelistKeys(0),
		},
		{
			name:                "Update many",
			whitelists:          getWhitelists(0, 1, 2, 3, 4),
			toUpdate:            getWhitelistKeys(0, 1, 2, 3, 4),
			toAdd:               []string{"Some process"},
			expectedSuccessKeys: getWhitelistKeys(0, 1, 2, 3, 4),
		},
		{
			name:                "Mixed failures",
			whitelists:          getWhitelists(0),
			toUpdate:            getWhitelistKeys(0, 1),
			toAdd:               []string{"Some process"},
			toRemove:            []string{stockProcesses[0]},
			expectedSuccessKeys: getWhitelistKeys(0),
			expectedErrorKeys:   getWhitelistKeys(1),
		},
		{
			name:                "Unrelated list",
			whitelists:          getWhitelists(0, 1),
			toUpdate:            getWhitelistKeys(0),
			toAdd:               []string{"Some process"},
			toRemove:            []string{stockProcesses[1]},
			expectedSuccessKeys: getWhitelistKeys(0),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fillDB(t, ds, c.whitelists)
			defer emptyDB(t, ds)

			request := &v1.UpdateProcessWhitelistsRequest{
				Keys:           c.toUpdate,
				AddElements:    fixtures.MakeElements(c.toAdd),
				RemoveElements: fixtures.MakeElements(c.toRemove),
			}
			response, err := service.UpdateProcessWhitelists((context.Context)(nil), request)
			assert.NoError(t, err)
			var successKeys []*storage.ProcessWhitelistKey
			for _, wl := range response.Whitelists {
				successKeys = append(successKeys, wl.GetKey())
				processes := set.NewStringSet()
				for _, process := range wl.Elements {
					processes.Add(process.GetElement().GetProcessName())
				}
				for _, add := range c.toAdd {
					assert.True(t, processes.Contains(add))
				}
				for _, remove := range c.toRemove {
					assert.False(t, processes.Contains(remove))
				}
				for _, stockProcess := range stockProcesses {
					if sliceutils.StringFind(c.toRemove, stockProcess) == -1 {
						assert.True(t, processes.Contains(stockProcess))
					}
				}
			}
			assert.ElementsMatch(t, c.expectedSuccessKeys, successKeys)
			var errorKeys []*storage.ProcessWhitelistKey
			for _, err := range response.Errors {
				errorKeys = append(errorKeys, err.GetKey())
			}
			assert.ElementsMatch(t, c.expectedErrorKeys, errorKeys)
		})
	}
}
