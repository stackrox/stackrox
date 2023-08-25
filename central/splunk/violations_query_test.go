package splunk

// This file contains datastore helpers and tests for queryAlerts() function.

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/pgtest"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Test suite to test everything to do with splunk violations. Actual test is split across multiple files.
// Moved here since some helpers in this file references it and for non-postgres tests, it won't be able to read it
// if the definition is another file
type violationsTestSuite struct {
	suite.Suite
	deployAlert, processAlert, k8sAlert, networkAlert, resourceAlert *storage.Alert
	allowCtx                                                         context.Context
}

func makeTimestamp(timeStr string) *types.Timestamp {
	ts, err := types.TimestampProto(mustParseTime(timeStr))
	utils.CrashOnError(err)
	return ts
}

func mustParseTime(timeStr string) time.Time {
	ts, err := time.Parse(time.RFC3339Nano, timeStr)
	utils.CrashOnError(err)
	return ts
}

// testDataStore contains all things that need to be created and disposed in order to use Alerts datastore.DataStore in
// tests.
type testDataStore struct {
	testDB   *pgtest.TestPostgres
	alertsDS datastore.DataStore
}

// makeDS creates a new temp datastore with only provided alerts for use in tests.
func makeDS(t *testing.T, alerts []*storage.Alert) testDataStore {
	testDB := pgtest.ForT(t)
	assert.NotNil(t, testDB)
	alertsDS, err := datastore.GetTestPostgresDataStore(t, testDB.DB)
	require.NoError(t, err)

	err = alertsDS.UpsertAlerts(sac.WithAllAccess(context.Background()), alerts)
	require.NoError(t, err)

	return testDataStore{testDB: testDB, alertsDS: alertsDS}
}

// teardown cleans up test datastore.
func (d *testDataStore) teardown(t *testing.T) {
	d.testDB.Teardown(t)
}

// these simply converts varargs to slice for slightly less typing (pun intended).
func these(alerts ...*storage.Alert) []*storage.Alert {
	return alerts
}

// withAlerts runs action in a scope of test datastore.DataStore that contains given alerts.
func (s *violationsTestSuite) withAlerts(alerts []*storage.Alert, action func(alertsDS datastore.DataStore)) {
	ds := makeDS(s.T(), alerts)
	defer ds.teardown(s.T())
	action(ds.alertsDS)
}

func (s *violationsTestSuite) TestQueryReturnsAlertsWithAllStates() {
	a1 := s.processAlert.Clone()
	a1.State = storage.ViolationState_SNOOZED
	a2 := s.k8sAlert.Clone()
	a2.State = storage.ViolationState_RESOLVED
	a3 := s.deployAlert.Clone()
	a3.State = storage.ViolationState_ATTEMPTED

	var alertIDs []string

	s.withAlerts(these(a1, a2, a3), func(alertsDS datastore.DataStore) {
		alertIDs = s.queryAlertsWithCheckpoint(alertsDS, "2000-01-01T00:00:00Z", defaultPaginationSettings.maxAlertsFromQuery)
	})

	s.Len(alertIDs, 3)
	s.Contains(alertIDs, s.processAlert.Id)
	s.Contains(alertIDs, s.k8sAlert.Id)
	s.Contains(alertIDs, s.deployAlert.Id)
}

func (s *violationsTestSuite) TestQueryAlertsWithTimestamp() {
	cases := []struct {
		name      string
		timestamp string
		ids       []string
	}{
		{
			name:      "More than a day earlier",
			timestamp: "2021-01-31T00:00:00Z",
			ids:       []string{s.processAlert.Id, s.k8sAlert.Id, s.deployAlert.Id},
		}, {
			name:      "Earlier same day",
			timestamp: "2021-02-01T16:05:00Z",
			ids:       []string{s.processAlert.Id, s.k8sAlert.Id, s.deployAlert.Id},
		}, {
			name:      "Between two, within a day",
			timestamp: "2021-02-01T17:00:00Z",
			ids:       []string{s.processAlert.Id, s.k8sAlert.Id},
		}, {
			name:      "Between two, different days",
			timestamp: "2021-02-05T00:00:00Z",
			ids:       []string{s.k8sAlert.Id},
		}, {
			name:      "After last, the same day",
			timestamp: "2021-02-15T19:04:37Z",
			ids:       []string{},
		}, {
			name:      "More than a day after last",
			timestamp: "2021-02-17T19:04:37Z",
			ids:       []string{},
		},
	}

	s.withAlerts(these(s.processAlert, s.k8sAlert, s.deployAlert), func(alertsDS datastore.DataStore) {
		for _, c := range cases {
			s.Run(c.name, func() {
				alerts := s.queryAlertsWithCheckpoint(alertsDS, c.timestamp, defaultPaginationSettings.maxAlertsFromQuery)
				s.Len(alerts, len(c.ids))
				for _, id := range c.ids {
					s.Contains(alerts, id)
				}
			})
		}
	})
}

