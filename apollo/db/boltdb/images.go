package boltdb

import (
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
)

const imageBucket = "images"

// GetImages returns all images regardless of request
func (b *BoltDB) GetImages(*v1.GetImagesRequest) ([]*v1.Image, error) {
	var images []*v1.Image
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(imageBucket))
		b.ForEach(func(k, v []byte) error {
			var image v1.Image
			if err := proto.Unmarshal(v, &image); err != nil {
				return err
			}
			images = append(images, &image)
			return nil
		})
		return nil
	})
	return images, err
}

func (b *BoltDB) upsertImage(image *v1.Image) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(imageBucket))
		bytes, err := proto.Marshal(image)
		if err != nil {
			log.Error(err)
			return err
		}
		err = b.Put([]byte(image.Sha), bytes)
		return err
	})
}

// AddImage inserts the image into bolt
func (b *BoltDB) AddImage(image *v1.Image) error {
	return b.upsertImage(image)
}

// UpdateImage inserts the image into bolt
func (b *BoltDB) UpdateImage(image *v1.Image) error {
	return b.upsertImage(image)
}

// RemoveImage removes the image from bolt
func (b *BoltDB) RemoveImage(sha string) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(imageBucket))
		err := b.Delete([]byte(sha))
		return err
	})
}
