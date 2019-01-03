package store

import (
	"fmt"
	"time"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/types"
	ops "github.com/stackrox/rox/pkg/metrics"
)

type storeImpl struct {
	db *bolt.DB
}

// ListImage returns ListImage with given id.
func (b *storeImpl) ListImage(id string) (image *storage.ListImage, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "ListImage")

	digest := types.NewDigest(id).Digest()
	err = b.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(listImageBucket)
		image = new(storage.ListImage)
		val := bucket.Get([]byte(digest))
		if val == nil {
			return nil
		}
		exists = true
		return proto.Unmarshal(val, image)
	})
	return
}

// ListImages returns all ListImages
func (b *storeImpl) ListImages() (images []*storage.ListImage, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "ListImage")

	err = b.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(listImageBucket)
		return b.ForEach(func(k, v []byte) error {
			var image storage.ListImage
			if err := proto.Unmarshal(v, &image); err != nil {
				return err
			}
			images = append(images, &image)
			return nil
		})
	})
	return
}

// GetImages returns all images regardless of request
func (b *storeImpl) GetImages() (images []*storage.Image, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetAll, "Image")

	err = b.db.View(func(tx *bolt.Tx) error {
		images, err = readAllImages(tx)
		return err
	})
	return
}

// CountImages returns all images regardless of request
func (b *storeImpl) CountImages() (count int, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetAll, "Image")

	err = b.db.View(func(tx *bolt.Tx) error {
		count = tx.Bucket(imageBucket).Stats().KeyN
		return nil
	})
	return
}

// GetImage returns image with given id.
func (b *storeImpl) GetImage(id string) (image *storage.Image, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "Image")

	err = b.db.View(func(tx *bolt.Tx) error {
		exists = hasImage(tx, []byte(idForSha(id)))
		if !exists {
			return nil
		}
		image, err = readImage(tx, []byte(idForSha(id)))
		return err
	})
	return
}

// GetImagesBatch returns image with given sha.
func (b *storeImpl) GetImagesBatch(shas []string) (images []*storage.Image, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "Image")

	err = b.db.View(func(tx *bolt.Tx) error {
		for _, sha := range shas {
			image, err := readImage(tx, []byte(idForSha(sha)))
			if err != nil {
				return err
			}
			images = append(images, image)
		}
		return nil
	})
	return
}

// UpdateImage updates a image to bolt.
func (b *storeImpl) UpsertImage(image *storage.Image) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Upsert, "Image")

	return b.db.Update(func(tx *bolt.Tx) error {
		err := writeImage(tx, image)
		if err != nil {
			return err
		}
		return upsertListImage(tx, image)
	})
}

// DeleteImage deletes an image an all it's data (but maintains the orch sha to registry sha mapping).
func (b *storeImpl) DeleteImage(id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "Image")

	return b.db.Update(func(tx *bolt.Tx) error {
		err := deleteImage(tx, []byte(idForSha(id)))
		if err != nil {
			return err
		}
		return deleteListImage(tx, []byte(idForSha(id)))
	})
}

// General helper functions.
////////////////////////////

func idForSha(sha string) string {
	return types.NewDigest(sha).Digest()
}

func convertImageToListImage(i *storage.Image) *storage.ListImage {
	listImage := &storage.ListImage{
		Id:      i.GetId(),
		Name:    i.GetName().GetFullName(),
		Created: i.GetMetadata().GetV1().GetCreated(),
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

// In-Transaction helper functions.
///////////////////////////////////

// readAllImages reads all the images in the DB within a transaction.
func readAllImages(tx *bolt.Tx) (images []*storage.Image, err error) {
	bucket := tx.Bucket(imageBucket)
	err = bucket.ForEach(func(k, v []byte) error {
		image, err := readImage(tx, k)
		if err != nil {
			return err
		}

		images = append(images, image)
		return nil
	})
	return
}

// HasImage returns whether a image exists for the given id.
func hasImage(tx *bolt.Tx, id []byte) bool {
	bucket := tx.Bucket(imageBucket)

	bytes := bucket.Get(id)
	if bytes == nil {
		return false
	}
	return true
}

// readImage reads a image within a transaction.
func readImage(tx *bolt.Tx, id []byte) (image *storage.Image, err error) {
	bucket := tx.Bucket(imageBucket)

	bytes := bucket.Get(id)
	if bytes == nil {
		err = fmt.Errorf("image with id: %s does not exist", id)
		return
	}

	image = new(storage.Image)
	err = proto.Unmarshal(bytes, image)
	return
}

// writeImage writes an image within a transaction.
func writeImage(tx *bolt.Tx, image *storage.Image) (err error) {
	bucket := tx.Bucket(imageBucket)

	id := []byte(idForSha(image.GetId()))

	bytes, err := proto.Marshal(image)
	if err != nil {
		return
	}
	bucket.Put(id, bytes)
	return
}

// deleteImage deletes an image within a transaction.
func deleteImage(tx *bolt.Tx, id []byte) (err error) {
	bucket := tx.Bucket(imageBucket)

	bucket.Delete(id)
	return
}

func upsertListImage(tx *bolt.Tx, image *storage.Image) error {
	bucket := tx.Bucket(listImageBucket)
	listImage := convertImageToListImage(image)
	bytes, err := proto.Marshal(listImage)
	if err != nil {
		return err
	}
	digest := types.NewDigest(image.GetId()).Digest()
	return bucket.Put([]byte(digest), bytes)
}

func deleteListImage(tx *bolt.Tx, id []byte) (err error) {
	bucket := tx.Bucket(listImageBucket)

	bucket.Delete(id)
	return
}