func (s *violationsTestSuite) TestQueryAlertsAreSortedByAlertID() {
	alerts := []*storage.Alert{s.processAlert, s.k8sAlert, s.deployAlert, s.networkAlert}
	sortedIDs := make([]string, 0, len(alerts))
	// Generate new random UUIDs for Alerts. This will make it highly likely that default ordering of alerts by
	// decreasing timestamp does not match ordering by ID. Therefore we'll be able to validate that our requested
	// sorting by ID really works.
	for _, a := range alerts {
		a.Id = uuid.NewV4().String()
		sortedIDs = append(sortedIDs, a.Id)
	}
	sort.Slice(sortedIDs, func(i, j int) bool {
		return sortedIDs[i] < sortedIDs[j]
	})

	var resultIDs []string
	s.withAlerts(alerts, func(alertsDS datastore.DataStore) {
		resultIDs = s.queryAlertsWithCheckpoint(alertsDS, "2000-01-01T00:00:00Z", defaultPaginationSettings.maxAlertsFromQuery)
	})

	s.Equal(sortedIDs, resultIDs)
}

func (s *violationsTestSuite) TestQueryAlertsFromAlertIDAndWithLimit() {
	ids := []string{
		"86a55daa-de0d-4649-a7a9-ad71eeebfb6a",
		"90e0feed-662c-4593-b414-e55d1eaff017",
		"f2d0efaa-2c54-402c-aeed-5b88ed5ccb8a",
		"f56ffae8-adf9-4983-8e56-e260f1ab3dc9",
	}
	nothing := ids[4:]

	cases := []struct {
		fromAlertID string
		limit       int32
		result      []string
	}{
		{fromAlertID: "", result: ids},
		{fromAlertID: "86a55daa-de0d-4649-a7a9-ad71eeebfb69", result: ids}, // Last letter is different.
		{fromAlertID: "86a55daa-de0d-4649-a7a9-ad71eeebfb6a", result: ids[1:]},
		{fromAlertID: "87", result: ids[1:]},
		{fromAlertID: "AA", result: ids[2:]},
		{fromAlertID: "ZZ", result: ids[2:]},
		{fromAlertID: "a", result: ids[2:]},
		{fromAlertID: "f56ffae8-adf9-4983-8e56-e260f1ab3dc", result: ids[3:]}, // Last letter was removed.
		{fromAlertID: "f56ffae8-adf9-4983-8e56-e260f1ab3dc9", result: nothing},
		{fromAlertID: "f6", result: nothing},
		{fromAlertID: "zzz", result: nothing},
		{limit: 0, result: ids}, // limit==0 works as if there's no limit.
		{limit: 1, result: ids[0:1]},
		{limit: 2, result: ids[0:2]},
		{limit: 4, result: ids},
		{limit: 5, result: ids},
		{fromAlertID: "0", limit: 1, result: ids[:1]},
		{fromAlertID: "86a55daa-de0d-4649-a7a9-ad71eeebfb6a", limit: 3, result: ids[1:]},
		{fromAlertID: "90e0feed-662c-4593-b414-e55d1eaff017", limit: 1, result: ids[2:3]},
		{fromAlertID: "f56ffae8-adf9-4983-8e56-e260f1ab3dc9", limit: 0, result: nothing},
		// Default limit should be sufficient to return all our test alerts, otherwise it is too small.
		{fromAlertID: "", limit: defaultPaginationSettings.maxAlertsFromQuery, result: ids},
	}

	s.withAlerts(these(s.processAlert, s.k8sAlert, s.deployAlert, s.networkAlert), func(alertsDS datastore.DataStore) {
		for _, c := range cases {
			s.Run(fmt.Sprintf("from:%q, limit:%d", c.fromAlertID, c.limit), func() {
				result := s.queryAlertsWithCheckpoint(alertsDS, "2000-01-01T00:00:00Z__2021-03-26T17:36:00Z__"+c.fromAlertID, c.limit)

				s.Equal(c.result, result)
			})
		}
	})
}

func (s *violationsTestSuite) queryAlertsWithCheckpoint(alertsDS datastore.DataStore, checkpoint string, maxAlerts int32) []string {
	returnedAlerts, err := queryAlerts(s.allowCtx, alertsDS, mustParseCheckpoint(s.T(), checkpoint), maxAlerts)
	s.NoError(err)

	alertIDs := make([]string, 0, len(returnedAlerts))
	for _, a := range returnedAlerts {
		alertIDs = append(alertIDs, a.Id)
	}

	return alertIDs
}
