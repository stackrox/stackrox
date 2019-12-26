package dackbox

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dackbox/crud"
)

var (
	// Bucket is the prefix for image objects in the db.
	Bucket = []byte("imageBucket")
	// ListBucket is the prefix for list image objects in the db.
	ListBucket = []byte("images_list")

	// Reader reads images.
	Reader = crud.NewReader(
		crud.WithAllocFunction(alloc),
	)

	// Upserter upserts images.
	Upserter = crud.NewUpserter(
		crud.WithKeyFunction(crud.PrefixKey(Bucket, keyFunc)),
		crud.WithPartialUpserter(ListPartialUpserter),
	)

	// Deleter deletes images and cleans up all referenced children.
	Deleter = crud.NewDeleter(
		crud.GCAllChildren(),
	)

	// ListReader reads list images from the db/
	ListReader = crud.NewReader(
		crud.WithAllocFunction(listAlloc),
	)

	// ListPartialUpserter upserts list images as part of a parent object transaction (the parent in this case is an image)
	ListPartialUpserter = crud.NewPartialUpserter(
		crud.WithSplitFunc(listImageConverter),
		crud.WithUpserter(
			crud.NewUpserter(
				crud.WithKeyFunction(crud.PrefixKey(ListBucket, keyFunc)),
			),
		),
	)
)

func init() {
	globaldb.RegisterBucket(Bucket, "Image")
	globaldb.RegisterBucket(ListBucket, "List Image")
}

func keyFunc(msg proto.Message) []byte {
	return []byte(msg.(interface{ GetId() string }).GetId())
}

func alloc() proto.Message {
	return &storage.Image{}
}

func listAlloc() proto.Message {
	return &storage.ListImage{}
}

// ProtoSplitFunction
/////////////////////
func listImageConverter(msg proto.Message) (proto.Message, []proto.Message) {
	return msg, []proto.Message{convertImageToListImage(msg.(*storage.Image))}
}

func convertImageToListImage(i *storage.Image) *storage.ListImage {
	listImage := &storage.ListImage{
		Id:          i.GetId(),
		Name:        i.GetName().GetFullName(),
		Created:     i.GetMetadata().GetV1().GetCreated(),
		LastUpdated: i.GetLastUpdated(),
	}

	if i.GetScan() != nil {
		listImage.SetComponents = &storage.ListImage_Components{
			Components: int32(len(i.GetScan().GetComponents())),
		}
		var numVulns int32
		var numFixableVulns int32
		var fixedByProvided bool
		for _, c := range i.GetScan().GetComponents() {
			numVulns += int32(len(c.GetVulns()))
			for _, v := range c.GetVulns() {
				if v.GetSetFixedBy() != nil {
					fixedByProvided = true
					if v.GetFixedBy() != "" {
						numFixableVulns++
					}
				}
			}
		}
		listImage.SetCves = &storage.ListImage_Cves{
			Cves: numVulns,
		}
		if numVulns == 0 || fixedByProvided {
			listImage.SetFixable = &storage.ListImage_FixableCves{
				FixableCves: numFixableVulns,
			}
		}
	}
	return listImage
}
