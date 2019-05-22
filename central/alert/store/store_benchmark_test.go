package store

import (
	"fmt"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/alert/convert"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const maxGRPCSize = 4194304

func getAlertStore(b *testing.B) Store {
	db, err := bolthelper.NewTemp(b.Name() + ".db")
	if err != nil {
		b.Fatal(err)
	}
	return New(db)
}

func BenchmarkAddAlert(b *testing.B) {
	store := getAlertStore(b)
	alert := fixtures.GetAlert()
	for i := 0; i < b.N; i++ {
		require.NoError(b, store.AddAlert(alert))
	}
}

func BenchmarkUpdateAlert(b *testing.B) {
	store := getAlertStore(b)
	alert := fixtures.GetAlert()
	for i := 0; i < b.N; i++ {
		require.NoError(b, store.UpdateAlert(alert))
	}
}

func BenchmarkGetAlert(b *testing.B) {
	store := getAlertStore(b)
	alert := fixtures.GetAlert()
	require.NoError(b, store.AddAlert(alert))
	for i := 0; i < b.N; i++ {
		_, exists, err := store.GetAlert(alert.GetId())
		require.True(b, exists)
		require.NoError(b, err)
	}
}

func BenchmarkListAlert(b *testing.B) {
	store := getAlertStore(b)
	alert := fixtures.GetAlert()
	require.NoError(b, store.AddAlert(alert))
	for i := 0; i < b.N; i++ {
		_, err := store.ListAlerts()
		require.NoError(b, err)
	}
}

func BenchmarkStateAlert(b *testing.B) {
	store := getAlertStore(b)
	alert := fixtures.GetAlert()
	require.NoError(b, store.AddAlert(alert))
	for i := 0; i < b.N; i++ {
		_, err := store.GetAlertStates()
		require.NoError(b, err)
	}
}

// This really isn't a benchmark, but just prints out how many ListAlerts can be returned in an API call
func BenchmarkListAlerts(b *testing.B) {
	listAlert := &storage.ListAlert{
		Id:   uuid.NewDummy().String(),
		Time: types.TimestampNow(),
		Policy: &storage.ListAlertPolicy{
			Id:          uuid.NewV4().String(),
			Name:        "this is my policy name",
			Severity:    storage.Severity_MEDIUM_SEVERITY,
			Description: "this is my description and it's fairly long, but typically descriptions are fairly long",
			Categories:  []string{"Category 1", "Category 2", "Category 3"},
		},
		Deployment: &storage.ListAlertDeployment{
			Id:          uuid.NewV4().String(),
			Name:        "quizzical_cat",
			UpdatedAt:   types.TimestampNow(),
			ClusterName: "remote",
		},
	}

	bytes, _ := proto.Marshal(listAlert)
	fmt.Printf("Max ListAlerts that can be returned: %d\n", maxGRPCSize/len(bytes))
}

func BenchmarkUnmarshalAlert(b *testing.B) {
	alert := fixtures.GetAlert()
	bytes, err := proto.Marshal(alert)
	require.NoError(b, err)

	var newAlert storage.Alert
	for i := 0; i < b.N; i++ {
		assert.NoError(b, proto.Unmarshal(bytes, &newAlert))
	}
}

func BenchmarkUnmarshalListAlert(b *testing.B) {
	bytes, err := proto.Marshal(convert.AlertToListAlert(fixtures.GetAlert()))
	require.NoError(b, err)

	var newAlert storage.ListAlert
	for i := 0; i < b.N; i++ {
		assert.NoError(b, proto.Unmarshal(bytes, &newAlert))
	}
}
