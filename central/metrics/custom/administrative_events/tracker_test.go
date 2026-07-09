package administrative_events

import (
	"context"
	"testing"

	adminEventDS "github.com/stackrox/rox/central/administration/events/datastore/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func Test_track(t *testing.T) {
	ctrl := gomock.NewController(t)
	ds := adminEventDS.NewMockDataStore(ctrl)

	ds.EXPECT().ListEvents(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ *v1.Query) ([]*storage.AdministrationEvent, error) {
			return []*storage.AdministrationEvent{
				{
					Type:   storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_LOG_MESSAGE,
					Level:  storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_ERROR,
					Domain: "AuthProvider",
					Resource: &storage.AdministrationEvent_Resource{
						Type: "AuthProvider",
						Name: "LDAP",
					},
					NumOccurrences: 5,
				},
				{
					Type:   storage.AdministrationEventType_ADMINISTRATION_EVENT_TYPE_GENERIC,
					Level:  storage.AdministrationEventLevel_ADMINISTRATION_EVENT_LEVEL_WARNING,
					Domain: "PolicyManagement",
					Resource: &storage.AdministrationEvent_Resource{
						Type: "Policy",
						Name: "No bash",
					},
					NumOccurrences: 3,
				},
			}, nil
		})

	var findings []*finding
	for f := range track(context.Background(), ds) {
		findings = append(findings, f)
	}

	assert.Len(t, findings, 2)

	assert.Equal(t, "ADMINISTRATION_EVENT_TYPE_LOG_MESSAGE", LazyLabels["Type"](findings[0]))
	assert.Equal(t, "ADMINISTRATION_EVENT_LEVEL_ERROR", LazyLabels["Level"](findings[0]))
	assert.Equal(t, "AuthProvider", LazyLabels["Domain"](findings[0]))
	assert.Equal(t, "AuthProvider", LazyLabels["ResourceType"](findings[0]))
	assert.Equal(t, "LDAP", LazyLabels["ResourceName"](findings[0]))
	assert.Equal(t, 5, findings[0].GetIncrement())

	assert.Equal(t, "ADMINISTRATION_EVENT_TYPE_GENERIC", LazyLabels["Type"](findings[1]))
	assert.Equal(t, "PolicyManagement", LazyLabels["Domain"](findings[1]))
	assert.Equal(t, 3, findings[1].GetIncrement())
}

func Test_track_nil_ds(t *testing.T) {
	var count int
	for range track(context.Background(), nil) {
		count++
	}
	assert.Zero(t, count)
}
