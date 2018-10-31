package messagecrud

import (
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
)

type messageCrudImpl struct {
	keyFunc   func(proto.Message) []byte
	allocFunc func() proto.Message

	db         *bolt.DB
	bucketName []byte
}

// Read reads and returns a single proto message from bolt.
func (crud *messageCrudImpl) Read(id string) (msg proto.Message, err error) {
	err = crud.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(crud.bucketName)
		bytes := b.Get([]byte(id))
		if len(bytes) == 0 {
			return nil
		}

		msg = crud.allocFunc()
		return proto.Unmarshal(bytes, msg)
	})
	return
}

// ReadBatch reads and returns a list of proto messages for a list of ids in the same order.
func (crud *messageCrudImpl) ReadBatch(ids []string) (msgs []proto.Message, err error) {
	err = crud.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(crud.bucketName)
		for _, id := range ids {
			v := b.Get([]byte(id))
			if v == nil {
				return fmt.Errorf("cannot find value for key: %s", id)
			}

			msg := crud.allocFunc()
			err := proto.Unmarshal(v, msg)
			if err != nil {
				return err
			}
			msgs = append(msgs, msg)
		}
		return nil
	})
	return
}

// ReadAll returns all of the proto messages stored in the bucket.
func (crud *messageCrudImpl) ReadAll() (msgs []proto.Message, err error) {
	// Read all the byte arrays.
	err = crud.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(crud.bucketName)
		return b.ForEach(func(k, v []byte) error {
			msg := crud.allocFunc()
			err := proto.Unmarshal(v, msg)
			if err != nil {
				return err
			}
			msgs = append(msgs, msg)
			return nil
		})
	})
	return
}

// Create creates a new entry in bolt for the input message.
// Returns an error if an entry with a matching id exists.
func (crud *messageCrudImpl) Create(msg proto.Message) (err error) {
	var bytes []byte
	bytes, err = proto.Marshal(msg)
	if err != nil {
		return
	}

	return crud.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(crud.bucketName)
		if b.Get(crud.keyFunc(msg)) != nil {
			return fmt.Errorf("value for key already exists: %s", string(crud.keyFunc(msg)))
		}
		return b.Put(crud.keyFunc(msg), bytes)
	})
}

// Create creates new entries in bolt for the input messages.
// Returns an error if any entry with a matching id already exists.
func (crud *messageCrudImpl) CreateBatch(msgs []proto.Message) (err error) {
	bytes := make(map[string][]byte, len(msgs))
	var v []byte
	for _, msg := range msgs {
		v, err = proto.Marshal(msg)
		if err != nil {
			return err
		}
		bytes[string(crud.keyFunc(msg))] = v
	}

	return crud.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(crud.bucketName)
		for k, v := range bytes {
			if b.Get([]byte(k)) != nil {
				return fmt.Errorf("value for key already exists: %s", k)
			}
			if err = b.Put([]byte(k), v); err != nil {
				return err
			}
		}
		return nil
	})
}

// Update updates a new entry in bolt for the input message.
// Returns an error an entry with the same id does not already exist.
func (crud *messageCrudImpl) Update(msg proto.Message) (err error) {
	var bytes []byte
	bytes, err = proto.Marshal(msg)
	if err != nil {
		return err
	}

	return crud.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(crud.bucketName)
		if b.Get(crud.keyFunc(msg)) == nil {
			return fmt.Errorf("value for key does not exist: %s", string(crud.keyFunc(msg)))
		}
		return b.Put(crud.keyFunc(msg), bytes)
	})
}

// Update updates the entries in bolt for the input messages.
// Returns an error if any input message does not have an existing entry.
func (crud *messageCrudImpl) UpdateBatch(msgs []proto.Message) (err error) {
	bytes := make(map[string][]byte, len(msgs))
	var v []byte
	for _, msg := range msgs {
		v, err = proto.Marshal(msg)
		if err != nil {
			return err
		}
		bytes[string(crud.keyFunc(msg))] = v
	}

	return crud.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(crud.bucketName)
		for k, v := range bytes {
			if b.Get([]byte(k)) == nil {
				return fmt.Errorf("value for key does not exist: %s", k)
			}
			if err = b.Put([]byte(k), v); err != nil {
				return err
			}
		}
		return nil
	})
}

// Upsert upserts the input message into bolt whether or not an entry with the same id already exists.
func (crud *messageCrudImpl) Upsert(msg proto.Message) (err error) {
	var bytes []byte
	bytes, err = proto.Marshal(msg)
	if err != nil {
		return err
	}

	return crud.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(crud.bucketName)
		return b.Put(crud.keyFunc(msg), bytes)
	})
}

// Upsert upserts the input messages into bolt whether or not entries with the same ids already exist.
func (crud *messageCrudImpl) UpsertBatch(msgs []proto.Message) (err error) {
	bytes := make(map[string][]byte, len(msgs))
	var v []byte
	for _, msg := range msgs {
		v, err = proto.Marshal(msg)
		if err != nil {
			return err
		}
		bytes[string(crud.keyFunc(msg))] = v
	}

	return crud.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(crud.bucketName)
		for k, v := range bytes {
			if err = b.Put([]byte(k), v); err != nil {
				return err
			}
		}
		return nil
	})
}

// Delete delete the input message in bolt whether or not an entry with the same id exists.
func (crud *messageCrudImpl) Delete(id string) (err error) {
	return crud.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(crud.bucketName)
		b.Delete([]byte(id))
		return nil
	})
}

// DeleteBatch deletes the messages associated with all of the input ids in bolt.
func (crud *messageCrudImpl) DeleteBatch(ids []string) (err error) {
	return crud.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(crud.bucketName)
		for _, id := range ids {
			b.Delete([]byte(id))
		}
		return nil
	})
}
