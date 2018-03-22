package boltdb

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/images"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
)

const imageBucket = "images"

func (b *BoltDB) getImage(sha string, bucket *bolt.Bucket) (image *v1.Image, exists bool, err error) {
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
func (b *BoltDB) GetImage(sha string) (image *v1.Image, exists bool, err error) {
	digest := images.NewDigest(sha).Digest()
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(imageBucket))
		image, exists, err = b.getImage(digest, bucket)
		return err
	})
	return
}

// GetImages returns all images regardless of request
func (b *BoltDB) GetImages(*v1.GetImagesRequest) ([]*v1.Image, error) {
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
func (b *BoltDB) CountImages() (count int, err error) {
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
func (b *BoltDB) AddImage(image *v1.Image) error {
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
		return b.indexer.AddImage(image)
	})
}

// UpdateImage updates a image to bolt
func (b *BoltDB) UpdateImage(image *v1.Image) error {
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
		return b.indexer.AddImage(image)
	})
}

// RemoveImage removes the image from bolt
func (b *BoltDB) RemoveImage(sha string) error {
	digest := images.NewDigest(sha).Digest()
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(imageBucket))
		key := []byte(digest)
		if exists := bucket.Get(key) != nil; !exists {
			return db.ErrNotFound{Type: "Image", ID: string(key)}
		}
		if err := bucket.Delete(key); err != nil {
			return err
		}
		return b.indexer.DeleteImage(digest)
	})
}
