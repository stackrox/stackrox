package boltdb

import (
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
)

const imageRuleBucket = "image_rules"

// GetImageRules returns all image rules regardless of request
func (b *BoltDB) GetImageRules(*v1.GetImageRulesRequest) ([]*v1.ImageRule, error) {
	var imageRules []*v1.ImageRule
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(imageRuleBucket))
		b.ForEach(func(k, v []byte) error {
			var imageRule v1.ImageRule
			if err := proto.Unmarshal(v, &imageRule); err != nil {
				return err
			}
			imageRules = append(imageRules, &imageRule)
			return nil
		})
		return nil
	})
	return imageRules, err
}

func (b *BoltDB) upsertImageRule(rule *v1.ImageRule) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(imageRuleBucket))
		bytes, err := proto.Marshal(rule)
		if err != nil {
			log.Error(err)
			return err
		}
		err = b.Put([]byte(rule.Name), bytes)
		return err
	})
}

// AddImageRule inserts the image rule
func (b *BoltDB) AddImageRule(rule *v1.ImageRule) error {
	return b.upsertImageRule(rule)
}

// UpdateImageRule inserts the image rule
func (b *BoltDB) UpdateImageRule(rule *v1.ImageRule) error {
	return b.upsertImageRule(rule)
}

// RemoveImageRule removes the image rule from the database
func (b *BoltDB) RemoveImageRule(name string) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(imageRuleBucket))
		err := b.Delete([]byte(name))
		return err
	})
}
