package badger

import (
	"time"

	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/proto"
	protoTypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/globaldb"
	"github.com/stackrox/rox/central/image/datastore/internal/store"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	generic "github.com/stackrox/rox/pkg/badgerhelper/crud"
	"github.com/stackrox/rox/pkg/images/types"
	ops "github.com/stackrox/rox/pkg/metrics"
)

var (
	imageBucket     = []byte("imageBucket")
	listImageBucket = []byte("images_list")
)

type storeImpl struct {
	imageCRUD          generic.Crud
	noUpdateTimestamps bool
}

func alloc() proto.Message {
	return &storage.Image{}
}

func keyFunc(msg proto.Message) []byte {
	return []byte(msg.(*storage.Image).GetId())
}

func listAlloc() proto.Message {
	return &storage.ListImage{}
}

func converter(msg proto.Message) proto.Message {
	return convertImageToListImage(msg.(*storage.Image))
}

// New returns a new Store instance using the provided bolt DB instance.
// noUpdateTimestamps controls whether timestamps are automatically updated
// whenever an image is upserted.
func New(db *badger.DB, noUpdateTimestamps bool) store.Store {
	globaldb.RegisterBucket(imageBucket, "Image")
	globaldb.RegisterBucket(listImageBucket, "Image")
	return &storeImpl{
		imageCRUD:          generic.NewCRUDWithPartial(db, imageBucket, keyFunc, alloc, listImageBucket, listAlloc, converter),
		noUpdateTimestamps: noUpdateTimestamps,
	}
}

// ListImage returns ListImage with given id.
func (b *storeImpl) ListImage(id string) (image *storage.ListImage, exists bool, err error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Get, "ListImage")

	digest := types.NewDigest(id).Digest()

	var msg proto.Message
	msg, exists, err = b.imageCRUD.ReadPartial(digest)
	if err != nil || !exists {
		return
	}
	image = msg.(*storage.ListImage)
	return
}

// ListImages returns all ListImages
func (b *storeImpl) ListImages() ([]*storage.ListImage, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.GetMany, "ListImage")

	msgs, err := b.imageCRUD.ReadAllPartial()
	if err != nil {
		return nil, err
	}
	images := make([]*storage.ListImage, 0, len(msgs))
	for _, m := range msgs {
		images = append(images, m.(*storage.ListImage))
	}
	return images, nil
}

// GetImages returns all images regardless of request
func (b *storeImpl) GetImages() ([]*storage.Image, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.GetAll, "Image")

	msgs, err := b.imageCRUD.ReadAll()
	if err != nil {
		return nil, err
	}
	images := make([]*storage.Image, 0, len(msgs))
	for _, m := range msgs {
		images = append(images, m.(*storage.Image))
	}
	return images, nil
}

// CountImages returns all images regardless of request
func (b *storeImpl) CountImages() (int, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Count, "Image")

	return b.imageCRUD.Count()
}

// GetImage returns image with given id.
func (b *storeImpl) GetImage(id string) (image *storage.Image, exists bool, err error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Get, "Image")

	digestID := types.NewDigest(id).Digest()

	var msg proto.Message
	msg, exists, err = b.imageCRUD.Read(digestID)
	if err != nil || !exists {
		return
	}
	image = msg.(*storage.Image)
	return
}

// GetImagesBatch returns image with given sha.
func (b *storeImpl) GetImagesBatch(digests []string) ([]*storage.Image, error) {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.GetMany, "Image")

	for i, s := range digests {
		digests[i] = types.NewDigest(s).Digest()
	}

	msgs, _, err := b.imageCRUD.ReadBatch(digests)
	if err != nil {
		return nil, err
	}

	images := make([]*storage.Image, 0, len(msgs))
	for _, m := range msgs {
		images = append(images, m.(*storage.Image))
	}
	return images, nil
}

// UpdateImage updates a image to bolt.
func (b *storeImpl) UpsertImage(image *storage.Image) error {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Upsert, "Image")

	if !b.noUpdateTimestamps {
		image.LastUpdated = protoTypes.TimestampNow()
	}
	return b.imageCRUD.Upsert(image)
}

// DeleteImage deletes an image a all it's data (but maintains the orch digest to registry digest mapping).
func (b *storeImpl) DeleteImage(id string) error {
	defer metrics.SetBadgerOperationDurationTime(time.Now(), ops.Remove, "Image")

	digest := types.NewDigest(id).Digest()
	return b.imageCRUD.Delete(digest)
}

func (b *storeImpl) GetTxnCount() (txNum uint64, err error) {
	return b.imageCRUD.GetTxnCount(), nil
}

func (b *storeImpl) IncTxnCount() error {
	return b.imageCRUD.IncTxnCount()
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
