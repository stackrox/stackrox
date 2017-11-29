package boltdb

import (
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
)

const imagePolicyBucket = "image_policies"

// GetImagePolicies returns all image policies regardless of request.
func (b *BoltDB) GetImagePolicies(*v1.GetImagePoliciesRequest) ([]*v1.ImagePolicy, error) {
	var imagePolicies []*v1.ImagePolicy
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(imagePolicyBucket))
		b.ForEach(func(k, v []byte) error {
			var imagePolicy v1.ImagePolicy
			if err := proto.Unmarshal(v, &imagePolicy); err != nil {
				return err
			}
			imagePolicies = append(imagePolicies, &imagePolicy)
			return nil
		})
		return nil
	})
	return imagePolicies, err
}

func (b *BoltDB) upsertImagePolicy(policy *v1.ImagePolicy) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(imagePolicyBucket))
		bytes, err := proto.Marshal(policy)
		if err != nil {
			log.Error(err)
			return err
		}
		return b.Put([]byte(policy.Name), bytes)
	})
}

// AddImagePolicy inserts the image policy.
func (b *BoltDB) AddImagePolicy(policy *v1.ImagePolicy) error {
	return b.upsertImagePolicy(policy)
}

// UpdateImagePolicy updates the image policy.
func (b *BoltDB) UpdateImagePolicy(policy *v1.ImagePolicy) error {
	return b.upsertImagePolicy(policy)
}

// RemoveImagePolicy removes the image policy from the database.
func (b *BoltDB) RemoveImagePolicy(name string) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(imagePolicyBucket))
		err := b.Delete([]byte(name))
		return err
	})
}
