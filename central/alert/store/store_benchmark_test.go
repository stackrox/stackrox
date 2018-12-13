package store

import (
	"fmt"
	"testing"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/uuid"
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
		store.AddAlert(alert)
	}
}

func BenchmarkUpdateAlert(b *testing.B) {
	store := getAlertStore(b)
	alert := fixtures.GetAlert()
	for i := 0; i < b.N; i++ {
		store.UpdateAlert(alert)
	}
}

func BenchmarkGetAlert(b *testing.B) {
	store := getAlertStore(b)
	alert := fixtures.GetAlert()
	store.AddAlert(alert)
	for i := 0; i < b.N; i++ {
		store.GetAlert(alert.GetId())
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
