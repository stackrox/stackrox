package boltdb

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
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
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(imageBucket))
		image, exists, err = b.getImage(sha, bucket)
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

// AddImage adds a image to bolt
func (b *BoltDB) AddImage(image *v1.Image) error {
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(imageBucket))
		_, exists, err := b.getImage(image.Sha, bucket)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("Image %v cannot be added because it already exists", image.GetSha())
		}
		bytes, err := proto.Marshal(image)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(image.Sha), bytes)
	})
}

// UpdateImage updates a image to bolt
func (b *BoltDB) UpdateImage(image *v1.Image) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(imageBucket))
		bytes, err := proto.Marshal(image)
		if err != nil {
			return err
		}
		return b.Put([]byte(image.Sha), bytes)
	})
}

// RemoveImage removes the image from bolt
func (b *BoltDB) RemoveImage(sha string) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(imageBucket))
		key := []byte(sha)
		if exists := b.Get(key) != nil; !exists {
			return db.ErrNotFound{Type: "Image", ID: string(key)}
		}
		return b.Delete(key)
	})
}
