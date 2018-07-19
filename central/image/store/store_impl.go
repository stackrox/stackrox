package store

import (
	"fmt"
	"time"

	"bitbucket.org/stack-rox/apollo/central/metrics"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/images"
	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
)

type storeImpl struct {
	*bolt.DB
}

// ListImage returns ListImage with given sha.
func (b *storeImpl) ListImage(sha string) (image *v1.ListImage, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "Get", "ListImage")
	digest := images.NewDigest(sha).Digest()
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(listImageBucket))
		image = new(v1.ListImage)
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
func (b *storeImpl) ListImages() (images []*v1.ListImage, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "GetMany", "ListImage")
	err = b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(listImageBucket))
		return b.ForEach(func(k, v []byte) error {
			var image v1.ListImage
			if err := proto.Unmarshal(v, &image); err != nil {
				return err
			}
			images = append(images, &image)
			return nil
		})
	})
	return
}

func convertImageToListImage(i *v1.Image) *v1.ListImage {
	listImage := &v1.ListImage{
		Sha:     i.GetName().GetSha(),
		Name:    i.GetName().GetFullName(),
		Created: i.GetMetadata().GetCreated(),
	}

	if i.GetScan() != nil {
		listImage.SetComponents = &v1.ListImage_Components{
			Components: int64(len(i.GetScan().GetComponents())),
		}
		var numVulns int64
		var numFixableVulns int64
		for _, c := range i.GetScan().GetComponents() {
			numVulns += int64(len(c.GetVulns()))
			for _, v := range c.GetVulns() {
				if v.FixedBy != "" {
					numFixableVulns++
				}
			}
		}
		listImage.SetCves = &v1.ListImage_Cves{
			Cves: numVulns,
		}
		listImage.SetFixable = &v1.ListImage_FixableCves{
			FixableCves: numFixableVulns,
		}
	}
	return listImage
}

func (b *storeImpl) upsertListImage(tx *bolt.Tx, image *v1.Image) error {
	bucket := tx.Bucket([]byte(listImageBucket))
	listImage := convertImageToListImage(image)
	bytes, err := proto.Marshal(listImage)
	if err != nil {
		return err
	}
	digest := images.NewDigest(image.GetName().GetSha()).Digest()
	return bucket.Put([]byte(digest), bytes)
}

func (b *storeImpl) getImage(sha string, bucket *bolt.Bucket) (image *v1.Image, exists bool, err error) {
	image = new(v1.Image)
	val := bucket.Get([]byte(sha))
	if val == nil {
		return
	}
	exists = true
	err = proto.Unmarshal(val, image)
	return
}

// GetImage returns image with given id.
func (b *storeImpl) GetImage(sha string) (image *v1.Image, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "Get", "Image")
	digest := images.NewDigest(sha).Digest()
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(imageBucket))
		image, exists, err = b.getImage(digest, bucket)
		return err
	})
	return
}

// GetImages returns all images regardless of request
func (b *storeImpl) GetImages() ([]*v1.Image, error) {
	var images []*v1.Image
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(imageBucket))
		return b.ForEach(func(k, v []byte) error {
			var image v1.Image
			if err := proto.Unmarshal(v, &image); err != nil {
				return err
			}
			images = append(images, &image)
			return nil
		})
	})
	return images, err
}

// CountImages returns the number of images.
func (b *storeImpl) CountImages() (count int, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "Count", "Image")
	err = b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(imageBucket))
		return b.ForEach(func(k, v []byte) error {
			count++
			return nil
		})
	})

	return
}

// AddImage adds a image to bolt
func (b *storeImpl) AddImage(image *v1.Image) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "Add", "Image")
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(imageBucket))
		digest := images.NewDigest(image.GetName().GetSha()).Digest()
		_, exists, err := b.getImage(digest, bucket)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("Image %s cannot be added because it already exists", digest)
		}
		bytes, err := proto.Marshal(image)
		if err != nil {
			return err
		}
		if err := bucket.Put([]byte(digest), bytes); err != nil {
			return err
		}
		return b.upsertListImage(tx, image)
	})
}

// UpdateImage updates a image to bolt
func (b *storeImpl) UpdateImage(image *v1.Image) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "Update", "Image")
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(imageBucket))
		bytes, err := proto.Marshal(image)
		if err != nil {
			return err
		}
		digest := images.NewDigest(image.GetName().GetSha()).Digest()
		if err := bucket.Put([]byte(digest), bytes); err != nil {
			return err
		}
		return b.upsertListImage(tx, image)
	})
}
