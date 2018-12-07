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

func getImageStore(b *testing.B) Store {
	db, err := bolthelper.NewTemp(b.Name() + ".db")
	if err != nil {
		b.Fatal(err)
	}
	return New(db)
}

func BenchmarkAddImage(b *testing.B) {
	store := getImageStore(b)
	image := fixtures.GetImage()
	for i := 0; i < b.N; i++ {
		store.UpsertImage(image)
	}
}

func BenchmarkGetImage(b *testing.B) {
	store := getImageStore(b)
	image := fixtures.GetImage()
	store.UpsertImage(image)
	for i := 0; i < b.N; i++ {
		store.GetImage(image.GetId())
	}
}

func BenchmarkListImage(b *testing.B) {
	store := getImageStore(b)
	image := fixtures.GetImage()
	store.UpsertImage(image)
	for i := 0; i < b.N; i++ {
		store.ListImage(image.GetId())
	}
}

// This really isn't a benchmark, but just prints out how many ListImages can be returned in an API call
func BenchmarkMaxListImage(b *testing.B) {
	listImage := &storage.ListImage{
		Id:   uuid.NewDummy().String(),
		Name: "quizzical_cat",
		SetComponents: &storage.ListImage_Components{
			Components: 10,
		},
		SetCves: &storage.ListImage_Cves{
			Cves: 10,
		},
		SetFixable: &storage.ListImage_FixableCves{
			FixableCves: 10,
		},
		Created: types.TimestampNow(),
	}

	bytes, _ := proto.Marshal(listImage)
	fmt.Printf("Max ListImages that can be returned: %d\n", maxGRPCSize/len(bytes))
}
